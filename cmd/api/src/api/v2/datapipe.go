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
	"net/http"

	"github.com/specterops/bloodhound/src/api"
	"github.com/specterops/bloodhound/log"
)

func (s Resources) GetDatapipeStatus(response http.ResponseWriter, request *http.Request) {
	api.WriteBasicResponse(request.Context(), s.TaskNotifier.GetStatus(), http.StatusOK, response)
}

func (s Resources) RequestAnalysis(response http.ResponseWriter, _ *http.Request) {
	defer log.Measure(log.LevelDebug, "Requesting analysis")()

	s.TaskNotifier.RequestEnrichment()

	response.WriteHeader(http.StatusAccepted)
}
