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

package deploy

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestKustomizeDeploy(t *testing.T) {
	tests := []struct {
		description string
		cfg         *latest.KustomizeDeploy
		builds      []build.Artifact
		commands    util.Command
		shouldErr   bool
		forceDeploy bool
	}{
		{
			description: "no manifest",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: ".",
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kustomize build .", ""),
		},
		{
			description: "deploy success",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: ".",
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kustomize build .", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			forceDeploy: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			t.NewTempDir().
				Chdir()

			k := NewKustomizeDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: test.cfg,
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					Namespace: testNamespace,
					Force:     test.forceDeploy,
				},
			})
			err := k.Deploy(context.Background(), ioutil.Discard, test.builds, nil).GetError()

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKustomizeCleanup(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	tests := []struct {
		description string
		cfg         *latest.KustomizeDeploy
		commands    util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: tmpDir.Root(),
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -"),
		},
		{
			description: "cleanup error",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: tmpDir.Root(),
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), deploymentWebYAML).
				AndRunErr("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "fail to read manifests",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: tmpDir.Root(),
			},
			commands: testutil.CmdRunOutErr(
				"kustomize build "+tmpDir.Root(),
				"",
				errors.New("BUG"),
			),
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)

			k := NewKustomizeDeployer(&runcontext.RunContext{
				WorkingDir: tmpDir.Root(),
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: test.cfg,
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					Namespace: testNamespace,
				},
			})
			err := k.Cleanup(context.Background(), ioutil.Discard)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestDependenciesForKustomization(t *testing.T) {
	tests := []struct {
		description        string
		yaml               string
		expected           []string
		shouldErr          bool
		skipConfigCreation bool
		createFiles        map[string]string
		configName         string
	}{
		{
			description: "resources",
			yaml:        `resources: [pod1.yaml, path/pod2.yaml]`,
			expected:    []string{"kustomization.yaml", "pod1.yaml", "path/pod2.yaml"},
			createFiles: map[string]string{
				"pod1.yaml":      "",
				"path/pod2.yaml": "",
			},
		},
		{
			description: "paches",
			yaml:        `patches: [patch1.yaml, path/patch2.yaml]`,
			expected:    []string{"kustomization.yaml", "patch1.yaml", "path/patch2.yaml"},
		},
		{
			description: "patchesStrategicMerge",
			yaml:        `patchesStrategicMerge: [patch1.yaml, path/patch2.yaml]`,
			expected:    []string{"kustomization.yaml", "patch1.yaml", "path/patch2.yaml"},
		},
		{
			description: "crds",
			yaml:        `patches: [crd1.yaml, path/crd2.yaml]`,
			expected:    []string{"kustomization.yaml", "crd1.yaml", "path/crd2.yaml"},
		},
		{
			description: "patches json 6902",
			yaml: `patchesJson6902:
- path: patch1.json
- path: path/patch2.json`,
			expected: []string{"kustomization.yaml", "patch1.json", "path/patch2.json"},
		},
		{
			description: "configMapGenerator",
			yaml: `configMapGenerator:
- files: [app1.properties]
- files: [app2.properties, app3.properties]`,
			expected: []string{"kustomization.yaml", "app1.properties", "app2.properties", "app3.properties"},
		},
		{
			description: "secretGenerator",
			yaml: `secretGenerator:
- files: [secret1.file]
- files: [secret2.file, secret3.file]`,
			expected: []string{"kustomization.yaml", "secret1.file", "secret2.file", "secret3.file"},
		},
		{
			description: "base exists locally",
			yaml:        `bases: [base]`,
			expected:    []string{"kustomization.yaml", "base/kustomization.yaml", "base/app.yaml"},
			createFiles: map[string]string{
				"base/kustomization.yaml": `resources: [app.yaml]`,
				"base/app.yaml":           "",
			},
		},
		{
			description: "missing base locally",
			yaml:        `bases: [missing-or-remote-base]`,
			expected:    []string{"kustomization.yaml"},
		},
		{
			description: "local kustomization resource",
			yaml:        `resources: [app.yaml, base]`,
			expected:    []string{"kustomization.yaml", "app.yaml", "base/kustomization.yaml", "base/app.yaml"},
			createFiles: map[string]string{
				"app.yaml":                "",
				"base/kustomization.yaml": `resources: [app.yaml]`,
				"base/app.yaml":           "",
			},
		},
		{
			description: "missing local kustomization resource",
			yaml:        `resources: [app.yaml, missing-or-remote-base]`,
			expected:    []string{"kustomization.yaml", "app.yaml"},
			createFiles: map[string]string{
				"app.yaml": "",
			},
		},
		{
			description: "mixed resource types",
			yaml:        `resources: [app.yaml, missing-or-remote-base, base]`,
			expected:    []string{"kustomization.yaml", "app.yaml", "base/kustomization.yaml", "base/app.yaml"},
			createFiles: map[string]string{
				"app.yaml":                "",
				"base/kustomization.yaml": `resources: [app.yaml]`,
				"base/app.yaml":           "",
			},
		},
		{
			description: "alt config name: kustomization.yml",
			yaml:        `resources: [app.yaml]`,
			expected:    []string{"kustomization.yml", "app.yaml"},
			createFiles: map[string]string{
				"app.yaml": "",
			},
			configName: "kustomization.yml",
		},
		{
			description: "alt config name: Kustomization",
			yaml:        `resources: [app.yaml]`,
			expected:    []string{"Kustomization", "app.yaml"},
			createFiles: map[string]string{
				"app.yaml": "",
			},
			configName: "Kustomization",
		},
		{
			description: "mixture of config names",
			yaml:        `resources: [app.yaml, base1, base2]`,
			expected:    []string{"Kustomization", "app.yaml", "base1/kustomization.yml", "base1/app.yaml", "base2/kustomization.yaml", "base2/app.yaml"},
			createFiles: map[string]string{
				"app.yaml":                 "",
				"base1/kustomization.yml":  `resources: [app.yaml]`,
				"base1/app.yaml":           "",
				"base2/kustomization.yaml": `resources: [app.yaml]`,
				"base2/app.yaml":           "",
			},
			configName: "Kustomization",
		},
		{
			description:        "remote or missing root kustomization config",
			expected:           []string{},
			configName:         "missing-or-remote-root-config",
			skipConfigCreation: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.configName == "" {
				test.configName = "kustomization.yaml"
			}

			tmpDir := t.NewTempDir()

			if !test.skipConfigCreation {
				tmpDir.Write(test.configName, test.yaml)
			}

			for path, contents := range test.createFiles {
				tmpDir.Write(path, contents)
			}

			k := NewKustomizeDeployer(&runcontext.RunContext{
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: &latest.KustomizeDeploy{
								KustomizePath: tmpDir.Root(),
							},
						},
					},
				},
				KubeContext: testKubeContext,
			})
			deps, err := k.Dependencies()

			t.CheckErrorAndDeepEqual(test.shouldErr, err, tmpDir.Paths(test.expected...), deps)
		})
	}
}
