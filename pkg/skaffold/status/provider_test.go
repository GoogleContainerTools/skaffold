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

package status

import (
	"reflect"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestMonitorProvider(t *testing.T) {
	tests := []struct {
		description string
		statusCheck *bool
		isNoop      bool
	}{
		{
			description: "unspecified statusCheck parameter",
		},
		{
			description: "statusCheck parameter set to true",
			statusCheck: util.BoolPtr(true),
		},
		{
			description: "statusCheck parameter set to false",
			statusCheck: util.BoolPtr(false),
			isNoop:      true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			m := NewMonitorProvider(nil).GetKubernetesMonitor(mockConfig{statusCheck: test.statusCheck})
			t.CheckDeepEqual(test.isNoop, reflect.Indirect(reflect.ValueOf(m)).Type() == reflect.TypeOf(NoopMonitor{}))
		})
	}
}

type mockConfig struct {
	status.Config
	statusCheck *bool
}

func (m mockConfig) StatusCheck() *bool { return m.statusCheck }

func (m mockConfig) GetKubeContext() string { return "" }

func (m mockConfig) StatusCheckDeadlineSeconds() int { return 0 }

func (m mockConfig) Muted() config.Muted { return config.Muted{} }
