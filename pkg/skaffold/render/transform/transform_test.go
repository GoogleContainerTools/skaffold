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
package transform

import (
	"testing"

	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewTransformer(t *testing.T) {
	tests := []struct {
		description string
		config      []latestV2.Transformer
	}{
		{
			description: "no transform",
			config:      []latestV2.Transformer{},
		},
		{
			description: "set-labels",
			config: []latestV2.Transformer{
				{Name: "set-annotations", ConfigMap: []string{"owner:skaffold-test"}},
				{Name: "set-labels", ConfigMap: []string{"owner:skaffold-test"}},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := NewTransformer(test.config)
			t.CheckNoError(err)
		})
	}
}

func TestNewValidator_Error(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		_, err := NewTransformer([]latestV2.Transformer{
			{Name: "bad-transformer"},
		})
		t.CheckErrorContains(`unsupported transformer "bad-transformer". please only use the`, err)
	})
}
