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
	"testing"

	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestRenderHooks(t *testing.T) {
	testutil.Run(t, "TestRenderHooks", func(t *testutil.T) {
		hooks := latest.RenderHooks{
			PreHooks: []latest.RenderHookItem{
				{
					HostHook: &latest.HostHook{
						OS:      []string{"linux", "darwin"},
						Command: []string{"sh", "-c", "echo pre-hook running with SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
					},
				},
				{
					HostHook: &latest.HostHook{
						OS:      []string{"windows"},
						Command: []string{"cmd.exe", "/C", "echo pre-hook running with SKAFFOLD_KUBE_CONTEXT=%SKAFFOLD_KUBE_CONTEXT%,SKAFFOLD_NAMESPACES=%SKAFFOLD_NAMESPACES%"},
					},
				},
			},
			PostHooks: []latest.RenderHookItem{
				{
					HostHook: &latest.HostHook{
						OS:      []string{"linux", "darwin"},
						Command: []string{"sh", "-c", "echo post-hook running with SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
					},
				},
				{
					HostHook: &latest.HostHook{
						OS:      []string{"windows"},
						Command: []string{"cmd.exe", "/C", "echo post-hook running with SKAFFOLD_KUBE_CONTEXT=%SKAFFOLD_KUBE_CONTEXT%,SKAFFOLD_NAMESPACES=%SKAFFOLD_NAMESPACES%"},
					},
				},
			},
		}
		preHostHookOut := "pre-hook running with SKAFFOLD_KUBE_CONTEXT=context1,SKAFFOLD_NAMESPACES=np1,np2"
		postHostHookOut := "post-hook running with SKAFFOLD_KUBE_CONTEXT=context1,SKAFFOLD_NAMESPACES=np1,np2"
		namespaces := []string{"np1", "np2"}
		opts := NewRenderEnvOpts(testKubeContext, namespaces)
		runner := newRenderRunner(hooks, &namespaces, opts)
		t.Override(&kubernetesclient.Client, fakeKubernetesClient)
		var preOut, postOut bytes.Buffer
		err := runner.RunPreHooks(context.Background(), &preOut)
		t.CheckNoError(err)
		t.CheckContains(preHostHookOut, preOut.String())
		err = runner.RunPostHooks(context.Background(), &postOut)
		t.CheckNoError(err)
		t.CheckContains(postHostHookOut, postOut.String())
	})
}
