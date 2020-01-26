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
	"io/ioutil"
	"testing"

	"github.com/pkg/errors"

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
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kustomize build .", ""),
		},
		{
			description: "deploy success",
			cfg: &latest.KustomizeDeploy{
				KustomizePaths: []string{"."},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kustomize build .", deploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
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
				KustomizePaths: []string{tmpDir.Root()},
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), deploymentWebYAML).
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
			expected:    []string{"kustomization.yaml", "path/pod2.yaml", "pod1.yaml"},
			createFiles: map[string]string{
				"pod1.yaml":      "",
				"path/pod2.yaml": "",
			},
		},
		{
			description: "patches",
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
			expected:    []string{"crd1.yaml", "kustomization.yaml", "path/crd2.yaml"},
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
			expected: []string{"app1.properties", "app2.properties", "app3.properties", "kustomization.yaml"},
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
			expected:    []string{"base/app.yaml", "base/kustomization.yaml", "kustomization.yaml"},
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
			expected:    []string{"app.yaml", "base/app.yaml", "base/kustomization.yaml", "kustomization.yaml"},
			createFiles: map[string]string{
				"app.yaml":                "",
				"base/kustomization.yaml": `resources: [app.yaml]`,
				"base/app.yaml":           "",
			},
		},
		{
			description: "missing local kustomization resource",
			yaml:        `resources: [app.yaml, missing-or-remote-base]`,
			expected:    []string{"app.yaml", "kustomization.yaml"},
			createFiles: map[string]string{
				"app.yaml": "",
			},
		},
		{
			description: "mixed resource types",
			yaml:        `resources: [app.yaml, missing-or-remote-base, base]`,
			expected:    []string{"app.yaml", "base/app.yaml", "base/kustomization.yaml", "kustomization.yaml"},
			createFiles: map[string]string{
				"app.yaml":                "",
				"base/kustomization.yaml": `resources: [app.yaml]`,
				"base/app.yaml":           "",
			},
		},
		{
			description: "alt config name: kustomization.yml",
			yaml:        `resources: [app.yaml]`,
			expected:    []string{"app.yaml", "kustomization.yml"},
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
			expected:    []string{"Kustomization", "app.yaml", "base1/app.yaml", "base1/kustomization.yml", "base2/app.yaml", "base2/kustomization.yaml"},
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
								KustomizePaths: []string{tmpDir.Root()},
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
	tests := []struct {
		description string
		builds      []build.Artifact
		input       string
		expected    string
		shouldErr   bool
	}{
		{
			description: "should succeed without error",
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
			input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
  - image: gcr.io/project/image2
    name: image2
`,
			expected: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.
				CmdRunOut("kubectl version --client -ojson", kubectlVersion).
				AndRunOut("kustomize build .", test.input))
			t.NewTempDir().
				Chdir()

			k := NewKustomizeDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KustomizeDeploy: &latest.KustomizeDeploy{
								KustomizePaths: []string{"."},
							},
						},
					},
				},
				KubeContext: testKubeContext,
				Opts: config.SkaffoldOptions{
					Namespace: testNamespace,
				},
			})
			var b bytes.Buffer
			err := k.Render(context.Background(), &b, test.builds, "")
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, b.String())
		})
	}
}
