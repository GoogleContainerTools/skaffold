/*
Copyright 2019 The Skaffold Authors

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

package recommender

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCustomRecos(t *testing.T) {
	tests := []struct {
		description   string
		deployContext map[string]string
		rulesJSON     []byte
		errCode       proto.StatusCode
		expected      proto.Suggestion
	}{
		{
			description: "test simple rule without any context prefixes",
			rulesJSON: []byte(`{
"rules": [{
     "errorCode": 401,
      "suggestionCode": 401,
      "suggestion": "Try deleting stale pods", 
      "contextPrefixMatches": {}
}]
}`),
			errCode: proto.StatusCode_STATUSCHECK_NODE_DISK_PRESSURE,
			expected: proto.Suggestion{
				SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_DISK_PRESSURE,
				Action:         "Try deleting stale pods",
			},
		},
		{
			description: "test simple rule with a context prefixes",
			rulesJSON: []byte(`{
	"rules": [{
			"errorCode": 401,
			"suggestionCode": 401,
			"suggestion": "Try running minikube status",
			"contextPrefixMatches": {
				"clusterName": "minikube"
			}
		},
		{
			"errorCode": 401,
			"suggestionCode": 401,
			"suggestion": "Try running gcloud",
			"contextPrefixMatches": {
				"clusterName": "gke_"
			}
		}
	]
}`),
			errCode: proto.StatusCode_STATUSCHECK_NODE_DISK_PRESSURE,
			deployContext: map[string]string{
				"clusterName": "gke_test",
			},
			expected: proto.Suggestion{
				SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_DISK_PRESSURE,
				Action:         "Try running gcloud",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			mDownload := func(_ string) ([]byte, error) {
				return test.rulesJSON, nil
			}
			t.Override(&downloadRules, mDownload)

			r, err := NewCustom("testHtpp.json", test.deployContext)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, r.Make(test.errCode))
		})
	}
}

func TestCustomRecosInt(t *testing.T) {
	tests := []struct {
		description   string
		deployContext map[string]string
		errCode       proto.StatusCode
		expected      proto.Suggestion
	}{
		{
			description: "test simple rule without any context prefixes",
			errCode:     proto.StatusCode_STATUSCHECK_NODE_DISK_PRESSURE,
			deployContext: map[string]string{
				"clusterName": "minikube",
			},
			expected: proto.Suggestion{
				SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_DISK_PRESSURE,
				Action:         "Try running minikube status",
			},
		},
		{
			description: "test simple rule with a context prefixes",
			errCode:     proto.StatusCode_STATUSCHECK_NODE_DISK_PRESSURE,
			deployContext: map[string]string{
				"clusterName": "gke_test",
			},
			expected: proto.Suggestion{
				SuggestionCode: proto.SuggestionCode_ADDRESS_NODE_DISK_PRESSURE,
				Action:         "Try running gcloud",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			r, err := NewCustom(DiagDefaultRules, test.deployContext)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, r.Make(test.errCode))
		})
	}
}
