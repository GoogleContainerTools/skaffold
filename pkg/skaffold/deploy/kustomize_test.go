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
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
				KustomizePaths: []string{"."},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kustomize build .", ""),
		},
		{
			description: "deploy success",
			cfg: &latest.KustomizeDeploy{
				KustomizePaths: []string{"."},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kustomize build .", deploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []build.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy: true,
		},
		{
			description: "deploy success with multiple kustomizations",
			cfg: &latest.KustomizeDeploy{
				KustomizePaths: []string{"a", "b"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112).
				AndRunOut("kustomize build a", deploymentWebYAML).
				AndRunOut("kustomize build b", deploymentAppYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", deploymentWebYAMLv1+"\n---\n"+deploymentAppYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []build.Artifact{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:v1",
				},
				{
					ImageName: "leeroy-app",
					Tag:       "leeroy-app:v1",
				},
			},
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
					WaitForDeletions: config.WaitForDeletions{
						Enabled: true,
						Delay:   0 * time.Second,
						Max:     10 * time.Second,
					},
				},
			}, nil)
			_, err := k.Deploy(context.Background(), ioutil.Discard, test.builds)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKustomizeCleanup(t *testing.T) {
	tmpDir := testutil.NewTempDir(t)

	tests := []struct {
		description string
		cfg         *latest.KustomizeDeploy
		commands    util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			cfg: &latest.KustomizeDeploy{
				KustomizePaths: []string{tmpDir.Root()},
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -"),
		},
		{
			description: "cleanup success with multiple kustomizations",
			cfg: &latest.KustomizeDeploy{
				KustomizePaths: tmpDir.Paths("a", "b"),
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Path("a"), deploymentWebYAML).
				AndRunOut("kustomize build "+tmpDir.Path("b"), deploymentAppYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -"),
		},
		{
			description: "cleanup error",
			cfg: &latest.KustomizeDeploy{
				KustomizePaths: []string{tmpDir.Root()},
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), deploymentWebYAML).
				AndRunErr("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "fail to read manifests",
			cfg: &latest.KustomizeDeploy{
				KustomizePaths: []string{tmpDir.Root()},
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
			}, nil)
			err := k.Cleanup(context.Background(), ioutil.Discard)

			t.CheckError(test.shouldErr, err)
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
			description:    "patches legacy",
			kustomizations: map[string]string{"kustomization.yaml": `patches: [patch1.yaml, path/patch2.yaml]`},
			expected:       []string{"kustomization.yaml", "patch1.yaml", "path/patch2.yaml"},
		},
		{
			description:    "patchesStrategicMerge",
			kustomizations: map[string]string{"kustomization.yaml": `patchesStrategicMerge: [patch1.yaml, "patch2.yaml", 'path/patch3.yaml']`},
			expected:       []string{"kustomization.yaml", "patch1.yaml", "patch2.yaml", "path/patch3.yaml"},
		},
		{
			description: "inline patchesStrategicMerge",
			kustomizations: map[string]string{"kustomization.yaml": `patchesStrategicMerge:
- |-
 apiVersion: v1`},
			expected: []string{"kustomization.yaml"},
		},
		{
			description:    "crds",
			kustomizations: map[string]string{"kustomization.yaml": `patches: [crd1.yaml, path/crd2.yaml]`},
			expected:       []string{"crd1.yaml", "kustomization.yaml", "path/crd2.yaml"},
		},
		{
			description: "patches json 6902",
			kustomizations: map[string]string{"kustomization.yaml": `patchesJson6902:
- path: patch1.json
- path: path/patch2.json`},
			expected: []string{"kustomization.yaml", "patch1.json", "path/patch2.json"},
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

			k := NewKustomizeDeployer(&runcontext.RunContext{
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: &latest.KustomizeDeploy{
								KustomizePaths: kustomizePaths,
							},
						},
					},
				},
				KubeContext: testKubeContext,
			}, nil)
			deps, err := k.Dependencies()

			t.CheckErrorAndDeepEqual(test.shouldErr, err, tmpDir.Paths(test.expected...), deps)
		})
	}
}

