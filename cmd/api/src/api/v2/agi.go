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

package v2

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/specterops/bloodhound/src/api"
	"github.com/specterops/bloodhound/src/ctx"
	"github.com/specterops/bloodhound/src/model"
	"github.com/specterops/bloodhound/src/utils"
	"github.com/gorilla/mux"
	"github.com/specterops/bloodhound/analysis"
	"github.com/specterops/bloodhound/dawgs/graph"
	"github.com/specterops/bloodhound/graphschema/ad"
	"github.com/specterops/bloodhound/graphschema/azure"
	"github.com/specterops/bloodhound/graphschema/common"
	"github.com/specterops/bloodhound/headers"
	"github.com/specterops/bloodhound/slices"
)

// CreateAssetGroupRequest holds data required to create an asset group
type CreateAssetGroupRequest struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

// AuditData returns an AuditData data structure corresponding to the CreateAssetGroupRequest
func (s CreateAssetGroupRequest) AuditData() model.AuditData {
	return model.AuditData{
		"asset_group_name": s.Name,
		"asset_group_tag":  s.Tag,
	}
}

// UpdateAssetGroupRequest holds the data required for updating an asset group
type UpdateAssetGroupRequest struct {
	Name string `json:"name"`
}

// AuditData returns an AuditData data structure corresponding to the UpdateAssetGroupRequest
func (s UpdateAssetGroupRequest) AuditData() model.AuditData {
	return model.AuditData{
		"name": s.Name,
	}
}

// ListAssetGroupsResponse holds the data returned to a list asset groups request
type ListAssetGroupsResponse struct {
	AssetGroups model.AssetGroups `json:"asset_groups"`
}

type AssetGroupCollectionsResponse struct {
	Data []any `json:"data"`
}

func (s Resources) ListAssetGroups(response http.ResponseWriter, request *http.Request) {
	var (
		order         []string
		assetGroups   model.AssetGroups
		sortByColumns = request.URL.Query()[api.QueryParameterSortBy]
	)

	for _, column := range sortByColumns {
		var descending bool
		if string(column[0]) == "-" {
			descending = true
			column = column[1:]
		}

		if !assetGroups.IsSortable(column) {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsNotSortable, request), response)
			return
		}

		if descending {
			order = append(order, column+" desc")
		} else {
			order = append(order, column)
		}

	}

	queryParameterFilterParser := model.NewQueryParameterFilterParser()
	if queryFilters, err := queryParameterFilterParser.ParseQueryParameterFilters(request); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsBadQueryParameterFilters, request), response)
		return
	} else {
		for name, filters := range queryFilters {
			if validPredicates, err := assetGroups.GetValidFilterPredicatesAsStrings(name); err != nil {
				api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, fmt.Sprintf("%s: %s", api.ErrorResponseDetailsColumnNotFilterable, name), request), response)
				return
			} else {
				for i, filter := range filters {
					if !slices.Contains(validPredicates, string(filter.Operator)) {
						api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, fmt.Sprintf("%s: %s %s", api.ErrorResponseDetailsFilterPredicateNotSupported, filter.Name, filter.Operator), request), response)
						return
					}

					queryFilters[name][i].IsStringData = assetGroups.IsString(filter.Name)
				}
			}
		}

		// ignoring the error here as this would've failed at ParseQueryParameterFilters before getting here
		if sqlFilter, err := queryFilters.BuildSQLFilter(); err != nil {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, "error building SQL for filter", request), response)
			return
		} else if assetGroups, err := s.DB.GetAllAssetGroups(strings.Join(order, ", "), sqlFilter); err != nil {
			api.HandleDatabaseError(request, response, err)
		} else {
			api.WriteBasicResponse(request.Context(), ListAssetGroupsResponse{AssetGroups: assetGroups}, http.StatusOK, response)
		}
	}
}

func (s Resources) GetAssetGroup(response http.ResponseWriter, request *http.Request) {
	var (
		pathVars        = mux.Vars(request)
		rawAssetGroupID = pathVars[api.URIPathVariableAssetGroupID]
	)

	if assetGroupID, err := strconv.Atoi(rawAssetGroupID); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsIDMalformed, request), response)
	} else if assetGroup, err := s.DB.GetAssetGroup(int32(assetGroupID)); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else {
		api.WriteBasicResponse(request.Context(), assetGroup, http.StatusOK, response)
	}
}

