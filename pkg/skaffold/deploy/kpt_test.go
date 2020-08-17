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

package deploy

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestKpt_Deploy(t *testing.T) {
	tests := []struct {
		description string
		expected    []string
		shouldErr   bool
	}{
		{
			description: "nil",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			k := NewKptDeployer(&runcontext.RunContext{}, nil)
			res, err := k.Deploy(nil, nil, nil)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, res)
		})
	}
}

func TestKpt_Dependencies(t *testing.T) {
	tests := []struct {
		description string
		expected    []string
		shouldErr   bool
	}{
		{
			description: "nil",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			k := NewKptDeployer(&runcontext.RunContext{}, nil)
			res, err := k.Dependencies()
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, res)
		})
	}
}

func TestKpt_Cleanup(t *testing.T) {
	tests := []struct {
		description string
		shouldErr   bool
	}{
		{
			description: "nil",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			k := NewKptDeployer(&runcontext.RunContext{}, nil)
			err := k.Cleanup(nil, nil)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKpt_Render(t *testing.T) {
	tests := []struct {
		description string
		shouldErr   bool
	}{
		{},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			k := NewKptDeployer(&runcontext.RunContext{}, nil)
			err := k.Render(nil, nil, nil, false, "")
			t.CheckError(test.shouldErr, err)
		})
	}
}