func TestKustomizeBuildCommandArgs(t *testing.T) {
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
			expectedArgs:  []string{"build"},
		},
		{
			description:   "One BuildArg, empty KustomizePaths",
			buildArgs:     []string{"--foo"},
			kustomizePath: "",
			expectedArgs:  []string{"build", "--foo"},
		},
		{
			description:   "no BuildArgs, non-empty KustomizePaths",
			buildArgs:     []string{},
			kustomizePath: "foo",
			expectedArgs:  []string{"build", "foo"},
		},
		{
			description:   "One BuildArg, non-empty KustomizePaths",
			buildArgs:     []string{"--foo"},
			kustomizePath: "bar",
			expectedArgs:  []string{"build", "--foo", "bar"},
		},
		{
			description:   "Multiple BuildArg, empty KustomizePaths",
			buildArgs:     []string{"--foo", "--bar"},
			kustomizePath: "",
			expectedArgs:  []string{"build", "--foo", "--bar"},
		},
		{
			description:   "Multiple BuildArg with spaces, empty KustomizePaths",
			buildArgs:     []string{"--foo bar", "--baz"},
			kustomizePath: "",
			expectedArgs:  []string{"build", "--foo", "bar", "--baz"},
		},
		{
			description:   "Multiple BuildArg with spaces, non-empty KustomizePaths",
			buildArgs:     []string{"--foo bar", "--baz"},
			kustomizePath: "barfoo",
			expectedArgs:  []string{"build", "--foo", "bar", "--baz", "barfoo"},
		},
		{
			description:   "Multiple BuildArg no spaces, non-empty KustomizePaths",
			buildArgs:     []string{"--foo", "bar", "--baz"},
			kustomizePath: "barfoo",
			expectedArgs:  []string{"build", "--foo", "bar", "--baz", "barfoo"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			args := buildCommandArgs(test.buildArgs, test.kustomizePath)
			t.CheckDeepEqual(test.expectedArgs, args)
		})
	}
}

func TestKustomizeRender(t *testing.T) {
	type kustomizationCall struct {
		folder      string
		buildResult string
	}
	tests := []struct {
		description    string
		builds         []build.Artifact
		labels         map[string]string
		kustomizations []kustomizationCall
		expected       string
		shouldErr      bool
	}{
		{
			description: "single kustomization",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			kustomizations: []kustomizationCall{
				{
					folder: ".",
					buildResult: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
  - image: gcr.io/project/image2
    name: image2
`,
				},
			},
			expected: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
		{
			description: "single kustomization with user labels",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			labels: map[string]string{"user/label": "test"},
			kustomizations: []kustomizationCall{
				{
					folder: ".",
					buildResult: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
  - image: gcr.io/project/image2
    name: image2
`,
				},
			},
			expected: `apiVersion: v1
kind: Pod
metadata:
  labels:
    user/label: test
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
		{
			description: "multiple kustomizations",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			kustomizations: []kustomizationCall{
				{
					folder: "a",
					buildResult: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
`,
				},
				{
					folder: "b",
					buildResult: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image2
    name: image2
`,
				},
			},
			expected: `apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
---
apiVersion: v1
kind: Pod
metadata:
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var kustomizationPaths []string
			fakeCmd := testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion112)
			for _, kustomizationCall := range test.kustomizations {
				fakeCmd.AndRunOut("kustomize build "+kustomizationCall.folder, kustomizationCall.buildResult)
				kustomizationPaths = append(kustomizationPaths, kustomizationCall.folder)
			}
			t.Override(&util.DefaultExecCommand, fakeCmd)
			t.NewTempDir().
				Chdir()

			k := NewKustomizeDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: &latest.KustomizeDeploy{
								KustomizePaths: kustomizationPaths,
							},
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					Namespace: testNamespace,
				},
			}, test.labels)
			var b bytes.Buffer
			err := k.Render(context.Background(), &b, test.builds, true, "")
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, b.String())
		})
	}
}
