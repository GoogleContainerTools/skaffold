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

package cmd

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"

	cfg "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestFlagsToConfigVersion(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig config.Config
		initResult     error
		shouldErr      bool
	}{
		{
			name: "error + default values of flags mapped to config values",
			args: []string{
				"init",
			},
			initResult: errors.New("test error"),
			shouldErr:  true,
			expectedConfig: config.Config{
				ComposeFile:            "",
				CliArtifacts:           nil,
				CliKubernetesManifests: nil,
				SkipBuild:              false,
				SkipDeploy:             false,
				Force:                  false,
				Analyze:                false,
				EnableJibInit:          false,
				EnableJibGradleInit:    false,
				EnableBuildpacksInit:   false,
				EnableNewInitFormat:    false,
				BuildpacksBuilder:      "gcr.io/buildpacks/builder:v1",
				Opts:                   opts,
				MaxFileSize:            maxFileSize,
			},
		},

		{
			name: "no error + non-default values for flags mapped to config values",
			args: []string{
				"init",
				"--compose-file=a-compose-file",
				"--artifact", "a1=b1",
				"-a", "a2=b2",
				"--kubernetes-manifest", "m1",
				"--kubernetes-manifest", "m2",
				"--skip-build",
				"--skip-deploy",
				"--force",
				"--analyze",
				"--XXenableJibInit",
				"--XXenableJibGradleInit",
				"--XXenableBuildpacksInit",
				"--XXenableNewInitFormat",
				"--XXdefaultBuildpacksBuilder", "buildpacks/builder",
			},
			expectedConfig: config.Config{
				ComposeFile:            "a-compose-file",
				CliArtifacts:           []string{"a1=b1", "a2=b2"},
				CliKubernetesManifests: []string{"m1", "m2"},
				SkipBuild:              true,
				SkipDeploy:             true,
				Force:                  true,
				Analyze:                true,
				EnableJibInit:          true,
				EnableJibGradleInit:    true,
				EnableBuildpacksInit:   true,
				EnableNewInitFormat:    true,
				BuildpacksBuilder:      "buildpacks/builder",
				Opts:                   opts,
				MaxFileSize:            maxFileSize,
			},
		},

		{
			name: "enableJibInit implies enableNewInitFormat",
			args: []string{
				"init",
				"--XXenableJibInit",
			},
			expectedConfig: config.Config{
				ComposeFile:            "",
				CliArtifacts:           nil,
				CliKubernetesManifests: nil,
				SkipBuild:              false,
				SkipDeploy:             false,
				Force:                  false,
				Analyze:                false,
				EnableJibInit:          true,
				EnableBuildpacksInit:   false,
				EnableNewInitFormat:    true,
				BuildpacksBuilder:      "gcr.io/buildpacks/builder:v1",
				Opts:                   opts,
				MaxFileSize:            maxFileSize,
			},
		},
		{
			name: "enableBuildpackInit implies enableNewInitFormat",
			args: []string{
				"init",
				"--XXenableBuildpacksInit",
			},
			expectedConfig: config.Config{
				ComposeFile:            "",
				CliArtifacts:           nil,
				CliKubernetesManifests: nil,
				SkipBuild:              false,
				SkipDeploy:             false,
				Force:                  false,
				Analyze:                false,
				EnableJibInit:          false,
				EnableBuildpacksInit:   true,
				EnableNewInitFormat:    true,
				BuildpacksBuilder:      "gcr.io/buildpacks/builder:v1",
				Opts:                   opts,
				MaxFileSize:            maxFileSize,
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			var capturedConfig config.Config
			t.Override(&initEntrypoint, func(_ context.Context, _ io.Writer, c config.Config) error {
				capturedConfig = c
				return test.initResult
			})
			t.SetArgs(test.args)

			err := NewCmdInit().Execute()

			// we ignore Skaffold options
			test.expectedConfig.Opts = capturedConfig.Opts
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedConfig, capturedConfig, cmp.AllowUnexported(cfg.StringOrUndefined{}))
		})
	}
}
