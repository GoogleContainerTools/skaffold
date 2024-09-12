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
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	kubectlR "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type gcsClientMock struct{}

func (g gcsClientMock) DownloadRecursive(ctx context.Context, src, dst string) error {
	return nil
}
func TestKubectlV1RenderDeploy(t *testing.T) {
	tests := []struct {
		description                 string
		generate                    latest.Generate
		kubectl                     latest.KubectlDeploy
		builds                      []graph.Artifact
		commands                    util.Command
		shouldErr                   bool
		forceDeploy                 bool
		waitForDeletions            bool
		skipSkaffoldNamespaceOption bool
		envs                        map[string]string
	}{
		{
			description:      "no manifest should error now since there is nothing to deploy",
			kubectl:          latest.KubectlDeploy{},
			shouldErr:        true,
			waitForDeletions: true,
			commands: testutil.
				CmdRunOutOnce("kubectl config view --minify -o jsonpath='{..namespace}'", "default"),
		},
		{
			description: "deploy success (disable validation)",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			kubectl: latest.KubectlDeploy{
				Flags: latest.KubectlFlags{
					DisableValidation: true,
				},
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --validate=false").
				AndRunOutOnce("kubectl config view --minify -o jsonpath='{..namespace}'", "testNamespace"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "deploy success (forced)",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f - --force --grace-period=0").
				AndRunOutOnce("kubectl config view --minify -o jsonpath='{..namespace}'", "testNamespace"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			forceDeploy:      true,
			waitForDeletions: true,
		},
		{
			description: "deploy success",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -").
				AndRunOutOnce("kubectl config view --minify -o jsonpath='{..namespace}'", "testNamespace"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "deploy success (kubectl v1.18)",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", "").
				AndRun("kubectl --context kubecontext --namespace testNamespace apply -f -").
				AndRunOutOnce("kubectl config view --minify -o jsonpath='{..namespace}'", "testNamespace"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions: true,
		},
		{
			description: "deploy success (default namespace)",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", "").
				AndRun("kubectl --context kubecontext apply -f -").
				AndRunOutOnce("kubectl config view --minify -o jsonpath='{..namespace}'", "default"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions:            true,
			skipSkaffoldNamespaceOption: true,
		},
		{
			description: "deploy success (default namespace with env template)",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			kubectl: latest.KubectlDeploy{
				DefaultNamespace: &TestNamespace2FromEnvTemplate,
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace2 get -f - --ignore-not-found -ojson", "").
				AndRun("kubectl --context kubecontext --namespace testNamespace2 apply -f -").
				AndRunOutOnce("kubectl config view --minify -o jsonpath='{..namespace}'", "testNamespace2"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			waitForDeletions:            true,
			skipSkaffoldNamespaceOption: true,
			envs: map[string]string{
				"MYENV": "Namesp",
			},
		},
		{
			description: "deploy command error",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace get -f - --ignore-not-found -ojson", "").
				AndRunErr("kubectl --context kubecontext --namespace testNamespace apply -f -", fmt.Errorf("")).
				AndRunOutOnce("kubectl config view --minify -o jsonpath='{..namespace}'", "testNamespace"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			shouldErr:        true,
			waitForDeletions: true,
		},
		{
			description: "additional flags",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			kubectl: latest.KubectlDeploy{
				Flags: latest.KubectlFlags{
					Global: []string{"-v=0"},
					Apply:  []string{"--overwrite=true"},
					Delete: []string{"ignored"},
				},
			},
			commands: testutil.
				CmdRunOut("kubectl --context kubecontext --namespace testNamespace get -v=0 -f - --ignore-not-found -ojson", "").
				AndRunErr("kubectl --context kubecontext --namespace testNamespace apply -v=0 --overwrite=true -f -", fmt.Errorf("")).
				AndRunOutOnce("kubectl config view --minify -o jsonpath='{..namespace}'", "testNamespace"),
			builds: []graph.Artifact{{
				ImageName: "leeroy-web",
				Tag:       "leeroy-web:v1",
			}},
			shouldErr:        true,
			waitForDeletions: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetEnvs(test.envs)
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&client.Client, deployutil.MockK8sClient)
			tmpDir := t.NewTempDir()
			tmpDir.Write("deployment.yaml", DeploymentWebYAML).
				Touch("empty.ignored").
				Chdir()

			skaffoldNamespaceOption := ""
			if !test.skipSkaffoldNamespaceOption {
				skaffoldNamespaceOption = TestNamespace
			}
			const configName = "default"
			k, err := NewDeployer(&kubectlConfig{
				workingDir: ".",
				force:      test.forceDeploy,
				waitForDeletions: config.WaitForDeletions{
					Enabled: test.waitForDeletions,
					Delay:   0 * time.Second,
					Max:     10 * time.Second,
				},
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{
					Namespace: skaffoldNamespaceOption}},
			}, &label.DefaultLabeller{}, &test.kubectl, nil, configName, nil)
			t.RequireNoError(err)

			rc := latest.RenderConfig{Generate: test.generate}
			mockCfg := &kubectlConfig{
				RunContext: runcontext.RunContext{
					WorkingDir: tmpDir.Root(),
					Pipelines: runcontext.NewPipelines(map[string]latest.Pipeline{
						configName: {
							Render: rc,
						},
					}, []string{configName}),
				},
			}

			r, err := kubectlR.New(mockCfg, rc, map[string]string{}, configName, skaffoldNamespaceOption, nil, !test.skipSkaffoldNamespaceOption)
			t.CheckNoError(err)
			var b bytes.Buffer
			m, errR := r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}},
				true)
			t.CheckNoError(errR)
			err = k.Deploy(context.Background(), io.Discard, test.builds, m)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKubectlCleanup(t *testing.T) {
	tests := []struct {
		description string
		generate    latest.Generate
		kubectl     latest.KubectlDeploy
		commands    util.Command
		shouldErr   bool
		dryRun      bool
	}{
		{
			description: "cleanup dry-run",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRun("kubectl --context kubecontext --namespace testNamespace delete --dry-run --ignore-not-found=true --wait=false -f -"),
			dryRun: true,
		},
		{
			description: "cleanup success",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -"),
		},
		{
			description: "cleanup success (kubectl v1.18)",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRun("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -"),
		},
		{
			description: "cleanup error",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			commands: testutil.
				CmdRunErr("kubectl --context kubecontext --namespace testNamespace delete --ignore-not-found=true --wait=false -f -", errors.New("BUG")),
			shouldErr: true,
		},
		{
			description: "additional flags",
			generate: latest.Generate{
				RawK8s: []string{"deployment.yaml"},
			},
			kubectl: latest.KubectlDeploy{
				Flags: latest.KubectlFlags{
					Global: []string{"-v=0"},
					Apply:  []string{"ignored"},
					Delete: []string{"--grace-period=1"},
				},
			},
			commands: testutil.
				CmdRun("kubectl --context kubecontext --namespace testNamespace delete -v=0 --grace-period=1 --ignore-not-found=true --wait=false -f -"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			tmpDir := t.NewTempDir()
			tmpDir.Write("deployment.yaml", DeploymentWebYAML).Chdir()
			const configName = "default"
			k, err := NewDeployer(&kubectlConfig{
				workingDir: ".",
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{Namespace: TestNamespace}},
			}, &label.DefaultLabeller{}, &test.kubectl, nil, configName, nil)
			t.RequireNoError(err)
			rc := latest.RenderConfig{Generate: test.generate}
			mockCfg := &kubectlConfig{
				RunContext: runcontext.RunContext{
					WorkingDir: tmpDir.Root(),
					Pipelines: runcontext.NewPipelines(map[string]latest.Pipeline{
						configName: {
							Render: rc,
						},
					}, []string{configName}),
				},
			}

			r, err := kubectlR.New(mockCfg, rc, map[string]string{}, configName, TestNamespace, nil, true)
			t.CheckNoError(err)
			var b bytes.Buffer
			m, errR := r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}},
				true)
			t.CheckNoError(errR)

			err = k.Cleanup(context.Background(), io.Discard, test.dryRun, m)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestKubectlRedeploy(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&client.Client, deployutil.MockK8sClient)
		t.NewTempDir().
			Write("deployment-web.yaml", DeploymentWebYAML).
			Write("deployment-app.yaml", DeploymentAppYAML)

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", "").
			AndRun("kubectl --context kubecontext apply -f -").
			AndRunOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", "").
			AndRun("kubectl --context kubecontext apply -f -"))

		const configName = "default"
		deployer, err := NewDeployer(&kubectlConfig{
			workingDir: ".",
			waitForDeletions: config.WaitForDeletions{
				Enabled: true,
				Delay:   0 * time.Millisecond,
				Max:     10 * time.Second},
		}, &label.DefaultLabeller{}, &latest.KubectlDeploy{}, nil, configName, nil)
		t.RequireNoError(err)

		// Deploy both manifests
		m, err := manifest.Load(bytes.NewReader([]byte(DeploymentAppYAMLv1)))
		m.Append([]byte(DeploymentWebYAMLv1))

		t.CheckNoError(err)

		manifestListByConfig := manifest.NewManifestListByConfig()
		manifestListByConfig.Add(configName, m)

		err = deployer.Deploy(context.Background(), io.Discard, []graph.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v1"},
		}, manifestListByConfig)
		t.CheckNoError(err)

		// Deploy one manifest since only one image is updated
		m, err = manifest.Load(bytes.NewReader([]byte(DeploymentAppYAMLv2)))
		t.CheckNoError(err)
		manifestListByConfig = manifest.NewManifestListByConfig()
		manifestListByConfig.Add(configName, m)
		err = deployer.Deploy(context.Background(), io.Discard, []graph.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		}, manifestListByConfig)
		t.CheckNoError(err)

		// Deploy zero manifest since no image is updated
		emptyManifestList := manifest.NewManifestListByConfig()
		err = deployer.Deploy(context.Background(), io.Discard, []graph.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
			{ImageName: "leeroy-app", Tag: "leeroy-app:v2"},
		}, emptyManifestList)
		t.CheckErrorContains("nothing to deploy", err)
	})
}

