/*
Copyright 2020 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validator

import (
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

// Recommender makes recommendations based on err in the actionable error
type Recommender interface {
	// Makes one or more recommendations for the ErrorCode in err and updates the err with suggestions
	Make(errCode proto.StatusCode) *proto.Suggestion
}