func (s Resources) GetAssetGroupCustomMemberCount(response http.ResponseWriter, request *http.Request) {
	var (
		pathVars        = mux.Vars(request)
		rawAssetGroupID = pathVars[api.URIPathVariableAssetGroupID]
	)

	if assetGroupID, err := strconv.Atoi(rawAssetGroupID); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsIDMalformed, request), response)
	} else if assetGroup, err := s.DB.GetAssetGroup(int32(assetGroupID)); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else {
		customSelectorCount := 0
		for _, selector := range assetGroup.Selectors {
			if !selector.SystemSelector {
				customSelectorCount++
			}
		}

		api.WriteBasicResponse(request.Context(), map[string]int{"custom_member_count": customSelectorCount}, http.StatusOK, response)
	}
}

func (s Resources) UpdateAssetGroup(response http.ResponseWriter, request *http.Request) {
	var (
		pathVars                = mux.Vars(request)
		rawAssetGroupID         = pathVars[api.URIPathVariableAssetGroupID]
		updateAssetGroupRequest UpdateAssetGroupRequest
	)

	if assetGroupID, err := strconv.Atoi(rawAssetGroupID); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsIDMalformed, request), response)
	} else if err := api.ReadJSONRequestPayloadLimited(&updateAssetGroupRequest, request); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, err.Error(), request), response)
	} else if err := s.DB.AppendAuditLog(*ctx.FromRequest(request), "UpdateAssetGroup", updateAssetGroupRequest); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else if assetGroup, err := s.DB.GetAssetGroup(int32(assetGroupID)); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else {
		assetGroup.Name = updateAssetGroupRequest.Name

		if err := s.DB.UpdateAssetGroup(assetGroup); err != nil {
			api.HandleDatabaseError(request, response, err)
		} else {
			api.WriteBasicResponse(request.Context(), assetGroup, http.StatusOK, response)
		}
	}
}

func (s Resources) CreateAssetGroup(response http.ResponseWriter, request *http.Request) {
	var createRequest CreateAssetGroupRequest

	if err := api.ReadJSONRequestPayloadLimited(&createRequest, request); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, err.Error(), request), response)
	} else if err := s.DB.AppendAuditLog(*ctx.FromRequest(request), "CreateAssetGroup", createRequest); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else if newAssetGroup, err := s.DB.CreateAssetGroup(createRequest.Name, createRequest.Tag, false); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else {
		assetGroupURL := *ctx.Get(request.Context()).Host
		assetGroupURL.Path = fmt.Sprintf("/api/v2/asset-groups/%d", newAssetGroup.ID)
		response.Header().Set(headers.Location.String(), assetGroupURL.String())

		api.WriteBasicResponse(request.Context(), newAssetGroup, http.StatusCreated, response)
	}
}

func (s Resources) DeleteAssetGroup(response http.ResponseWriter, request *http.Request) {
	var (
		pathVars        = mux.Vars(request)
		rawAssetGroupID = pathVars[api.URIPathVariableAssetGroupID]
	)

	if assetGroupID, err := strconv.Atoi(rawAssetGroupID); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsIDMalformed, request), response)
	} else if assetGroup, err := s.DB.GetAssetGroup(int32(assetGroupID)); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else if assetGroup.SystemGroup {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusConflict, "Cannot delete a system defined asset group.", request), response)
	} else if err := s.DB.AppendAuditLog(*ctx.FromRequest(request), "DeleteAssetGroup", assetGroup); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else if err := s.DB.DeleteAssetGroup(assetGroup); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else {
		response.WriteHeader(http.StatusOK)
	}
}

