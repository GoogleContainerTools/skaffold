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

package v2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRunContext_UpdateNamespaces(t *testing.T) {
	tests := []struct {
		description   string
		oldNamespaces []string
		newNamespaces []string
		expected      []string
	}{
		{
			description:   "update namespace when not present in runContext",
			oldNamespaces: []string{"test"},
			newNamespaces: []string{"another"},
			expected:      []string{"another", "test"},
		},
		{
			description:   "update namespace with duplicates should not return duplicate",
			oldNamespaces: []string{"test", "foo"},
			newNamespaces: []string{"another", "foo", "another"},
			expected:      []string{"another", "foo", "test"},
		},
		{
			description:   "update namespaces when namespaces is empty",
			oldNamespaces: []string{"test", "foo"},
			newNamespaces: []string{},
			expected:      []string{"test", "foo"},
		},
		{
			description:   "update namespaces when runcontext namespaces is empty",
			oldNamespaces: []string{},
			newNamespaces: []string{"test", "another"},
			expected:      []string{"another", "test"},
		},
		{
			description:   "update namespaces when both namespaces and runcontext namespaces is empty",
			oldNamespaces: []string{},
			newNamespaces: []string{},
			expected:      []string{},
		},
		{
			description:   "update namespace when runcontext namespace has an empty string",
			oldNamespaces: []string{""},
			newNamespaces: []string{"another"},
			expected:      []string{"another"},
		},
		{
			description:   "update namespace when namespace is empty string",
			oldNamespaces: []string{"test"},
			newNamespaces: []string{""},
			expected:      []string{"test"},
		},
		{
			description:   "update namespace when namespace is empty string and runContext is empty",
			oldNamespaces: []string{},
			newNamespaces: []string{""},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			runCtx := &RunContext{
				Namespaces: test.oldNamespaces,
			}

			runCtx.UpdateNamespaces(test.newNamespaces)

			t.CheckDeepEqual(test.expected, runCtx.Namespaces)
		})
	}
}

func TestGetHydrationDir_Default(t *testing.T) {
	testutil.Run(t, "default to <WORKDIR>/.kpt-pipeline", func(t *testutil.T) {
		tmpDir := t.NewTempDir()
		tmpDir.Chdir()
		runCtx := &RunContext{
			Opts:       config.SkaffoldOptions{HydrationDir: constants.DefaultHydrationDir, AssumeYes: true},
			WorkingDir: tmpDir.Root(),
		}
		actual, err := runCtx.GetHydrationDir()
		t.CheckNoError(err)
		t.CheckDeepEqual(filepath.Join(tmpDir.Root(), ".kpt-pipeline"), actual)
	})
}

func TestGetHydrationDir_CustomHydrationDir(t *testing.T) {
	testutil.Run(t, "--hydration-dir flag is given", func(t *testutil.T) {
		tmpDir := t.NewTempDir()
		tmpDir.Chdir()
		expected := filepath.Join(tmpDir.Root(), "test-hydration")
		runCtx := &RunContext{
			Opts: config.SkaffoldOptions{HydrationDir: expected, AssumeYes: true},
		}
		actual, err := runCtx.GetHydrationDir()
		t.CheckNoError(err)
		t.CheckDeepEqual(expected, actual)
		_, err = os.Stat(actual)
		t.CheckFalse(os.IsNotExist(err))
	})
}
