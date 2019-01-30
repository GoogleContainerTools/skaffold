/*
Copyright 2018 The Skaffold Authors

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

package docker

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
)

type testAuthHelper struct {
	getAuthConfigErr     error
	getAllAuthConfigsErr error
}

var gcrAuthConfig = types.AuthConfig{
	Username:      "bob",
	Password:      "saget",
	ServerAddress: "https://gcr.io",
}

func (t testAuthHelper) GetAuthConfig(string) (types.AuthConfig, error) {
	return gcrAuthConfig, t.getAuthConfigErr
}

func (t testAuthHelper) GetAllAuthConfigs() (map[string]types.AuthConfig, error) {
	return map[string]types.AuthConfig{
		"gcr.io": gcrAuthConfig,
	}, t.getAllAuthConfigsErr
}

func TestGetEncodedRegistryAuth(t *testing.T) {
	var tests = []struct {
		description string
		image       string
		authType    AuthConfigHelper
		expected    string
		shouldErr   bool
	}{
		{
			description: "encode successful",
			image:       "gcr.io/skaffold",
			authType:    testAuthHelper{},
			expected:    "eyJ1c2VybmFtZSI6ImJvYiIsInBhc3N3b3JkIjoic2FnZXQiLCJzZXJ2ZXJhZGRyZXNzIjoiaHR0cHM6Ly9nY3IuaW8ifQ==",
		},
		{
			description: "bad registry name",
			image:       ".",
			authType:    testAuthHelper{},
			shouldErr:   true,
		},
		{
			description: "bad registry name",
			image:       "gcr.io/skaffold",
			authType:    testAuthHelper{getAuthConfigErr: fmt.Errorf("")},
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			defer func(h AuthConfigHelper) { DefaultAuthHelper = h }(DefaultAuthHelper)
			DefaultAuthHelper = test.authType

			l := &localDaemon{}
			out, err := l.encodedRegistryAuth(context.Background(), test.authType, test.image)

			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, out)
		})
	}
}
