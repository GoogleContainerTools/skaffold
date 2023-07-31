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

package kubectl

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/kustomize"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestKustomizeRenderDeploy(t *testing.T) {
	tests := []struct {
		description                 string
		paths                       []string
		kDeploy                     latest.KubectlDeploy
		builds                      []graph.Artifact
		commands                    util.Command
		shouldErr                   bool
		forceDeploy                 bool
		skipSkaffoldNamespaceOption bool
		envs                        map[string]string
	}{
		{
			description: "deploy success",
			paths:       []string{"."},
			commands: testutil.
				CmdRunOut("kustomize build .", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy: true,
		},
		{
			description: "deploy success (default namespace)",
			paths:       []string{"."},
			kDeploy: latest.KubectlDeploy{
				DefaultNamespace: &TestNamespace2,
			},
			commands: testutil.
				CmdRunOut("kustomize build .", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace2 get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace2 apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy:                 true,
			skipSkaffoldNamespaceOption: true,
		},
		{
			description: "deploy success (default namespace with env template)",
			kDeploy: latest.KubectlDeploy{
				DefaultNamespace: &TestNamespace2FromEnvTemplate,
			},
			paths: []string{"."},
			commands: testutil.
				CmdRunOut("kustomize build .", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace2 get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace2 apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy:                 true,
			skipSkaffoldNamespaceOption: true,
			envs: map[string]string{
				"MYENV": "Namesp",
			},
		},
		{
			description: "deploy success (kustomizePaths with env template)",
			paths:       []string{"{{ .MYENV }}"},
			commands: testutil.
				CmdRunOut("kustomize build a", DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy: true,
			envs: map[string]string{
				"MYENV": "a",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetEnvs(test.envs)
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&client.Client, deployutil.MockK8sClient)
			t.Override(&generate.KustomizeBinaryCheck, func() bool { return true })
			t.Override(&generate.KubectlVersionCheck, func(*kubectl.CLI) bool { return true })
			tmpDir := t.NewTempDir()
			setUpKustomizePaths(tmpDir)
			tmpDir.Chdir()
			skaffoldNamespaceOption := ""
			if !test.skipSkaffoldNamespaceOption {
				skaffoldNamespaceOption = TestNamespace
			}
			const configName = "default"
			rc := latest.RenderConfig{Generate: latest.Generate{
				Kustomize: &latest.Kustomize{
					Paths: test.paths,
				},
			}}
			mockCfg := &kubectlConfig{
				RunContext: runcontext.RunContext{},
			}
			r, err := kustomize.New(mockCfg, rc, map[string]string{}, configName, skaffoldNamespaceOption, nil, !test.skipSkaffoldNamespaceOption)
			t.CheckNoError(err)
			var b bytes.Buffer
			m, errR := r.Render(context.Background(), &b, test.builds, true)
			t.CheckNoError(errR)

			k, err := NewDeployer(&kubectlConfig{
				force: test.forceDeploy,
				waitForDeletions: config.WaitForDeletions{
					Enabled: true,
					Delay:   0 * time.Second,
					Max:     10 * time.Second,
				},
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{
					Namespace: skaffoldNamespaceOption,
				}}}, &label.DefaultLabeller{}, &test.kDeploy, nil, "default")
			t.RequireNoError(err)

			err = k.Deploy(context.Background(), io.Discard, test.builds, m)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKustomizeCleanup(t *testing.T) {
	tmpDir := testutil.NewTempDir(t)
	setUpKustomizePaths(tmpDir)
	tests := []struct {
		description string
		paths       []string
		commands    util.Command
		shouldErr   bool
		renderErr   bool
		dryRun      bool
	}{
		{
			description: "cleanup dry-run",
			paths:       []string{"."},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), DeploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --dry-run --ignore-not-found=true --wait=false -f -"),
			dryRun: true,
		},
		{
			description: "cleanup success",
			paths:       []string{"."},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), DeploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -"),
		},
		{
			description: "cleanup error",
			paths:       []string{"."},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), DeploymentWebYAML).
				AndRunErr("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "fail to read manifests",
			paths:       []string{"."},
			commands: testutil.
				CmdRunOutErr("kustomize build "+tmpDir.Root(), "", errors.New("BUG")),
			renderErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&generate.KustomizeBinaryCheck, func() bool { return true })
			t.Override(&generate.KubectlVersionCheck, func(*kubectl.CLI) bool { return true })
			const configName = "default"
			rc := latest.RenderConfig{Generate: latest.Generate{
				Kustomize: &latest.Kustomize{
					Paths: test.paths,
				},
			}}
			mockCfg := &kubectlConfig{
				workingDir: tmpDir.Root(),
				RunContext: runcontext.RunContext{},
			}
			r, err := kustomize.New(mockCfg, rc, map[string]string{}, configName, TestNamespace, nil, true)
			t.CheckNoError(err)
			var b bytes.Buffer
			m, errR := r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}},
				true)
			t.CheckError(test.renderErr, errR)
			if !test.renderErr {
				k, err := NewDeployer(&kubectlConfig{
					RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{
						Namespace: TestNamespace}},
				}, &label.DefaultLabeller{}, &latest.KubectlDeploy{}, nil, "default")
				t.RequireNoError(err)
				err = k.Cleanup(context.Background(), io.Discard, test.dryRun, m)
				t.CheckError(test.shouldErr, err)
			}
		})
	}
}

