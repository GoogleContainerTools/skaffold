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

package ui

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewProgressGroup(t *testing.T) {
	tests := []struct {
		description string
	}{
		{
			description: "create a new progress group",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			NewProgressGroup()

			t.CheckTrue(current.BarCount() == 0)
		})
	}
}

func TestAddNewSpinner(t *testing.T) {
	tests := []struct {
		description string
		numBars     int
	}{
		{
			description: "Add one spinner",
			numBars:     1,
		},
		{
			description: "Add multiple spinners",
			numBars:     3,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			NewProgressGroup()
			for i := 0; i < test.numBars; i++ {
				spin := AddNewSpinner("", fmt.Sprintf("bar-%d", i), nil)
				spin.Increment()
			}

			t.CheckTrue(current.BarCount() == test.numBars)
			Wait()
		})
	}
}
