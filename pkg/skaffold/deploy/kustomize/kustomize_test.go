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

package kustomize

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/hooks"
	ctl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestKustomizeDeploy(t *testing.T) {
	tests := []struct {
		description                 string
		kustomize                   latest.KustomizeDeploy
		builds                      []graph.Artifact
		commands                    util.Command
		shouldErr                   bool
		forceDeploy                 bool
		skipSkaffoldNamespaceOption bool
		kustomizeCmdPresent         bool
		envs                        map[string]string
	}{
		{
			description: "no manifest",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: []string{"."},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectl.KubectlVersion118).
				AndRunOut("kustomize build .", ""),
			kustomizeCmdPresent: true,
		},
		{
			description: "deploy success",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: []string{"."},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectl.KubectlVersion118).
				AndRunOut("kustomize build .", kubectl.DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", kubectl.DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy:         true,
			kustomizeCmdPresent: true,
		},
		{
			description: "deploy success (default namespace)",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths:   []string{"."},
				DefaultNamespace: &kubectl.TestNamespace2,
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectl.KubectlVersion112).
				AndRunOut("kustomize build .", kubectl.DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace2 get -f - --ignore-not-found -ojson", kubectl.DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace2 apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy:                 true,
			skipSkaffoldNamespaceOption: true,
			kustomizeCmdPresent:         true,
		},
		{
			description: "deploy success (default namespace with env template)",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths:   []string{"."},
				DefaultNamespace: &kubectl.TestNamespace2FromEnvTemplate,
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectl.KubectlVersion112).
				AndRunOut("kustomize build .", kubectl.DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace2 get -f - --ignore-not-found -ojson", kubectl.DeploymentWebYAMLv1, "").
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
			kustomizeCmdPresent: true,
		},
		{
			description: "deploy success (kustomizePaths with env template)",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: []string{"/a/b/{{ .MYENV }}"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectl.KubectlVersion118).
				AndRunOut("kustomize build /a/b/c", kubectl.DeploymentWebYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", kubectl.DeploymentWebYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy: true,
			envs: map[string]string{
				"MYENV": "c",
			},
			kustomizeCmdPresent: true,
		},
		{
			description: "deploy success with multiple kustomizations",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: []string{"a", "b"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectl.KubectlVersion118).
				AndRunOut("kustomize build a", kubectl.DeploymentWebYAML).
				AndRunOut("kustomize build b", kubectl.DeploymentAppYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", kubectl.DeploymentWebYAMLv1+"\n---\n"+kubectl.DeploymentAppYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:v1",
				},
				{
					ImageName: "leeroy-app",
					Tag:       "leeroy-app:v1",
				},
			},
			forceDeploy:         true,
			kustomizeCmdPresent: true,
		},
		{
			description: "built-in kubectl kustomize",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: []string{"a", "b"},
			},
			commands: testutil.
				CmdRunOut("kubectl version --client -ojson", kubectl.KubectlVersion118).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace kustomize a", kubectl.DeploymentWebYAML).
				AndRunOut("kubectl --context kubecontext --namespace testNamespace kustomize b", kubectl.DeploymentAppYAML).
				AndRunInputOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", kubectl.DeploymentWebYAMLv1+"\n---\n"+kubectl.DeploymentAppYAMLv1, "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0"),
			builds: []graph.Artifact{
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
			t.SetEnvs(test.envs)
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&client.Client, deployutil.MockK8sClient)
			t.Override(&KustomizeBinaryCheck, func() bool { return test.kustomizeCmdPresent })
			t.NewTempDir().
				Chdir()

			skaffoldNamespaceOption := ""
			if !test.skipSkaffoldNamespaceOption {
				skaffoldNamespaceOption = kubectl.TestNamespace
			}

			k, err := NewDeployer(&kustomizeConfig{
				workingDir: ".",
				force:      test.forceDeploy,
				waitForDeletions: config.WaitForDeletions{
					Enabled: true,
					Delay:   0 * time.Second,
					Max:     10 * time.Second,
				},
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{
					Namespace: skaffoldNamespaceOption,
				}}}, &label.DefaultLabeller{}, &test.kustomize, "default")
			t.RequireNoError(err)
			err = k.Deploy(context.Background(), io.Discard, test.builds, manifest.ManifestListByConfig{})

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKustomizeCleanup(t *testing.T) {
	tmpDir := testutil.NewTempDir(t)

	tests := []struct {
		description string
		kustomize   latest.KustomizeDeploy
		commands    util.Command
		shouldErr   bool
		dryRun      bool
	}{
		{
			description: "cleanup dry-run",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: []string{tmpDir.Root()},
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), kubectl.DeploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --dry-run --ignore-not-found=true --wait=false -f -"),
			dryRun: true,
		},
		{
			description: "cleanup success",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: []string{tmpDir.Root()},
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), kubectl.DeploymentWebYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -"),
		},
		{
			description: "cleanup success with multiple kustomizations",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: tmpDir.Paths("a", "b"),
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Path("a"), kubectl.DeploymentWebYAML).
				AndRunOut("kustomize build "+tmpDir.Path("b"), kubectl.DeploymentAppYAML).
				AndRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -"),
		},
		{
			description: "cleanup error",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: []string{tmpDir.Root()},
			},
			commands: testutil.
				CmdRunOut("kustomize build "+tmpDir.Root(), kubectl.DeploymentWebYAML).
				AndRunErr("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "fail to read manifests",
			kustomize: latest.KustomizeDeploy{
				KustomizePaths: []string{tmpDir.Root()},
			},
			commands: testutil.
				CmdRunOutErr("kustomize build "+tmpDir.Root(), "", errors.New("BUG")),
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&KustomizeBinaryCheck, func() bool { return true })

			k, err := NewDeployer(&kustomizeConfig{
				workingDir: tmpDir.Root(),
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{
					Namespace: kubectl.TestNamespace}},
			}, &label.DefaultLabeller{}, &test.kustomize, "default")
			t.RequireNoError(err)
			err = k.Cleanup(context.Background(), io.Discard, test.dryRun, manifest.NewManifestListByConfig())

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKustomizeHooks(t *testing.T) {
	tests := []struct {
		description string
		runner      hooks.Runner
		shouldErr   bool
	}{
		{
			description: "hooks run successfully",
			runner: hooks.MockRunner{
				PreHooks: func(context.Context, io.Writer) error {
					return nil
				},
				PostHooks: func(context.Context, io.Writer) error {
					return nil
				},
			},
		},
		{
			description: "hooks fails",
			runner: hooks.MockRunner{
				PreHooks: func(context.Context, io.Writer) error {
					return errors.New("failed to execute hooks")
				},
				PostHooks: func(context.Context, io.Writer) error {
					return errors.New("failed to execute hooks")
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&KustomizeBinaryCheck, func() bool { return true })
			t.Override(&hooks.NewDeployRunner, func(*ctl.CLI, latest.DeployHooks, *[]string, logger.Formatter, hooks.DeployEnvOpts) hooks.Runner {
				return test.runner
			})

			k, err := NewDeployer(&kustomizeConfig{
				workingDir: ".",
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{
					Namespace: kubectl.TestNamespace}},
			}, &label.DefaultLabeller{}, &latest.KustomizeDeploy{}, "default")
			t.RequireNoError(err)
			err = k.PreDeployHooks(context.Background(), io.Discard)
			t.CheckError(test.shouldErr, err)
			err = k.PostDeployHooks(context.Background(), io.Discard)
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

			k, err := NewDeployer(&kustomizeConfig{}, &label.DefaultLabeller{}, &latest.KustomizeDeploy{KustomizePaths: kustomizePaths}, "default")
			t.RequireNoError(err)

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
			expectedArgs:  nil,
		},
		{
			description:   "One BuildArg, empty KustomizePaths",
			buildArgs:     []string{"--foo"},
			kustomizePath: "",
			expectedArgs:  []string{"--foo"},
		},
		{
			description:   "no BuildArgs, non-empty KustomizePaths",
			buildArgs:     []string{},
			kustomizePath: "foo",
			expectedArgs:  []string{"foo"},
		},
		{
			description:   "One BuildArg, non-empty KustomizePaths",
			buildArgs:     []string{"--foo"},
			kustomizePath: "bar",
			expectedArgs:  []string{"--foo", "bar"},
		},
		{
			description:   "Multiple BuildArg, empty KustomizePaths",
			buildArgs:     []string{"--foo", "--bar"},
			kustomizePath: "",
			expectedArgs:  []string{"--foo", "--bar"},
		},
		{
			description:   "Multiple BuildArg with spaces, empty KustomizePaths",
			buildArgs:     []string{"--foo bar", "--baz"},
			kustomizePath: "",
			expectedArgs:  []string{"--foo", "bar", "--baz"},
		},
		{
			description:   "Multiple BuildArg with spaces, non-empty KustomizePaths",
			buildArgs:     []string{"--foo bar", "--baz"},
			kustomizePath: "barfoo",
			expectedArgs:  []string{"--foo", "bar", "--baz", "barfoo"},
		},
		{
			description:   "Multiple BuildArg no spaces, non-empty KustomizePaths",
			buildArgs:     []string{"--foo", "bar", "--baz"},
			kustomizePath: "barfoo",
			expectedArgs:  []string{"--foo", "bar", "--baz", "barfoo"},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			args := BuildCommandArgs(test.buildArgs, test.kustomizePath)
			t.CheckDeepEqual(test.expectedArgs, args)
		})
	}
}