func TestDependenciesForKustomization(t *testing.T) {
	tests := []struct {
		description    string
		expected       []string
		shouldErr      bool
		createFiles    map[string]string
		kustomizations map[string]string
	}{
		{
			description:    "resources",
			kustomizations: map[string]string{"kustomization.yaml": `resources: [pod1.yaml, path/pod2.yaml]`},
			expected:       []string{"kustomization.yaml", "path/pod2.yaml", "pod1.yaml"},
			createFiles: map[string]string{
				"pod1.yaml":      "",
				"path/pod2.yaml": "",
			},
		},
		{
			description: "extended patches with paths",
			kustomizations: map[string]string{"kustomization.yaml": `patches:
- path: patch1.yaml
  target:
    kind: Deployment`},
			createFiles: map[string]string{
				"patch1.yaml": "",
			},
			expected: []string{"kustomization.yaml", "patch1.yaml"},
		},
		{
			description: "extended patches with inline",
			kustomizations: map[string]string{"kustomization.yaml": `patches:
- patch: |-
    inline: patch
  target:
    kind: Deployment`},
			expected: []string{"kustomization.yaml"},
		},
		{
			description: "patches legacy",
			kustomizations: map[string]string{"kustomization.yaml": `patches: 
- path: patch1.yaml 
- path: path/patch2.yaml`},
			createFiles: map[string]string{
				"patch1.yaml":      "",
				"path/patch2.yaml": "",
			},
			expected: []string{"kustomization.yaml", "patch1.yaml", "path/patch2.yaml"},
		},
		{
			description: "ignore patches legacy, path doesn't exist",
			kustomizations: map[string]string{"kustomization.yaml": `patches: 
- path: patch1.yaml 
- path: path/patch2.yaml`},
			expected: []string{"kustomization.yaml"},
		},
		{
			description:    "patchesStrategicMerge",
			kustomizations: map[string]string{"kustomization.yaml": `patchesStrategicMerge: [patch1.yaml, "patch2.yaml", 'path/patch3.yaml']`},
			expected:       []string{"kustomization.yaml", "patch1.yaml", "patch2.yaml", "path/patch3.yaml"},
		},
		{
			description: "inline patches",
			kustomizations: map[string]string{"kustomization.yaml": `patches:
- patch: |-
  apiVersion: v1`},
			expected: []string{"kustomization.yaml"},
		},
		{
			description:    "crds",
			kustomizations: map[string]string{"kustomization.yaml": `crds: [crd1.yaml, path/crd2.yaml]`},
			expected:       []string{"crd1.yaml", "kustomization.yaml", "path/crd2.yaml"},
		},
		{
			description: "patches json 6902",
			kustomizations: map[string]string{"kustomization.yaml": `patchesJson6902:
- path: patch1.json
- path: path/patch2.json`},
			createFiles: map[string]string{
				"patch1.json":      "",
				"path/patch2.json": "",
			},
			expected: []string{"kustomization.yaml", "patch1.json", "path/patch2.json"},
		},
		{
			description: "ignore patches json 6902 path doesn't exist",
			kustomizations: map[string]string{"kustomization.yaml": `patchesJson6902:
- path: patch1.json`,
			},
			expected: []string{"kustomization.yaml"},
		},
		{
			description: "ignore patch without path",
			kustomizations: map[string]string{"kustomization.yaml": `patchesJson6902:
- patch: |-
    - op: replace
      path: /path
      value: any`},
			expected: []string{"kustomization.yaml"},
		},
		{
			description: "configMapGenerator",
			kustomizations: map[string]string{"kustomization.yaml": `configMapGenerator:
- files: [app1.properties]
- files: [app2.properties, app3.properties]
- env: app1.env
- envs: [app2.env, app3.env]`},
			expected: []string{"app1.env", "app1.properties", "app2.env", "app2.properties", "app3.env", "app3.properties", "kustomization.yaml"},
		},
		{
			description: "secretGenerator",
			kustomizations: map[string]string{"kustomization.yaml": `secretGenerator:
- files: [secret1.file]
- files: [secret2.file, secret3.file]
- env: secret1.env
- envs: [secret2.env, secret3.env]`},
			expected: []string{"kustomization.yaml", "secret1.env", "secret1.file", "secret2.env", "secret2.file", "secret3.env", "secret3.file"},
		},
		{
			description:    "base exists locally",
			kustomizations: map[string]string{"kustomization.yaml": `bases: [base]`},
			expected:       []string{"base/app.yaml", "base/kustomization.yaml", "kustomization.yaml"},
			createFiles: map[string]string{
				"base/kustomization.yaml": `resources: [app.yaml]`,
				"base/app.yaml":           "",
			},
		},
		{
			description:    "missing base locally",
			kustomizations: map[string]string{"kustomization.yaml": `bases: [missing-or-remote-base]`},
			expected:       []string{"kustomization.yaml"},
		},
		{
			description:    "local kustomization resource",
			kustomizations: map[string]string{"kustomization.yaml": `resources: [app.yaml, base]`},
			expected:       []string{"app.yaml", "base/app.yaml", "base/kustomization.yaml", "kustomization.yaml"},
			createFiles: map[string]string{
				"app.yaml":                "",
				"base/kustomization.yaml": `resources: [app.yaml]`,
				"base/app.yaml":           "",
			},
		},
		{
			description:    "missing local kustomization resource",
			kustomizations: map[string]string{"kustomization.yaml": `resources: [app.yaml, missing-or-remote-base]`},
			expected:       []string{"app.yaml", "kustomization.yaml"},
			createFiles: map[string]string{
				"app.yaml": "",
			},
		},
		{
			description:    "mixed resource types",
			kustomizations: map[string]string{"kustomization.yaml": `resources: [app.yaml, missing-or-remote-base, base]`},
			expected:       []string{"app.yaml", "base/app.yaml", "base/kustomization.yaml", "kustomization.yaml"},
			createFiles: map[string]string{
				"app.yaml":                "",
				"base/kustomization.yaml": `resources: [app.yaml]`,
				"base/app.yaml":           "",
			},
		},
		{
			description:    "alt config name: kustomization.yml",
			kustomizations: map[string]string{"kustomization.yml": `resources: [app.yaml]`},
			expected:       []string{"app.yaml", "kustomization.yml"},
			createFiles: map[string]string{
				"app.yaml": "",
			},
		},
		{
			description:    "alt config name: Kustomization",
			kustomizations: map[string]string{"Kustomization": `resources: [app.yaml]`},
			expected:       []string{"Kustomization", "app.yaml"},
			createFiles: map[string]string{
				"app.yaml": "",
			},
		},
		{
			description:    "mixture of config names",
			kustomizations: map[string]string{"Kustomization": `resources: [app.yaml, base1, base2]`},
			expected:       []string{"Kustomization", "app.yaml", "base1/app.yaml", "base1/kustomization.yml", "base2/app.yaml", "base2/kustomization.yaml"},
			createFiles: map[string]string{
				"app.yaml":                 "",
				"base1/kustomization.yml":  `resources: [app.yaml]`,
				"base1/app.yaml":           "",
				"base2/kustomization.yaml": `resources: [app.yaml]`,
				"base2/app.yaml":           "",
			},
		},
		{
			description: "multiple kustomizations",
			kustomizations: map[string]string{
				"a/kustomization.yaml": `resources: [../base1]`,
				"b/Kustomization":      `resources: [../base2]`,
			},
			expected: []string{"a/kustomization.yaml", "b/Kustomization", "base1/app.yaml", "base1/kustomization.yml", "base2/app.yaml", "base2/kustomization.yaml"},
			createFiles: map[string]string{
				"base1/kustomization.yml":  `resources: [app.yaml]`,
				"base1/app.yaml":           "",
				"base2/kustomization.yaml": `resources: [app.yaml]`,
				"base2/app.yaml":           "",
			},
		},
		{
			description: "remote or missing root kustomization config",
			expected:    []string{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir()

			var kustomizePaths []string
			for path, contents := range test.kustomizations {
				tmpDir.Write(path, contents)
				kustomizePaths = append(kustomizePaths, filepath.Dir(tmpDir.Path(path)))
			}

			for path, contents := range test.createFiles {
				tmpDir.Write(path, contents)
			}

			rc := latest.RenderConfig{Generate: latest.Generate{
				Kustomize: &latest.Kustomize{
					Paths: kustomizePaths,
				},
			}}
			mockCfg := &kubectlConfig{
				RunContext: runcontext.RunContext{},
			}
			r, err := kustomize.New(mockCfg, rc, map[string]string{}, "default", "", nil, false)
			t.CheckNoError(err)

			deps, err := r.ManifestDeps()

			t.CheckErrorAndDeepEqual(test.shouldErr, err, tmpDir.Paths(test.expected...), deps)
		})
	}
}

func setUpKustomizePaths(tmpDir *testutil.TempDir) {
	for _, d := range []string{".", "a", "b"} {
		// create dir
		if d != "." {
			tmpDir.Mkdir(d)
		}
		tmpDir.Write(filepath.Join(d, "kustomization.yaml"), "")
	}
}
