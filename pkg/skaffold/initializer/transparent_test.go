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
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	initconfig "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const manifest = `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-example
  labels:
    app: skaffold-example
spec:
  selector:
    matchLabels:
      app: skaffold-example
  replicas: 2
  template:
    metadata:
      labels:
        app: skaffold-example
    spec:
      containers:
      - name: skaffold-example
        image: skaffold-helm
`

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
			name: "helm init passes",
			dir:  "testdata/init/helm-deployment",
			config: initconfig.Config{
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
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

			tmpDir := t.NewTempDir()
			tmpDir.Write("template/deployment.yaml", manifest)
			cmd := fmt.Sprintf("helm template charts -f charts/values.yaml --output-dir %s", tmpDir.Path("template"))
			t.Override(&util.DefaultExecCommand, testutil.CmdRun(cmd))
			t.Override(&render.TempDir, func(dir, pattern string) (name string, err error) {
				return tmpDir.Path("template"), nil
			})

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

func TestValidCmd(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{
			name:     "valid string",
			cmd:      "dev",
			expected: true,
		},
		{
			name:     "invalid",
			cmd:      "build",
			expected: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			config := config.SkaffoldOptions{
				Command: test.cmd,
			}
			valid := ValidCmd(config)

			t.CheckDeepEqual(test.expected, valid)
		})
	}
}
