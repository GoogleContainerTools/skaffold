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
	"context"
	"io"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	initconfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestTransparentInit(t *testing.T) {
	tests := []struct {
		name             string
		dir              string
		config           initconfig.Config
		expectedError    string
		expectedExitCode int
		doneResponse     bool
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
			name: "error writing config file",
			dir:  "testdata/init/hello",
			config: initconfig.Config{
				Force: true,
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
		{
			name: "user selects 'no'",
			dir:  "testdata/init/hello",
			config: initconfig.Config{
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
			doneResponse: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Chdir(test.dir)

			t.Override(&confirmInitOptions, func(_ io.Writer, _ *latest.SkaffoldConfig) (bool, error) {
				return test.doneResponse, nil
			})

			got, err := Transparent(context.TODO(), os.Stdout, test.config)

			switch {
			case test.expectedError != "":
				t.CheckErrorContains(test.expectedError, err)
				t.CheckDeepEqual(exitCode(err), test.expectedExitCode)
			case test.doneResponse == true:
				t.CheckErrorAndDeepEqual(false, err, (*latest.SkaffoldConfig)(nil), got)
			default:
				t.CheckNoError(err)
				checkGeneratedConfig(t, ".")
			}
		})
	}
}