func TestHasRunnableHooks(t *testing.T) {
	tests := []struct {
		description string
		cfg         latest.KustomizeDeploy
		expected    bool
	}{
		{
			description: "no hooks defined",
			cfg:         latest.KustomizeDeploy{},
		},
		{
			description: "has pre-deploy hook defined",
			cfg: latest.KustomizeDeploy{
				LifecycleHooks: latest.DeployHooks{PreHooks: []latest.DeployHookItem{{}}},
			},
			expected: true,
		},
		{
			description: "has post-deploy hook defined",
			cfg: latest.KustomizeDeploy{
				LifecycleHooks: latest.DeployHooks{PostHooks: []latest.DeployHookItem{{}}},
			},
			expected: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			k, err := NewDeployer(&kustomizeConfig{}, &label.DefaultLabeller{}, &test.cfg, "default")
			t.RequireNoError(err)
			actual := k.HasRunnableHooks()
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

type kustomizeConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	force                 bool
	workingDir            string
	waitForDeletions      config.WaitForDeletions
}

func (c *kustomizeConfig) ForceDeploy() bool                                   { return c.force }
func (c *kustomizeConfig) WaitForDeletions() config.WaitForDeletions           { return c.waitForDeletions }
func (c *kustomizeConfig) WorkingDir() string                                  { return c.workingDir }
func (c *kustomizeConfig) GetKubeContext() string                              { return kubectl.TestKubeContext }
func (c *kustomizeConfig) GetKubeNamespace() string                            { return c.Opts.Namespace }
func (c *kustomizeConfig) PortForwardResources() []*latest.PortForwardResource { return nil }