func (s Resources) DeleteAssetGroupSelector(response http.ResponseWriter, request *http.Request) {
	var (
		pathVars                = mux.Vars(request)
		rawAssetGroupID         = pathVars[api.URIPathVariableAssetGroupID]
		rawAssetGroupSelectorID = pathVars[api.URIPathVariableAssetGroupSelectorID]
	)

	if assetGroupID, err := strconv.Atoi(rawAssetGroupID); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsIDMalformed, request), response)
	} else if _, err := s.DB.GetAssetGroup(int32(assetGroupID)); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else if assetGroupSelectorID, err := strconv.Atoi(rawAssetGroupSelectorID); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsIDMalformed, request), response)
	} else if assetGroupSelector, err := s.DB.GetAssetGroupSelector(int32(assetGroupSelectorID)); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else if assetGroupSelector.SystemSelector {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusConflict, "Cannot delete a system defined asset group selector.", request), response)
	} else if err := s.DB.AppendAuditLog(*ctx.FromRequest(request), "DeleteAssetGroupSelector", assetGroupSelector); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else if err := s.DB.DeleteAssetGroupSelector(assetGroupSelector); err != nil {
		api.HandleDatabaseError(request, response, err)
	} else {
		response.WriteHeader(http.StatusOK)
	}
}

func (s Resources) ListAssetGroupCollections(response http.ResponseWriter, request *http.Request) {
	var (
		rawAssetGroupID = mux.Vars(request)[api.URIPathVariableAssetGroupID]

		assetGroupID int
		assetGroup   model.AssetGroup

		order                 []string
		assetGroupCollections model.AssetGroupCollections
		sortByColumns         = request.URL.Query()[api.QueryParameterSortBy]
	)

	for _, column := range sortByColumns {
		var descending bool
		if string(column[0]) == "-" {
			descending = true
			column = column[1:]
		}

		if !assetGroupCollections.IsSortable(column) {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsNotSortable, request), response)
			return
		}

		if descending {
			order = append(order, column+" desc")
		} else {
			order = append(order, column)
		}

	}

	queryParameterFilterParser := model.NewQueryParameterFilterParser()
	if queryFilters, err := queryParameterFilterParser.ParseQueryParameterFilters(request); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsBadQueryParameterFilters, request), response)
		return
	} else {
		for name, filters := range queryFilters {
			if validPredicates, err := assetGroupCollections.GetValidFilterPredicatesAsStrings(name); err != nil {
				api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, fmt.Sprintf("%s: %s", api.ErrorResponseDetailsColumnNotFilterable, name), request), response)
				return
			} else {
				for _, filter := range filters {
					if !slices.Contains(validPredicates, string(filter.Operator)) {
						api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, fmt.Sprintf("%s: %s %s", api.ErrorResponseDetailsFilterPredicateNotSupported, filter.Name, filter.Operator), request), response)
						return
					}
				}
			}
		}

		// ignoring the error here as this would've failed at ParseQueryParameterFilters before getting here
		if sqlFilter, err := queryFilters.BuildSQLFilter(); err != nil {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, "error building SQL for filter", request), response)
			return
		} else if assetGroupID, err = strconv.Atoi(rawAssetGroupID); err != nil {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsIDMalformed, request), response)
		} else if assetGroup, err = s.DB.GetAssetGroup(int32(assetGroupID)); err != nil {
			api.HandleDatabaseError(request, response, err)
		} else if collections, err := s.DB.GetAssetGroupCollections(assetGroup.ID, strings.Join(order, ", "), sqlFilter); err != nil {
			api.HandleDatabaseError(request, response, err)
		} else {
			data := make([]any, 0)
			for _, dataElement := range collections {
				data = append(data, dataElement)
			}

			api.WriteBasicResponse(request.Context(), data, http.StatusOK, response)
		}
	}
}

// getLatestQueryParameter parses the "latest" value
func getLatestQueryParameter(query url.Values) (bool, error) {
	keys, wantsLatest := query["latest"]

	if len(keys) > 0 && len(keys[0]) > 0 {
		return false, fmt.Errorf(api.ErrorResponseDetailsLatestMalformed)
	}

	return wantsLatest, nil
}

