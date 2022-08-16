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
package validate

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewValidator(t *testing.T) {
	tests := []struct {
		description string
		config      []latest.Validator
	}{
		{
			description: "no validation",
			config:      []latest.Validator{},
		},
		{
			description: "kubeval validator",
			config: []latest.Validator{
				{Name: "kubeval"},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := NewValidator(test.config)
			t.CheckNoError(err)
		})
	}
}

func TestNewValidator_Error(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		_, err := NewValidator([]latest.Validator{
			{Name: "bad-validator"},
		})
		t.CheckContains(`unsupported validator "bad-validator". please only use the`, err.Error())
	})
}
