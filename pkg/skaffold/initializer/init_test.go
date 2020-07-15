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

package initializer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	initconfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDoInit(t *testing.T) {
	tests := []struct {
		name             string
		dir              string
		config           initconfig.Config
		expectedError    string
		expectedExitCode int
	}{
		//TODO: mocked kompose test
		{
			name: "getting-started",
			dir:  "testdata/init/hello",
			config: initconfig.Config{
				Force: true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "ignore existing tags",
			dir:  "testdata/init/ignore-tags",
			config: initconfig.Config{
				Force: true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "microservices (backwards compatibility)",
			dir:  "testdata/init/microservices",
			config: initconfig.Config{
				Force: true,
				CliArtifacts: []string{
					"leeroy-app/Dockerfile=gcr.io/k8s-skaffold/leeroy-app",
					"leeroy-web/Dockerfile=gcr.io/k8s-skaffold/leeroy-web",
				},
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "microservices",
			dir:  "testdata/init/microservices",
			config: initconfig.Config{
				Force: true,
				CliArtifacts: []string{
					`{"builder":"Docker","payload":{"path":"leeroy-app/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-app"}`,
					`{"builder":"Docker","payload":{"path":"leeroy-web/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-web"}`,
				},
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "CLI artifacts + manifest placeholders",
			dir:  "testdata/init/allcli",
			config: initconfig.Config{
				Force: true,
				CliArtifacts: []string{
					`{"builder":"Docker","payload":{"path":"Dockerfile"},"image":"passed-in-artifact"}`,
				},
				CliKubernetesManifests: []string{
					"manifest-placeholder1.yaml",
					"manifest-placeholder2.yaml",
				},
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "CLI artifacts but no manifests",
			dir:  "testdata/init/allcli",
			config: initconfig.Config{
				Force: true,
				CliArtifacts: []string{
					`{"builder":"Docker","payload":{"path":"Dockerfile"},"image":"passed-in-artifact"}`,
				},
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
			expectedError:    "one or more valid Kubernetes manifests are required to run skaffold",
			expectedExitCode: 102,
		},
		{
			name: "error writing config file",
			dir:  "testdata/init/microservices",
			config: initconfig.Config{
				Force: true,
				CliArtifacts: []string{
					`{"builder":"Docker","payload":{"path":"leeroy-app/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-app"}`,
					`{"builder":"Docker","payload":{"path":"leeroy-web/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-web"}`,
				},
				Opts: config.SkaffoldOptions{
					// erroneous config file as . is a directory
					ConfigurationFile: ".",
				},
			},
			expectedError:    "writing config to file: open .: is a directory",
			expectedExitCode: 1,
		},
		{
			name: "error no builders",
			dir:  "testdata/init/no-builder",
			config: initconfig.Config{
				Force: true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
			expectedError:    "please provide at least one build config",
			expectedExitCode: 101,
		},
		{
			name: "error no manifests",
			dir:  "testdata/init/hello-no-manifest",
			config: initconfig.Config{
				Force: true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
			expectedError:    "one or more valid Kubernetes manifests are required to run skaffold",
			expectedExitCode: 102,
		},
		{
			name: "existing config",
			dir:  "testdata/init/hello",
			config: initconfig.Config{
				Force: false,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml",
				},
			},
			expectedError:    "pre-existing skaffold.yaml found",
			expectedExitCode: 103,
		},
		{
			name: "builder/image ambiguity",
			dir:  "testdata/init/microservices",
			config: initconfig.Config{
				Force: true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
			expectedError:    "unable to automatically resolve builder/image pairs",
			expectedExitCode: 104,
		},
		{
			name: "kustomize",
			dir:  "testdata/init/getting-started-kustomize",
			config: initconfig.Config{
				Force: true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "helm fails",
			dir:  "testdata/init/helm-deployment",
			config: initconfig.Config{
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
			expectedError: `Projects set up to deploy with helm must be manually configured.

See https://skaffold.dev/docs/pipeline-stages/deployers/helm/ for a detailed guide on setting your project up with skaffold.`,
			expectedExitCode: 1,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Chdir(test.dir)

			err := DoInit(context.TODO(), os.Stdout, test.config)

			if test.expectedError != "" {
				t.CheckErrorContains(test.expectedError, err)
				t.CheckDeepEqual(exitCode(err), test.expectedExitCode)
			} else {
				t.CheckNoError(err)
				checkGeneratedConfig(t, ".")
			}
		})
	}
}

func TestDoInitAnalyze(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		config      initconfig.Config
		expectedOut string
	}{
		{
			name: "analyze microservices",
			dir:  "testdata/init/microservices",
			config: initconfig.Config{
				Analyze: true,
			},
			expectedOut: strip(`{
							"dockerfiles":["leeroy-app/Dockerfile","leeroy-web/Dockerfile"],
							"images":["gcr.io/k8s-skaffold/leeroy-app","gcr.io/k8s-skaffold/leeroy-web"]
							}`) + "\n",
		},
		{
			name: "analyze microservices new format",
			dir:  "testdata/init/microservices",
			config: initconfig.Config{
				Analyze:             true,
				EnableNewInitFormat: true,
			},
			expectedOut: strip(`{
									"builders":[
										{"name":"Docker","payload":{"path":"leeroy-app/Dockerfile"}},
										{"name":"Docker","payload":{"path":"leeroy-web/Dockerfile"}}
									],
									"images":[
										{"name":"gcr.io/k8s-skaffold/leeroy-app","foundMatch":false},
										{"name":"gcr.io/k8s-skaffold/leeroy-web","foundMatch":false}]}`) + "\n",
		},
		{
			name: "no error with no manifests in analyze mode with skip-deploy",
			dir:  "testdata/init/hello-no-manifest",

			config: initconfig.Config{
				Analyze:    true,
				SkipDeploy: true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},

			expectedOut: strip(`{"dockerfiles":["Dockerfile"]}`) + "\n",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			var out bytes.Buffer
			t.Chdir(test.dir)

			err := DoInit(context.TODO(), &out, test.config)

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedOut, out.String())
		})
	}
}

func strip(s string) string {
	cutString := "\n\t\r"
	stripped := ""
	for _, r := range s {
		if strings.ContainsRune(cutString, r) {
			continue
		}
		stripped = fmt.Sprintf("%s%c", stripped, r)
	}
	return stripped
}

func checkGeneratedConfig(t *testutil.T, dir string) {
	expectedOutput, err := schema.ParseConfig(filepath.Join(dir, "skaffold.yaml"))
	t.CheckNoError(err)

	output, err := schema.ParseConfig(filepath.Join(dir, "skaffold.yaml.out"))
	t.CheckNoError(err)
	t.CheckDeepEqual(expectedOutput, output)
}

type ExitCoder interface {
	ExitCode() int
}

func exitCode(err error) int {
	var exitErr ExitCoder
	if ok := errors.As(err, &exitErr); ok {
		return exitErr.ExitCode()
	}

	return 1
}
