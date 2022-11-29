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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	k8sstatus "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type mockStatusConfig struct {
	k8sstatus.Config
	statusCheck *bool
}

func (m mockStatusConfig) StatusCheck() *bool { return m.statusCheck }

func (m mockStatusConfig) GetKubeContext() string { return "" }

func (m mockStatusConfig) StatusCheckDeadlineSeconds() int { return 0 }

func (m mockStatusConfig) StatusCheckTolerateFailures() bool { return false }

func (m mockStatusConfig) FastFailStatusCheck() bool { return true }

func (m mockStatusConfig) Muted() config.Muted { return config.Muted{} }

func TestGetMonitor(t *testing.T) {
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
			statusCheck: util.Ptr(true),
		},
		{
			description: "statusCheck parameter set to false",
			statusCheck: util.Ptr(false),
			isNoop:      true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			m := NewMonitor(mockStatusConfig{statusCheck: test.statusCheck}, test.description, label.NewLabeller(false, nil, ""), nil)
			t.CheckDeepEqual(test.isNoop, reflect.Indirect(reflect.ValueOf(m)).Type() == reflect.TypeOf(k8sstatus.NoopMonitor{}))
		})
	}
}