func TestKubectlWaitForDeletions(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&client.Client, deployutil.MockK8sClient)
		tmpDir := t.NewTempDir().Write("deployment-web.yaml", DeploymentWebYAML)

		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-app"}},
					{"metadata":{"name":"leeroy-front"}}
				]
			}`).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-app"}},
					{"metadata":{"name":"leeroy-front"}}
				]
			}`).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"name":"leeroy-front"}}
				]
			}`).
			AndRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, "").
			AndRunInput("kubectl --context kubecontext apply -f -", DeploymentWebYAMLv1),
		)
		const configName = "default"
		deployer, err := NewDeployer(&kubectlConfig{
			workingDir: tmpDir.Root(),
			waitForDeletions: config.WaitForDeletions{
				Enabled: true,
				Delay:   0 * time.Millisecond,
				Max:     10 * time.Second,
			},
		}, &label.DefaultLabeller{}, &latest.KubectlDeploy{}, nil, configName, nil)
		t.RequireNoError(err)

		var out bytes.Buffer
		m, err := manifest.Load(bytes.NewReader([]byte(DeploymentWebYAMLv1)))
		t.CheckNoError(err)

		manifestListByConfig := manifest.NewManifestListByConfig()
		manifestListByConfig.Add(configName, m)

		err = deployer.Deploy(context.Background(), &out, []graph.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		}, manifestListByConfig)

		t.CheckNoError(err)
	})
}

func TestKubectlWaitForDeletionsFails(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Write("deployment-web.yaml", DeploymentWebYAML)

		t.Override(&client.Client, deployutil.MockK8sClient)
		t.Override(&util.DefaultExecCommand, testutil.
			CmdRunInputOut("kubectl --context kubecontext get -f - --ignore-not-found -ojson", DeploymentWebYAMLv1, `{
				"items":[
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-web"}},
					{"metadata":{"deletionTimestamp":"2020-07-24T12:40:32Z","name":"leeroy-app"}}
				]
			}`),
		)

		const configName = "default"
		deployer, err := NewDeployer(&kubectlConfig{
			workingDir: tmpDir.Root(),
			waitForDeletions: config.WaitForDeletions{
				Enabled: true,
				Delay:   10 * time.Second,
				Max:     100 * time.Millisecond,
			},
		}, &label.DefaultLabeller{}, &latest.KubectlDeploy{}, nil, configName, nil)
		t.RequireNoError(err)

		m, err := manifest.Load(bytes.NewReader([]byte(DeploymentWebYAMLv1)))
		t.CheckNoError(err)

		manifestListByConfig := manifest.NewManifestListByConfig()
		manifestListByConfig.Add(configName, m)

		err = deployer.Deploy(context.Background(), io.Discard, []graph.Artifact{
			{ImageName: "leeroy-web", Tag: "leeroy-web:v1"},
		}, manifestListByConfig)

		t.CheckErrorContains(`2 resources failed to complete their deletion before a new deployment: "leeroy-web", "leeroy-app"`, err)
	})
}

func TestGCSManifests(t *testing.T) {
	tests := []struct {
		description string
		generate    latest.Generate
		commands    util.Command
		shouldErr   bool
	}{
		{
			description: "manifest from GCS",
			generate: latest.Generate{
				RawK8s: []string{"gs://dev/deployment.yaml"},
			},
			commands: testutil.
				CmdRun("kubectl --context kubecontext --namespace testNamespace apply -f -"),
		}}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&client.Client, deployutil.MockK8sClient)
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&manifest.GetGCSClient, func() manifest.GCSClient {
				return gcsClientMock{}
			})
			if err := os.MkdirAll(manifest.ManifestTmpDir, os.ModePerm); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(manifest.ManifestTmpDir+"/deployment.yaml", []byte(DeploymentWebYAML), os.ModePerm); err != nil {
				t.Fatal(err)
			}
			const configName = "default"
			rc := latest.RenderConfig{Generate: test.generate}
			mockCfg := &kubectlConfig{
				RunContext: runcontext.RunContext{
					Pipelines: runcontext.NewPipelines(map[string]latest.Pipeline{
						configName: {
							Render: latest.RenderConfig{Generate: test.generate},
						},
					}, []string{configName}),
				},
			}
			r, err := kubectlR.New(mockCfg, rc, map[string]string{}, configName, TestNamespace, nil, true)
			t.CheckNoError(err)
			var b bytes.Buffer
			m, errR := r.Render(context.Background(), &b, []graph.Artifact{{ImageName: "leeroy-web", Tag: "leeroy-web:v1"}},
				true)
			t.CheckNoError(errR)

			k, err := NewDeployer(&kubectlConfig{
				workingDir: ".",
				RunContext: runcontext.RunContext{Opts: config.SkaffoldOptions{Namespace: TestNamespace}},
			}, &label.DefaultLabeller{}, &latest.KubectlDeploy{}, nil, configName, nil)
			t.RequireNoError(err)

			err = k.Deploy(context.Background(), io.Discard, nil, m)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestHasRunnableHooks(t *testing.T) {
	tests := []struct {
		description string
		cfg         latest.KubectlDeploy
		expected    bool
	}{
		{
			description: "no hooks defined",
			cfg:         latest.KubectlDeploy{},
		},
		{
			description: "has pre-deploy hook defined",
			cfg: latest.KubectlDeploy{
				LifecycleHooks: latest.DeployHooks{PreHooks: []latest.DeployHookItem{{}}},
			},
			expected: true,
		},
		{
			description: "has post-deploy hook defined",
			cfg: latest.KubectlDeploy{
				LifecycleHooks: latest.DeployHooks{PostHooks: []latest.DeployHookItem{{}}},
			},
			expected: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			k, err := NewDeployer(&kubectlConfig{}, &label.DefaultLabeller{}, &test.cfg, nil, "default", nil)
			t.RequireNoError(err)
			actual := k.HasRunnableHooks()
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

type kubectlConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	workingDir            string
	defaultRepo           string
	multiLevelRepo        *bool
	force                 bool
	waitForDeletions      config.WaitForDeletions
}

func (c *kubectlConfig) GetKubeContext() string                              { return "kubecontext" }
func (c *kubectlConfig) GetKubeNamespace() string                            { return c.Opts.Namespace }
func (c *kubectlConfig) WorkingDir() string                                  { return c.workingDir }
func (c *kubectlConfig) ForceDeploy() bool                                   { return c.force }
func (c *kubectlConfig) DefaultRepo() *string                                { return &c.defaultRepo }
func (c *kubectlConfig) MultiLevelRepo() *bool                               { return c.multiLevelRepo }
func (c *kubectlConfig) WaitForDeletions() config.WaitForDeletions           { return c.waitForDeletions }
func (c *kubectlConfig) PortForwardResources() []*latest.PortForwardResource { return nil }
func (c *kubectlConfig) GetWorkingDir() string                               { return c.workingDir }
func (c *kubectlConfig) TransformAllowList() []latest.ResourceFilter         { return nil }
func (c *kubectlConfig) TransformDenyList() []latest.ResourceFilter          { return nil }
func (c *kubectlConfig) TransformRulesFile() string                          { return "" }
