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

package ko

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands/options"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const (
	testKoBuildOptionsEnvVar = "TEST_KO_BUILDER_IMAGE_LABEL_ENV_VAR"
)

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		description string
		artifact    latest.Artifact
		platforms   platform.Matcher
		envVarValue string
		runMode     config.RunMode
		wantBo      options.BuildOptions
	}{
		{
			description: "all zero value",
			artifact: latest.Artifact{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{},
				},
			},
			wantBo: options.BuildOptions{
				ConcurrentBuilds: 1,
				SBOM:             "none",
				Trimpath:         true,
				UserAgent:        version.UserAgentWithClient(),
			},
		},
		{
			description: "all options",
			artifact: latest.Artifact{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{
						BaseImage: "gcr.io/distroless/base:nonroot",
						Dir:       "gomoddir",
						Env: []string{
							"FOO=BAR",
							fmt.Sprintf("frob={{.%s}}", testKoBuildOptionsEnvVar),
						},
						Flags: []string{
							"-v",
							fmt.Sprintf("-flag-{{.%s}}", testKoBuildOptionsEnvVar),
							fmt.Sprintf("-flag2-{{.Env.%s}}", testKoBuildOptionsEnvVar),
						},
						Labels: map[string]string{
							"foo":  "bar",
							"frob": fmt.Sprintf("{{.%s}}", testKoBuildOptionsEnvVar),
						},
						Ldflags: []string{
							"-s",
							fmt.Sprintf("-ldflag-{{.%s}}", testKoBuildOptionsEnvVar),
							fmt.Sprintf("-ldflag2-{{.Env.%s}}", testKoBuildOptionsEnvVar),
						},
						Main: "cmd/app",
					},
				},
				ImageName: "ko://example.com/foo",
				Workspace: "workdir",
			},
			platforms: platform.Matcher{Platforms: []specs.Platform{{OS: "linux", Architecture: "amd64"}, {OS: "linux", Architecture: "arm64"}}},

			envVarValue: "baz",
			runMode:     config.RunModes.Debug,
			wantBo: options.BuildOptions{
				BaseImage: "gcr.io/distroless/base:nonroot",
				BuildConfigs: map[string]build.Config{
					"example.com/foo": {
						ID:      "ko://example.com/foo",
						Dir:     ".",
						Env:     []string{"FOO=BAR", "frob=baz"},
						Flags:   build.FlagArray{"-v", "-flag-baz", "-flag2-baz"},
						Ldflags: build.StringArray{"-s", "-ldflag-baz", "-ldflag2-baz"},
						Main:    "cmd/app",
					},
				},
				ConcurrentBuilds:     1,
				DisableOptimizations: true,
				Labels:               []string{"foo=bar", "frob=baz"},
				Platforms:            []string{"linux/amd64", "linux/arm64"},
				SBOM:                 "none",
				Trimpath:             false,
				UserAgent:            version.UserAgentWithClient(),
				WorkingDirectory:     "workdir" + string(filepath.Separator) + "gomoddir",
			},
		},
		{
			description: "compatibility with ko envvar expansion syntax for flags and ldflags",
			artifact: latest.Artifact{
				ArtifactType: latest.ArtifactType{
					KoArtifact: &latest.KoArtifact{
						Flags: []string{
							"-v",
							fmt.Sprintf("-flag-{{.Env.%s}}", testKoBuildOptionsEnvVar),
						},
						Ldflags: []string{
							"-s",
							fmt.Sprintf("-ldflag-{{.Env.%s}}", testKoBuildOptionsEnvVar),
						},
					},
				},
				ImageName: "ko://example.com/foo",
			},
			envVarValue: "xyzzy",
			wantBo: options.BuildOptions{
				BuildConfigs: map[string]build.Config{
					"example.com/foo": {
						ID:      "ko://example.com/foo",
						Dir:     ".",
						Flags:   build.FlagArray{"-v", "-flag-xyzzy"},
						Ldflags: build.StringArray{"-s", "-ldflag-xyzzy"},
					},
				},
				ConcurrentBuilds: 1,
				SBOM:             "none",
				Trimpath:         true,
				UserAgent:        version.UserAgentWithClient(),
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Setenv(testKoBuildOptionsEnvVar, test.envVarValue)
			gotBo, err := buildOptions(&test.artifact, test.runMode, test.platforms)
			t.CheckErrorAndFailNow(false, err)
			t.CheckDeepEqual(test.wantBo, *gotBo,
				cmpopts.EquateEmpty(),
				cmpopts.SortSlices(func(x, y string) bool { return x < y }),
			)
		})
	}
}
