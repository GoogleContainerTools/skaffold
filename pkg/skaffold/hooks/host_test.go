// +build !windows

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

package hooks

import (
	"bytes"
	"context"
	"testing"

	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRun(t *testing.T) {
	tests := []struct {
		description string
		hook        hostHook
		expected    string
	}{
		{
			description: "linux, darwin host hook",
			hook: hostHook{
				cfg: v1.HostHook{
					OS:      []string{"linux", "darwin"},
					Command: []string{"sh", "-c", "echo FOO=$FOO"},
				},
				env: []string{"FOO=bar"},
			},
			expected: "FOO=bar\n",
		},
		{
			description: "windows host hook",
			hook: hostHook{
				cfg: v1.HostHook{
					OS:      []string{"windows"},
					Command: []string{"cmd.exe", "/C", "echo %FOO%"},
				},
				env: []string{"FOO=bar"},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var buf bytes.Buffer
			err := test.hook.run(context.Background(), &buf)
			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected, buf.String())
		})
	}
}
