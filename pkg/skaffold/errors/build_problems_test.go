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

package errors

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestMakeAuthSuggestionsForRepo(t *testing.T) {
	testutil.CheckDeepEqual(t, &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_DOCKER_AUTH_CONFIGURE,
		Action:         "try `docker login`",
	}, makeAuthSuggestionsForRepo(""))
	testutil.CheckDeepEqual(t, &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_GCLOUD_DOCKER_AUTH_CONFIGURE,
		Action:         "try `gcloud auth configure-docker`",
	}, makeAuthSuggestionsForRepo("gcr.io/test"))
	testutil.CheckDeepEqual(t, &proto.Suggestion{
		SuggestionCode: proto.SuggestionCode_GCLOUD_DOCKER_AUTH_CONFIGURE,
		Action:         "try `gcloud auth configure-docker`",
	}, makeAuthSuggestionsForRepo("eu.gcr.io/test"))
}
