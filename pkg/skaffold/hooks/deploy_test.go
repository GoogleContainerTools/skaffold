/*
Copyright 2021 The Skaffold Authors

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

package hooks

import (
	"bytes"
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const testKubeContext = "context1"

func TestDeployHooks(t *testing.T) {
	testutil.Run(t, "TestDeployHooks", func(t *testutil.T) {
		hooks := latest.DeployHooks{
			PreHooks: []latest.DeployHookItem{
				{
					HostHook: &latest.HostHook{
						OS:      []string{"linux", "darwin"},
						Command: []string{"sh", "-c", "echo pre-hook running with SKAFFOLD_RUN_ID=$SKAFFOLD_RUN_ID,SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
					},
				},
				{
					HostHook: &latest.HostHook{
						OS:      []string{"windows"},
						Command: []string{"cmd.exe", "/C", "echo pre-hook running with SKAFFOLD_RUN_ID=%SKAFFOLD_RUN_ID%,SKAFFOLD_KUBE_CONTEXT=%SKAFFOLD_KUBE_CONTEXT%,SKAFFOLD_NAMESPACES=%SKAFFOLD_NAMESPACES%"},
					},
				},
				{
					ContainerHook: &latest.NamedContainerHook{
						ContainerHook: latest.ContainerHook{
							Command: []string{"foo", "pre-hook"},
						},
						PodName:       "pod1",
						ContainerName: "container1",
					},
				},
			},
			PostHooks: []latest.DeployHookItem{
				{
					HostHook: &latest.HostHook{
						OS:      []string{"linux", "darwin"},
						Command: []string{"sh", "-c", "echo post-hook running with SKAFFOLD_RUN_ID=$SKAFFOLD_RUN_ID,SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
					},
				},
				{
					HostHook: &latest.HostHook{
						OS:      []string{"windows"},
						Command: []string{"cmd.exe", "/C", "echo post-hook running with SKAFFOLD_RUN_ID=%SKAFFOLD_RUN_ID%,SKAFFOLD_KUBE_CONTEXT=%SKAFFOLD_KUBE_CONTEXT%,SKAFFOLD_NAMESPACES=%SKAFFOLD_NAMESPACES%"},
					},
				},
				{
					ContainerHook: &latest.NamedContainerHook{
						ContainerHook: latest.ContainerHook{
							Command: []string{"foo", "post-hook"},
						},
						PodName:       "pod1",
						ContainerName: "container1",
					},
				},
			},
		}
		preHostHookOut := "pre-hook running with SKAFFOLD_RUN_ID=run_id,SKAFFOLD_KUBE_CONTEXT=context1,SKAFFOLD_NAMESPACES=np1,np2"
		preContainerHookOut := "container pre-hook succeeded"
		postHostHookOut := "post-hook running with SKAFFOLD_RUN_ID=run_id,SKAFFOLD_KUBE_CONTEXT=context1,SKAFFOLD_NAMESPACES=np1,np2"
		postContainerHookOut := "container post-hook succeeded"

		deployer := latest.KubectlDeploy{
			LifecycleHooks: hooks,
		}
		namespaces := []string{"np1", "np2"}
		opts := NewDeployEnvOpts("run_id", testKubeContext, namespaces)
		formatter := func(corev1.Pod, corev1.ContainerStatus, func() bool) log.Formatter { return mockLogFormatter{} }
		runner := NewDeployRunner(&kubectl.CLI{KubeContext: testKubeContext}, deployer.LifecycleHooks, &namespaces, formatter, opts, nil)

		t.Override(&util.DefaultExecCommand,
			testutil.CmdRunWithOutput("kubectl --context context1 exec pod1 --namespace np1 -c container1 -- foo pre-hook", preContainerHookOut).
				AndRunWithOutput("kubectl --context context1 exec pod1 --namespace np1 -c container1 -- foo post-hook", postContainerHookOut))
		t.Override(&kubernetesclient.Client, fakeKubernetesClient)
		var preOut, postOut bytes.Buffer
		err := runner.RunPreHooks(context.Background(), &preOut)
		t.CheckNoError(err)
		t.CheckContains(preHostHookOut, preOut.String())
		t.CheckContains(preContainerHookOut, strings.TrimRight(preOut.String(), "\r\n"))
		err = runner.RunPostHooks(context.Background(), &postOut)
		t.CheckNoError(err)
		t.CheckContains(postHostHookOut, postOut.String())
		t.CheckContains(postContainerHookOut, postOut.String())
	})
}

func TestNewCloudRunDeployRunnerHooksMapping(t *testing.T) {
	tests := []struct {
		description   string
		expectedHooks latest.DeployHooks
		inputConfig   latest.CloudRunDeployHooks
	}{
		{
			description: "no hooks specified",
			expectedHooks: latest.DeployHooks{
				PreHooks:  []latest.DeployHookItem{},
				PostHooks: []latest.DeployHookItem{},
			},
			inputConfig: latest.CloudRunDeployHooks{},
		},
		{
			description: "pre hooks specified",
			expectedHooks: latest.DeployHooks{
				PreHooks: []latest.DeployHookItem{
					{
						HostHook: &latest.HostHook{
							Command: []string{"no-real-command-1 arg1 arg2"},
							OS:      []string{"darwin", "linux"},
							Dir:     ".",
						},
					},
					{
						HostHook: &latest.HostHook{
							Command: []string{"no-real-command-2 arg1 arg2"},
							OS:      []string{"linux", "darwin"},
							Dir:     "~/",
						},
					},
				},
				PostHooks: []latest.DeployHookItem{},
			},
			inputConfig: latest.CloudRunDeployHooks{
				PreHooks: []latest.HostHook{
					{
						Command: []string{"no-real-command-1 arg1 arg2"},
						OS:      []string{"darwin", "linux"},
						Dir:     ".",
					},
					{
						Command: []string{"no-real-command-2 arg1 arg2"},
						OS:      []string{"linux", "darwin"},
						Dir:     "~/",
					},
				},
			},
		},
		{
			description: "post hooks specified",
			expectedHooks: latest.DeployHooks{
				PreHooks: []latest.DeployHookItem{},
				PostHooks: []latest.DeployHookItem{
					{
						HostHook: &latest.HostHook{
							Command: []string{"no-real-command-1 arg1 arg2"},
							OS:      []string{"darwin", "linux"},
							Dir:     ".",
						},
					},
					{
						HostHook: &latest.HostHook{
							Command: []string{"no-real-command-2 arg1 arg2"},
							OS:      []string{"linux", "darwin"},
							Dir:     "~/",
						},
					},
				},
			},
			inputConfig: latest.CloudRunDeployHooks{
				PostHooks: []latest.HostHook{
					{
						Command: []string{"no-real-command-1 arg1 arg2"},
						OS:      []string{"darwin", "linux"},
						Dir:     ".",
					},
					{
						Command: []string{"no-real-command-2 arg1 arg2"},
						OS:      []string{"linux", "darwin"},
						Dir:     "~/",
					},
				},
			},
		},
		{
			description: "post and pre hooks specified",
			expectedHooks: latest.DeployHooks{
				PreHooks: []latest.DeployHookItem{
					{
						HostHook: &latest.HostHook{
							Command: []string{"pre-no-real-command-1 arg1 arg2"},
							OS:      []string{"darwin", "linux"},
							Dir:     ".",
						},
					},
				},
				PostHooks: []latest.DeployHookItem{
					{
						HostHook: &latest.HostHook{
							Command: []string{"post-no-real-command-1 arg1 arg2"},
							OS:      []string{"darwin", "linux"},
							Dir:     ".",
						},
					},
					{
						HostHook: &latest.HostHook{
							Command: []string{"post-no-real-command-2 arg1 arg2"},
							OS:      []string{"linux", "darwin"},
							Dir:     "~/",
						},
					},
				},
			},
			inputConfig: latest.CloudRunDeployHooks{
				PreHooks: []latest.HostHook{
					{
						Command: []string{"pre-no-real-command-1 arg1 arg2"},
						OS:      []string{"darwin", "linux"},
						Dir:     ".",
					},
				},
				PostHooks: []latest.HostHook{
					{
						Command: []string{"post-no-real-command-1 arg1 arg2"},
						OS:      []string{"darwin", "linux"},
						Dir:     ".",
					},
					{
						Command: []string{"post-no-real-command-2 arg1 arg2"},
						OS:      []string{"linux", "darwin"},
						Dir:     "~/",
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			runner, ok := (NewCloudRunDeployRunner(test.inputConfig, DeployEnvOpts{})).(deployRunner)
			t.CheckTrue(ok)
			t.CheckDeepEqual(runner.DeployHooks, test.expectedHooks)
		})
	}
}
