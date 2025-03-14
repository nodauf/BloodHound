// Copyright 2023 Specter Ops, Inc.
// 
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// 
// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"regexp"

	"github.com/specterops/bloodhound/src/model"
	"gorm.io/gorm"
	"github.com/specterops/bloodhound/log"
)

const (
	tierZeroAssetGroupName = "Admin Tier Zero"
	tierZeroAssetGroupTag  = "admin_tier_0"
)

var (
	selectorRegexes = []*regexp.Regexp{
		regexp.MustCompile(`match\s+\(t\)\s+WHERE\s+\(.+\)\s+AND\s+t\.objectid="([^"]+)"`),
		regexp.MustCompile(`match\s+\([^{]+\{objectid:\s+"([^"]+)"\}\)`),
	}
)

func SelectorToObjectID(rawSelector string) string {
	for _, selectorRegex := range selectorRegexes {
		if matches := selectorRegex.FindStringSubmatch(rawSelector); len(matches) == 2 {
			return matches[1]
		}
	}

	return rawSelector
}

func (s *Migrator) updateAssetGroups() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var systemAssetGroups model.AssetGroups

		// Lookup system asset groups
		if result := tx.Where("system_group = true").Find(&systemAssetGroups); result.Error != nil {
			return result.Error
		}

		// Create the tier zero asset group if it doesn't already exist
		if _, hasTierZero := systemAssetGroups.FindByName(tierZeroAssetGroupName); !hasTierZero {
			log.Infof("Missing the default Admin Tier Zero asset group. Creating it now.")

			newTierZeroAG := model.AssetGroup{
				Name:        tierZeroAssetGroupName,
				Tag:         tierZeroAssetGroupTag,
				SystemGroup: true,
			}

			if result := tx.Create(&newTierZeroAG); result.Error != nil {
				return result.Error
			}
		}

		// Load the AG selectors to migrate the selectors away from cypher
		for _, assetGroup := range systemAssetGroups {
			var selectors model.AssetGroupSelectors

			if result := tx.Where("asset_group_id = ?", assetGroup.ID).Find(&selectors); result.Error != nil {
				return result.Error
			}

			for _, selector := range selectors {
				oldSelector := selector.Selector

				if rewrittenSelector := SelectorToObjectID(oldSelector); rewrittenSelector != oldSelector {
					selector.Selector = rewrittenSelector
					tx.Save(selector)
				}
			}
		}

		return nil
	})
}
