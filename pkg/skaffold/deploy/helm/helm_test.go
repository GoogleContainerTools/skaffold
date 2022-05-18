/*
Copyright 2022 The Skaffold Authors

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

package helm

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/hooks"
	ctl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/blang/semver"
)

func TestBin(t *testing.T) {
	tests := []struct {
		description string
		helmVersion string
		expected    string
		shouldErr   bool
	}{
		{"Helm 2.0RC1", version20rc, "2.0.0-rc.1", false},
		{"Helm 2.15.1", version21, "2.15.1", false},
		{"Helm 3.0b3", version30b, "3.0.0-beta.3", false},
		{"Helm 3.0", version30, "3.0.0", false},
		{"Helm 3.1.1", version31, "3.1.1", false},
		{"Helm 3.5.2 without leading 'v'", version35, "3.5.2", false},
		{"Custom Helm 3.3 build from Manjaro", "v3.3", "3.3.0", false}, // not semver compliant
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", test.helmVersion))
			ver, err := binVer(context.Background())

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, ver.String())
		})
	}
}

func TestHelmCleanup(t *testing.T) {
	tests := []struct {
		description      string
		commands         util.Command
		helm             latest.HelmDeploy
		namespace        string
		builds           []graph.Artifact
		expectedWarnings []string
		dryRun           bool
	}{
		{
			description: "helm3 cleanup dry-run",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext get manifest skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
			dryRun: true,
		},
		{
			description: "helm3 cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --kubeconfig kubeconfig"),
			helm:   testDeployConfig,
			builds: testBuilds,
		},
		{
			description: "helm3 namespace cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testReleaseNamespace --kubeconfig kubeconfig"),
			helm:   testDeployNamespacedConfig,
			builds: testBuilds,
		},
		{
			description: "helm3 namespace (with env template) cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testReleaseFOOBARNamespace --kubeconfig kubeconfig"),
			helm:   testDeployEnvTemplateNamespacedConfig,
			builds: testBuilds,
		},
		{
			description: "helm3 namespaced context cleanup success",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testNamespace --kubeconfig kubeconfig"),
			helm:      testDeployConfig,
			namespace: kubectl.TestNamespace,
			builds:    testBuilds,
		},
		{
			description: "helm3 namespaced context cleanup success overriding release namespace",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext delete skaffold-helm --namespace testNamespace --kubeconfig kubeconfig"),
			helm:      testDeployNamespacedConfig,
			namespace: kubectl.TestNamespace,
			builds:    testBuilds,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			fakeWarner := &warnings.Collect{}
			t.Override(&warnings.Printf, fakeWarner.Warnf)
			t.Override(&util.OSEnviron, func() []string { return []string{"FOO=FOOBAR"} })
			t.Override(&util.DefaultExecCommand, test.commands)

			deployer, err := NewDeployer(context.Background(), &helmConfig{
				namespace: test.namespace,
			}, &label.DefaultLabeller{}, &test.helm)
			t.RequireNoError(err)

			deployer.Cleanup(context.Background(), ioutil.Discard, test.dryRun)

			t.CheckDeepEqual(test.expectedWarnings, fakeWarner.Warnings)
		})
	}
}

func TestParseHelmRelease(t *testing.T) {
	tests := []struct {
		description string
		yaml        []byte
		shouldErr   bool
	}{
		{
			description: "parse valid deployment yaml",
			yaml:        []byte(validDeployYaml),
		},
		{
			description: "parse valid service yaml",
			yaml:        []byte(validServiceYaml),
		},
		{
			description: "parse invalid deployment yaml",
			yaml:        []byte(invalidDeployYaml),
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := parseRuntimeObject(kubectl.TestNamespace, test.yaml)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestWriteBuildArtifacts(t *testing.T) {
	tests := []struct {
		description string
		builds      []graph.Artifact
		result      string
	}{
		{
			description: "nil",
			builds:      nil,
			result:      `{"builds":null}`,
		},
		{
			description: "empty",
			builds:      []graph.Artifact{},
			result:      `{"builds":[]}`,
		},
		{
			description: "multiple images with tags",
			builds:      []graph.Artifact{{ImageName: "name", Tag: "name:tag"}, {ImageName: "name2", Tag: "name2:tag"}},
			result:      `{"builds":[{"imageName":"name","tag":"name:tag"},{"imageName":"name2","tag":"name2:tag"}]}`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			file, cleanup, err := writeBuildArtifacts(test.builds)
			t.CheckError(false, err)
			if content, err := ioutil.ReadFile(file); err != nil {
				t.Errorf("error reading file %q: %v", file, err)
			} else {
				t.CheckDeepEqual(test.result, string(content))
			}
			cleanup()
		})
	}
}

func TestGenerateSkaffoldDebugFilter(t *testing.T) {
	tests := []struct {
		description string
		buildFile   string
		result      []string
	}{
		{
			description: "empty buildfile is skipped",
			buildFile:   "",
			result:      []string{"filter", "--debugging", "--kube-context", "kubecontext", "--kubeconfig", "kubeconfig"},
		},
		{
			description: "buildfile is added",
			buildFile:   "buildfile",
			result:      []string{"filter", "--debugging", "--kube-context", "kubecontext", "--build-artifacts", "buildfile", "--kubeconfig", "kubeconfig"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version31))
			h, err := NewDeployer(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &testDeployConfig)
			h3 := h.(*Deployer31)
			t.RequireNoError(err)
			result := h3.generateSkaffoldDebugFilter(test.buildFile)
			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestHelmHooks(t *testing.T) {
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
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version31))
			t.Override(&hooks.NewDeployRunner, func(*ctl.CLI, latest.DeployHooks, *[]string, logger.Formatter, hooks.DeployEnvOpts) hooks.Runner {
				return test.runner
			})
			v, err := semver.New("3.0.0")
			t.RequireNoError(err)
			k, err := NewDeployer30(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &testDeployConfig, *v)
			t.RequireNoError(err)
			err = k.PreDeployHooks(context.Background(), ioutil.Discard)
			t.CheckError(test.shouldErr, err)
			err = k.PostDeployHooks(context.Background(), ioutil.Discard)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestHasRunnableHooks(t *testing.T) {
	tests := []struct {
		description string
		cfg         latest.HelmDeploy
		expected    bool
	}{
		{
			description: "no hooks defined",
			cfg:         latest.HelmDeploy{},
		},
		{
			description: "has pre-deploy hook defined",
			cfg: latest.HelmDeploy{
				LifecycleHooks: latest.DeployHooks{PreHooks: []latest.DeployHookItem{{}}},
			},
			expected: true,
		},
		{
			description: "has post-deploy hook defined",
			cfg: latest.HelmDeploy{
				LifecycleHooks: latest.DeployHooks{PostHooks: []latest.DeployHookItem{{}}},
			},
			expected: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version31))
			v, err := semver.New("3.0.0")
			t.RequireNoError(err)
			k, err := NewDeployer30(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &test.cfg, *v)
			t.RequireNoError(err)
			actual := k.HasRunnableHooks()
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestHelmDependencies(t *testing.T) {
	tests := []struct {
		description           string
		files                 []string
		valuesFiles           []string
		skipBuildDependencies bool
		remote                bool
		expected              func(folder *testutil.TempDir) []string
	}{
		{
			description:           "charts download dir and lock files are included when skipBuildDependencies is true",
			files:                 []string{"Chart.yaml", "Chart.lock", "charts/xyz.tar", "tmpcharts/xyz.tar", "templates/deploy.yaml"},
			skipBuildDependencies: true,
			expected: func(folder *testutil.TempDir) []string {
				return []string{
					folder.Path("Chart.lock"),
					folder.Path("Chart.yaml"),
					folder.Path("charts/xyz.tar"),
					folder.Path("templates/deploy.yaml"),
					folder.Path("tmpcharts/xyz.tar"),
				}
			},
		},
		{
			description:           "charts download dir and lock files are excluded when skipBuildDependencies is false",
			files:                 []string{"Chart.yaml", "Chart.lock", "charts/xyz.tar", "tmpcharts/xyz.tar", "templates/deploy.yaml"},
			skipBuildDependencies: false,
			expected: func(folder *testutil.TempDir) []string {
				return []string{
					folder.Path("Chart.yaml"),
					folder.Path("templates/deploy.yaml"),
				}
			},
		},
		{
			description:           "values file is included",
			skipBuildDependencies: false,
			files:                 []string{"Chart.yaml"},
			valuesFiles:           []string{"/folder/values.yaml"},
			expected: func(folder *testutil.TempDir) []string {
				return []string{"/folder/values.yaml", folder.Path("Chart.yaml")}
			},
		},
		{
			description:           "no deps for remote chart path",
			skipBuildDependencies: false,
			files:                 []string{"Chart.yaml"},
			remote:                true,
			expected: func(folder *testutil.TempDir) []string {
				return nil
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version30))
			tmpDir := t.NewTempDir().Touch(test.files...)
			var local, remote string
			if test.remote {
				remote = "foo/bar"
			} else {
				local = tmpDir.Root()
			}

			deployer, err := NewDeployer(context.Background(), &helmConfig{}, &label.DefaultLabeller{}, &latest.HelmDeploy{
				Releases: []latest.HelmRelease{{
					Name:                  "skaffold-helm",
					ChartPath:             local,
					RemoteChart:           remote,
					ValuesFiles:           test.valuesFiles,
					ArtifactOverrides:     map[string]string{"image": "skaffold-helm"},
					Overrides:             schemautil.HelmOverrides{Values: map[string]interface{}{"foo": "bar"}},
					SetValues:             map[string]string{"some.key": "somevalue"},
					SkipBuildDependencies: test.skipBuildDependencies,
				}},
			})
			t.RequireNoError(err)
			deps, err := deployer.Dependencies()

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expected(tmpDir), deps)
		})
	}
}

func TestImageSetFromConfig(t *testing.T) {
	tests := []struct {
		description string
		valueName   string
		tag         string
		expected    string
		strategy    *latest.HelmConventionConfig
		shouldErr   bool
	}{
		{
			description: "Helm set values with no convention config",
			valueName:   "image",
			tag:         "skaffold-helm:1.0.0",
			expected:    "image=skaffold-helm:1.0.0",
			strategy:    nil,
			shouldErr:   false,
		},
		{
			description: "Helm set values with helm conventions",
			valueName:   "image",
			tag:         "skaffold-helm:1.0.0",
			expected:    "image.repository=skaffold-helm,image.tag=1.0.0",
			strategy:    &latest.HelmConventionConfig{},
			shouldErr:   false,
		},
		{
			description: "Helm set values with helm conventions and explicit registry value",
			valueName:   "image",
			tag:         "docker.io/skaffold-helm:1.0.0",
			expected:    "image.registry=docker.io,image.repository=skaffold-helm,image.tag=1.0.0",
			strategy: &latest.HelmConventionConfig{
				ExplicitRegistry: true,
			},
			shouldErr: false,
		},
		{
			description: "Invalid tag with helm conventions",
			valueName:   "image",
			tag:         "skaffold-helm:1.0.0,0",
			expected:    "",
			strategy:    &latest.HelmConventionConfig{},
			shouldErr:   true,
		},
		{
			description: "Helm set values with helm conventions and explicit registry value, but missing in tag",
			valueName:   "image",
			tag:         "skaffold-helm:1.0.0",
			expected:    "",
			strategy: &latest.HelmConventionConfig{
				ExplicitRegistry: true,
			},
			shouldErr: true,
		},
		{
			description: "Helm set values using digest",
			valueName:   "image",
			tag:         "skaffold-helm:stable@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
			expected:    "image.repository=skaffold-helm,image.tag=stable@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
			strategy:    &latest.HelmConventionConfig{},
			shouldErr:   false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			values, err := imageSetFromConfig(test.strategy, test.valueName, test.tag)
			t.CheckError(test.shouldErr, err)
			t.CheckDeepEqual(test.expected, values)
		})
	}
}

func TestHelmRender(t *testing.T) {
	tests := []struct {
		description string
		shouldErr   bool
		commands    util.Command
		helm        latest.HelmDeploy
		env         []string
		outputFile  string
		expected    string
		builds      []graph.Artifact
		namespace   string
	}{
		{
			description: "normal render v3",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --kubeconfig kubeconfig"),
			helm: testDeployConfig,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render to a file",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRunWithOutput("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --kubeconfig kubeconfig",
					"Dummy Output"),
			helm:       testDeployConfig,
			outputFile: "dummy.yaml",
			expected:   "Dummy Output\n",
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with templated config",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set image.name=skaffold-helm --set image.tag=skaffold-helm:tag1 --set missing.key=<MISSING> --set other.key=FOOBAR --set some.key=somevalue --set FOOBAR=somevalue --kubeconfig kubeconfig"),
			helm: testDeployConfigTemplated,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with templated values file",
			shouldErr:   false,
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 -f /some/file-FOOBAR.yaml --kubeconfig kubeconfig"),
			helm: testDeployConfigValuesFilesTemplated,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with missing templated release name should fail",
			shouldErr:   true,
			commands:    testutil.CmdRunWithOutput("helm version --client", version31),
			helm:        testDeployWithTemplatedName,
			builds:      testBuilds,
		},
		{
			description: "render with templated release name",
			env:         []string{"USER=user"},
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template user-skaffold-helm examples/test --set-string image.tag=docker.io:5000/skaffold-helm:3605e7bc17cf46e53f4d81c4cbc24e5b4c495184 --set some.key=somevalue --kubeconfig kubeconfig"),
			helm:   testDeployWithTemplatedName,
			builds: testBuilds,
		},
		{
			description: "render with namespace",
			shouldErr:   false,
			commands: testutil.CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --namespace testReleaseNamespace --kubeconfig kubeconfig"),
			helm: testDeployNamespacedConfig,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with namespace",
			shouldErr:   false,
			commands: testutil.CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --namespace testReleaseFOOBARNamespace --kubeconfig kubeconfig"),
			helm: testDeployEnvTemplateNamespacedConfig,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with remote repo",
			shouldErr:   false,
			commands: testutil.CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --repo https://charts.helm.sh/stable --kubeconfig kubeconfig"),
			helm: testDeployConfigRemoteRepo,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with remote chart",
			shouldErr:   false,
			commands: testutil.CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm-remote stable/chartmuseum --repo https://charts.helm.sh/stable --kubeconfig kubeconfig"),
			helm: testDeployRemoteChart,
		},
		{
			description: "render with remote chart with version",
			shouldErr:   false,
			commands: testutil.CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm-remote stable/chartmuseum --version 1.0.0 --repo https://charts.helm.sh/stable --kubeconfig kubeconfig"),
			helm: testDeployRemoteChartVersion,
		},
		{
			description: "render with cli namespace",
			shouldErr:   false,
			namespace:   "clinamespace",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --namespace clinamespace --kubeconfig kubeconfig"),
			helm: testDeployConfig,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
		{
			description: "render with HelmRelease.Namespace and cli namespace",
			shouldErr:   false,
			namespace:   "clinamespace",
			commands: testutil.
				CmdRunWithOutput("helm version --client", version31).
				AndRun("helm --kube-context kubecontext template skaffold-helm examples/test --set-string image=skaffold-helm:tag1 --set some.key=somevalue --namespace clinamespace --kubeconfig kubeconfig"),
			helm: testDeployNamespacedConfig,
			builds: []graph.Artifact{
				{
					ImageName: "skaffold-helm",
					Tag:       "skaffold-helm:tag1",
				}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			file := ""
			if test.outputFile != "" {
				file = t.NewTempDir().Path(test.outputFile)
			}

			t.Override(&util.OSEnviron, func() []string { return append([]string{"FOO=FOOBAR"}, test.env...) })
			t.Override(&util.DefaultExecCommand, test.commands)

			deployer, err := NewDeployer(context.Background(), &helmConfig{
				namespace: test.namespace,
			}, &label.DefaultLabeller{}, &test.helm)
			t.RequireNoError(err)
			err = deployer.Render(context.Background(), ioutil.Discard, test.builds, true, file)
			t.CheckError(test.shouldErr, err)

			if file != "" {
				dat, _ := ioutil.ReadFile(file)
				t.CheckDeepEqual(string(dat), test.expected)
			}
		})
	}
}
