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
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestKustomizeDeploy(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *latest.KustomizeDeploy
		builds      []build.Artifact
		command     util.Command
		shouldErr   bool
		forceDeploy bool
	}{
		{
			description: "no manifest",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: ".",
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl version --client -ojson", kubectlVersion).
				WithRunOut("kustomize build .", ""),
		},
		{
			description: "deploy success",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: ".",
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kubectl version --client -ojson", kubectlVersion).
				WithRunOut("kustomize build .", deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:123",
			}},
			forceDeploy: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.command)
			t.NewTempDir().
				Chdir()

			k := NewKustomizeDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: &latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: test.cfg,
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: &config.SkaffoldOptions{
					Namespace: testNamespace,
					Force:     test.forceDeploy,
				},
			})
			err := k.Deploy(context.Background(), ioutil.Discard, test.builds, nil)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKustomizeCleanup(t *testing.T) {
	tmpDir, cleanup := testutil.NewTempDir(t)
	defer cleanup()

	var tests = []struct {
		description string
		cfg         *latest.KustomizeDeploy
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: tmpDir.Root(),
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kustomize build "+tmpDir.Root(), deploymentWebYAML).
				WithRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -"),
		},
		{
			description: "cleanup error",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: tmpDir.Root(),
			},
			command: testutil.NewFakeCmd(t).
				WithRunOut("kustomize build "+tmpDir.Root(), deploymentWebYAML).
				WithRunErr("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "fail to read manifests",
			cfg: &latest.KustomizeDeploy{
				KustomizePath: tmpDir.Root(),
			},
			command:   testutil.FakeRunOutErr(t, "kustomize build "+tmpDir.Root(), ``, errors.New("BUG")),
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.command)

			k := NewKustomizeDeployer(&runcontext.RunContext{
				WorkingDir: tmpDir.Root(),
				Cfg: &latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: test.cfg,
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: &config.SkaffoldOptions{
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
		description string
		yaml        string
		expected    []string
		shouldErr   bool
	}{
		{
			description: "resources",
			yaml:        `resources: [pod1.yaml, path/pod2.yaml]`,
			expected:    []string{"kustomization.yaml", "pod1.yaml", "path/pod2.yaml"},
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
			description: "unknown base",
			yaml:        `bases: [other]`,
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("kustomization.yaml", test.yaml)

			k := NewKustomizeDeployer(&runcontext.RunContext{
				Cfg: &latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: &latest.KustomizeDeploy{
								KustomizePath: tmpDir.Root(),
							},
						},
					},
				},
				KubeContext: testKubeContext,
				Opts:        &config.SkaffoldOptions{},
			})
			deps, err := k.Dependencies()

			t.CheckErrorAndDeepEqual(test.shouldErr, err, tmpDir.Paths(test.expected...), deps)
		})
	}
}
