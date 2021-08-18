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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDeployHooks(t *testing.T) {
	testutil.Run(t, "TestDeployHooks", func(t *testutil.T) {
		hooks := v1.DeployHooks{
			PreHooks: []v1.DeployHookItem{
				{
					HostHook: &v1.HostHook{
						OS:      []string{"linux", "darwin"},
						Command: []string{"sh", "-c", "echo pre-hook running with SKAFFOLD_RUN_ID=$SKAFFOLD_RUN_ID,SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
					},
				},
				{
					HostHook: &v1.HostHook{
						OS:      []string{"windows"},
						Command: []string{"cmd.exe", "/C", "echo pre-hook running with SKAFFOLD_RUN_ID=%SKAFFOLD_RUN_ID%,SKAFFOLD_KUBE_CONTEXT=%SKAFFOLD_KUBE_CONTEXT%,SKAFFOLD_NAMESPACES=%SKAFFOLD_NAMESPACES%"},
					},
				},
				{
					ContainerHook: &v1.NamedContainerHook{
						ContainerHook: v1.ContainerHook{
							Command: []string{"foo", "pre-hook"},
						},
						PodName:       "pod1",
						ContainerName: "container1",
					},
				},
			},
			PostHooks: []v1.DeployHookItem{
				{
					HostHook: &v1.HostHook{
						OS:      []string{"linux", "darwin"},
						Command: []string{"sh", "-c", "echo post-hook running with SKAFFOLD_RUN_ID=$SKAFFOLD_RUN_ID,SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
					},
				},
				{
					HostHook: &v1.HostHook{
						OS:      []string{"windows"},
						Command: []string{"cmd.exe", "/C", "echo post-hook running with SKAFFOLD_RUN_ID=%SKAFFOLD_RUN_ID%,SKAFFOLD_KUBE_CONTEXT=%SKAFFOLD_KUBE_CONTEXT%,SKAFFOLD_NAMESPACES=%SKAFFOLD_NAMESPACES%"},
					},
				},
				{
					ContainerHook: &v1.NamedContainerHook{
						ContainerHook: v1.ContainerHook{
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

		deployer := v1.KubectlDeploy{
			LifecycleHooks: hooks,
		}
		namespaces := []string{"np1", "np2"}
		kubeContext := "context1"
		opts := NewDeployEnvOpts("run_id", kubeContext, namespaces)
		formatter := func(corev1.Pod, corev1.ContainerStatus, func() bool) log.Formatter { return mockLogFormatter{} }
		runner := NewDeployRunner(&kubectl.CLI{KubeContext: kubeContext}, deployer.LifecycleHooks, &namespaces, formatter, opts)

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
