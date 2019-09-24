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

package resource

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPod_CheckStatus(t *testing.T) {
	tests := []struct {
		description string
		expectedErr string
		complete    bool
	}{
		{
			description: "not implemented",
			expectedErr: "not yet implemented",
			complete:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			p := NewPod("dep", "test")
			p.CheckStatus(context.Background(), &runcontext.RunContext{})
			t.CheckDeepEqual(test.complete, p.IsStatusCheckComplete())
			t.CheckErrorContains(test.expectedErr, p.Status().Error())
		})
	}
}

func TestPod_Deadline(t *testing.T) {
	p := NewPod("dep", "test")
	testutil.CheckDeepEqual(t, 2*time.Minute, p.Deadline())
}
