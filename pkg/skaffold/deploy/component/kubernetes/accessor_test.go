/*
Copyright 2021 The Skaffold Authors

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

package kubernetes

import (
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type mockAccessConfig struct {
	portforward.Config
	opts config.PortForwardOptions
}

func (m mockAccessConfig) Mode() config.RunMode { return "" }

func (m mockAccessConfig) PortForwardOptions() config.PortForwardOptions { return m.opts }

func (m mockAccessConfig) PortForwardResources() []*latest.PortForwardResource { return nil }

func TestGetAccessor(t *testing.T) {
	tests := []struct {
		description string
		enabled     bool
		isNoop      bool
	}{
		{
			description: "unspecified parameter defaults to disabled",
			isNoop:      true,
		},
		{
			description: "portForwardEnabled parameter set to true",
			enabled:     true,
		},
		{
			description: "portForwardEnabled parameter set to false",
			isNoop:      true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			opts := config.PortForwardOptions{}
			if test.enabled {
				opts.Append("1") // default enabled mode
			}
			a := NewAccessor(mockAccessConfig{opts: opts}, test.description, &kubectl.CLI{}, nil, label.NewLabeller(false, nil, ""), nil)
			t.CheckDeepEqual(test.isNoop, reflect.Indirect(reflect.ValueOf(a)).Type() == reflect.TypeOf(access.NoopAccessor{}))
		})
	}
}