func (s Resources) ListAssetGroupMembers(response http.ResponseWriter, request *http.Request) {
	var (
		pathVars        = mux.Vars(request)
		rawAssetGroupID = pathVars[api.URIPathVariableAssetGroupID]
		agMembers       api.AssetGroupMembers
		queryParams     = request.URL.Query()
		sortByColumns   = queryParams[api.QueryParameterSortBy]
	)

	queryParameterFilterParser := model.NewQueryParameterFilterParser()
	if queryFilters, err := queryParameterFilterParser.ParseQueryParameterFilters(request); err != nil {
		api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsBadQueryParameterFilters, request), response)
		return
	} else {
		for name, filters := range queryFilters {
			if validPredicates, err := agMembers.GetValidFilterPredicatesAsStrings(name); err != nil {
				api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, fmt.Sprintf("%s: %s", api.ErrorResponseDetailsColumnNotFilterable, name), request), response)
				return
			} else {
				for _, filter := range filters {
					if !slices.Contains(validPredicates, string(filter.Operator)) {
						api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, fmt.Sprintf("%s: %s %s", api.ErrorResponseDetailsFilterPredicateNotSupported, filter.Name, filter.Operator), request), response)
						return
					}
				}
			}
		}

		if assetGroupID, err := strconv.Atoi(rawAssetGroupID); err != nil {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, api.ErrorResponseDetailsIDMalformed, request), response)
		} else if assetGroup, err := s.DB.GetAssetGroup(int32(assetGroupID)); err != nil {
			api.HandleDatabaseError(request, response, err)
		} else if assetGroupNodes, err := s.GraphQuery.GetAssetGroupNodes(request.Context(), assetGroup.Tag); err != nil {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Graph error fetching nodes for asset group ID %v: %v", assetGroup.ID, err), request), response)
		} else if agMembers, err = parseAGMembersFromNodes(assetGroupNodes, assetGroup.Selectors, int(assetGroup.ID)).SortBy(sortByColumns); err != nil {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, err.Error(), request), response)
		} else if agMembers, err = agMembers.Filter(queryFilters); err != nil {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusInternalServerError, fmt.Sprintf("error filtering asset group members: %v", err), request), response)
		} else if skip, err := ParseSkipQueryParameter(queryParams, 0); err != nil {
			api.WriteErrorResponse(request.Context(), ErrBadQueryParameter(request, model.PaginationQueryParameterSkip, err), response)
		} else if skip > len(agMembers) {
			api.WriteErrorResponse(request.Context(), api.BuildErrorResponse(http.StatusBadRequest, fmt.Sprintf(utils.ErrorInvalidSkip, "value must be less than total count"), request), response)
		} else if limit, err := ParseLimitQueryParameter(queryParams, 100); err != nil {
			api.WriteErrorResponse(request.Context(), ErrBadQueryParameter(request, model.PaginationQueryParameterLimit, err), response)
		} else {
			endIndex := len(agMembers)
			if skip+limit < endIndex {
				endIndex = skip + limit
			}
			api.WriteResponseWrapperWithPagination(request.Context(), api.ListAssetGroupMembersResponse{Members: agMembers[skip:endIndex]}, limit, skip, len(agMembers), http.StatusOK, response)
		}
	}
}

func parseAGMembersFromNodes(nodes graph.NodeSet, selectors model.AssetGroupSelectors, assetGroupID int) api.AssetGroupMembers {
	agMembers := api.AssetGroupMembers{}
	for _, node := range nodes {
		isCustomMember := false
		// a member is custom if at least one selector exists for that object ID
		for _, agSelector := range selectors {
			if agSelector.Selector == node.Properties.Map[common.ObjectID.String()].(string) {
				isCustomMember = true
			}
		}

		agMember := api.AssetGroupMember{
			AssetGroupID: assetGroupID,
			ObjectID:     node.Properties.Map[common.ObjectID.String()].(string),
			PrimaryKind:  analysis.GetNodeKindDisplayLabel(node),
			Kinds:        node.Kinds.Strings(),
			Name:         node.Properties.Map[common.Name.String()].(string),
			CustomMember: isCustomMember,
		}

		if tenantID := node.Properties.Map[azure.TenantID.String()]; tenantID != nil {
			agMember.EnvironmentID = tenantID.(string)
			agMember.EnvironmentKind = azure.Tenant.String()
		} else {
			agMember.EnvironmentID = node.Properties.Map[ad.DomainSID.String()].(string)
			agMember.EnvironmentKind = ad.Domain.String()
		}
		agMembers = append(agMembers, agMember)
	}
	return agMembers
}
