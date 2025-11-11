/*
Copyright 2022 The Skaffold Authors

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

package kustomize

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestBuildCommandArgs(t *testing.T) {
	tests := []struct {
		description   string
		buildArgs     []string
		kustomizePath string
		expectedArgs  []string
	}{
		{
			description:   "no BuildArgs, empty KustomizePaths ",
			buildArgs:     []string{},
			kustomizePath: "",
			expectedArgs:  nil,
		},
		{
			description:   "One BuildArg, empty KustomizePaths",
			buildArgs:     []string{"--foo"},
			kustomizePath: "",
			expectedArgs:  []string{"--foo"},
		},
		{
			description:   "no BuildArgs, non-empty KustomizePaths",
			buildArgs:     []string{},
			kustomizePath: "foo",
			expectedArgs:  []string{"foo"},
		},
		{
			description:   "One BuildArg, non-empty KustomizePaths",
			buildArgs:     []string{"--foo"},
			kustomizePath: "bar",
			expectedArgs:  []string{"--foo", "bar"},
		},
		{
			description:   "Multiple BuildArg, empty KustomizePaths",
			buildArgs:     []string{"--foo", "--bar"},
			kustomizePath: "",
			expectedArgs:  []string{"--foo", "--bar"},
		},
		{
			description:   "Multiple BuildArg with spaces, empty KustomizePaths",
			buildArgs:     []string{"--foo bar", "--baz"},
			kustomizePath: "",
			expectedArgs:  []string{"--foo", "bar", "--baz"},
		},
		{
			description:   "Multiple BuildArg with spaces, non-empty KustomizePaths",
			buildArgs:     []string{"--foo bar", "--baz"},
			kustomizePath: "barfoo",
			expectedArgs:  []string{"--foo", "bar", "--baz", "barfoo"},
		},
		{
			description:   "Multiple BuildArg no spaces, non-empty KustomizePaths",
			buildArgs:     []string{"--foo", "bar", "--baz"},
			kustomizePath: "barfoo",
			expectedArgs:  []string{"--foo", "bar", "--baz", "barfoo"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			args := kustomizeBuildArgs(test.buildArgs, test.kustomizePath)
			t.CheckDeepEqual(test.expectedArgs, args)
		})
	}
}

func TestMirror(t *testing.T) {
	tests := []struct {
		description string
		shouldErr   bool
		createFiles map[string]string
		touchFiles  []string
	}{
		{
			description: "mirror configmap and secret generators using env files, regular files, and regular files with keys",
			shouldErr:   false,
			createFiles: map[string]string{
				"kustomization.yaml": `configMapGenerator:
  - name: app-env
    envs:
      - configmaps/app-env/app.env
  - name: app-config
    files:
      - credentials.pub=configmaps/app-config/credentials.local.pub
      - configmaps/app-config/setup.json

secretGenerator:
  - name: app-env-secrets
    envs:
      - secrets/app-env-secrets/secrets.env
  - name: app-config-secrets
    files:
      - credentials.key=secrets/app-config-secrets/credentials.local.key
      - secrets/app-config-secrets/eyesonly.txt
`},
			touchFiles: []string{
				"configmaps/app-env/app.env",
				"configmaps/app-config/credentials.local.pub",
				"configmaps/app-config/setup.json",
				"secrets/app-env-secrets/secrets.env",
				"secrets/app-config-secrets/credentials.local.key",
				"secrets/app-config-secrets/eyesonly.txt",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			// Create the file structure required for the test
			sourceDir := t.NewTempDir()

			for path, contents := range test.createFiles {
				sourceDir.Write(path, contents)
			}

			sourceDir.Touch(test.touchFiles...)

			// Create the instance to test
			mockCfg := render.MockConfig{
				WorkingDir: sourceDir.Root(),
			}

			rc := latest.RenderConfig{
				Generate: latest.Generate{
					Kustomize: &latest.Kustomize{
						Paths: []string{sourceDir.Root()},
					},
				},
			}

			k, err := New(mockCfg, rc, map[string]string{}, "default", "", nil, false)
			t.CheckNoError(err)

			// Test the mirror function on a temporary workspace
			targetDir := t.NewTempDir()
			fs := newTmpFS(targetDir.Root())
			defer fs.Cleanup()

			t_err := k.mirror(sourceDir.Root(), fs)

			// Gather a list of the expected files
			var expectedFiles []string

			sourceFileList, err := sourceDir.List()
			t.CheckNoError(err)

			sourceVolName := filepath.VolumeName(sourceDir.Root())
			if sourceVolName != "" {
				for _, f := range sourceFileList {
					expectedFiles = append(expectedFiles, strings.TrimPrefix(f, sourceVolName))
				}
			} else {
				expectedFiles = sourceFileList
			}

			slices.Sort(expectedFiles)

			// Gather a list of files that have been created by the mirror function
			targetFileList, err := targetDir.List()
			t.CheckNoError(err)

			minPrefixLen := len(targetDir.Root()) + len(sourceDir.Root()) - len(sourceVolName)
			prefixLen := len(targetDir.Root())

			var mirroredFiles []string
			for _, f := range targetFileList {
				if len(f) >= minPrefixLen {
					mirroredFiles = append(mirroredFiles, f[prefixLen:])
				}
			}

			slices.Sort(mirroredFiles)

			// Validate test results
			t.CheckErrorAndDeepEqual(test.shouldErr, t_err, expectedFiles, mirroredFiles)
		})
	}
}
