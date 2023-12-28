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
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

const SimpleManifest = `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    test-name: before
spec:
  containers:
  - name: getting-started
    image: skaffold-example`

func TestRenderHooksLinux(t *testing.T) {
	tests := []struct {
		description     string
		hooks           latest.RenderHooks
		preHostHookOut  string
		postHostHookOut string
	}{
		{
			description: "test on linux machine match",
			hooks: latest.RenderHooks{
				PreHooks: []latest.RenderHookItem{
					{
						HostHook: &latest.HostHook{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo pre-hook running with SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
						},
					},
				},
				PostHooks: []latest.PostRenderHookItem{
					{
						HostHook: &latest.PostRenderHostHook{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo post-hook running with SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
						},
					},
				},
			},
			preHostHookOut:  "pre-hook running with SKAFFOLD_KUBE_CONTEXT=context1,SKAFFOLD_NAMESPACES=np1,np2",
			postHostHookOut: "post-hook running with SKAFFOLD_KUBE_CONTEXT=context1,SKAFFOLD_NAMESPACES=np1,np2",
		},
		{
			description: "test on linux non match",
			hooks: latest.RenderHooks{
				PreHooks: []latest.RenderHookItem{
					{
						HostHook: &latest.HostHook{
							OS:      []string{"windows"},
							Command: []string{"pwsh", "-Command", "echo", "pre-hook running with SKAFFOLD_KUBE_CONTEXT=$([Environment]::GetEnvironmentVariable('SKAFFOLD_KUBE_CONTEXT')),SKAFFOLD_NAMESPACES=$([Environment]::GetEnvironmentVariable('SKAFFOLD_NAMESPACES'))"},
						},
					},
				},
				PostHooks: []latest.PostRenderHookItem{
					{
						HostHook: &latest.PostRenderHostHook{
							OS:      []string{"windows"},
							Command: []string{"pwsh", "-Command", "echo", "post-hook running with SKAFFOLD_KUBE_CONTEXT=$([Environment]::GetEnvironmentVariable('SKAFFOLD_KUBE_CONTEXT')),SKAFFOLD_NAMESPACES=$([Environment]::GetEnvironmentVariable('SKAFFOLD_NAMESPACES'))"},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if runtime.GOOS == Windows {
				t.Skip("Tests require linux machine")
			}

			preOutFile, err := os.CreateTemp("", "")
			t.CheckNoError(err)
			t.Cleanup(func() {
				os.Remove(preOutFile.Name())
			})
			test.hooks.PreHooks[0].HostHook.Command[2] = test.hooks.PreHooks[0].HostHook.Command[2] + " > " + preOutFile.Name()
			postOutFile, err := os.CreateTemp("", "")
			t.CheckNoError(err)
			t.Cleanup(func() {
				os.Remove(postOutFile.Name())
			})
			test.hooks.PostHooks[0].HostHook.Command[2] = test.hooks.PostHooks[0].HostHook.Command[2] + " > " + postOutFile.Name()

			namespaces := []string{"np1", "np2"}
			opts := NewRenderEnvOpts(testKubeContext, namespaces)
			runner := newRenderRunner(test.hooks, &namespaces, opts, "")

			t.Override(&kubernetesclient.Client, fakeKubernetesClient)

			err = runner.RunPreHooks(context.Background(), os.Stdout)
			t.CheckNoError(err)
			preOut, err := io.ReadAll(preOutFile)
			t.CheckNoError(err)
			t.CheckContains(test.preHostHookOut, string(preOut))
			_, err = runner.RunPostHooks(context.Background(), manifest.ManifestList{}, os.Stdout)
			t.CheckNoError(err)
			postOut, err := io.ReadAll(postOutFile)
			t.CheckNoError(err)
			t.CheckContains(test.postHostHookOut, string(postOut))
		})
	}
}
func TestRenderHooksWindows(t *testing.T) {
	tests := []struct {
		description     string
		hooks           latest.RenderHooks
		preHostHookOut  string
		postHostHookOut string
		shouldErr       bool
	}{
		{
			description: "test on windows machine non-match",
			shouldErr:   true,
			hooks: latest.RenderHooks{
				PreHooks: []latest.RenderHookItem{
					{
						HostHook: &latest.HostHook{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo pre-hook running with SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
						},
					},
				},
				PostHooks: []latest.PostRenderHookItem{
					{
						HostHook: &latest.PostRenderHostHook{
							OS:      []string{"linux", "darwin"},
							Command: []string{"sh", "-c", "echo post-hook running with SKAFFOLD_KUBE_CONTEXT=$SKAFFOLD_KUBE_CONTEXT,SKAFFOLD_NAMESPACES=$SKAFFOLD_NAMESPACES"},
						},
					},
				},
			},
		},
		{
			description: "test on window match",
			hooks: latest.RenderHooks{
				PreHooks: []latest.RenderHookItem{
					{
						HostHook: &latest.HostHook{
							OS:      []string{"windows"},
							Command: []string{"pwsh", "-Command", `echo "pre-hook running with SKAFFOLD_KUBE_CONTEXT=$([Environment]::GetEnvironmentVariable('SKAFFOLD_KUBE_CONTEXT')),SKAFFOLD_NAMESPACES=$([Environment]::GetEnvironmentVariable('SKAFFOLD_NAMESPACES'))"`},
						},
					},
				},
				PostHooks: []latest.PostRenderHookItem{
					{
						HostHook: &latest.PostRenderHostHook{
							OS:      []string{"windows"},
							Command: []string{"pwsh", "-Command", `echo "post-hook running with SKAFFOLD_KUBE_CONTEXT=$([Environment]::GetEnvironmentVariable('SKAFFOLD_KUBE_CONTEXT')),SKAFFOLD_NAMESPACES=$([Environment]::GetEnvironmentVariable('SKAFFOLD_NAMESPACES'))"`},
						},
					},
				},
			},
			preHostHookOut:  "pre-hook running with SKAFFOLD_KUBE_CONTEXT=context1,SKAFFOLD_NAMESPACES=np1,np2",
			postHostHookOut: "post-hook running with SKAFFOLD_KUBE_CONTEXT=context1,SKAFFOLD_NAMESPACES=np1,np2",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if runtime.GOOS != Windows {
				t.Skip("Tests require windows machine")
			}

			dir := os.TempDir()
			preOutFile := filepath.Join(dir, "pre")
			t.Cleanup(func() {
				os.Remove(preOutFile)
			})

			test.hooks.PreHooks[0].HostHook.Command[2] = test.hooks.PreHooks[0].HostHook.Command[2] + " | Set-Content -Path " + preOutFile
			postOutFile := filepath.Join(dir, "post")
			t.Cleanup(func() {
				os.Remove(postOutFile)
			})

			test.hooks.PostHooks[0].HostHook.Command[2] = test.hooks.PostHooks[0].HostHook.Command[2] + " | Set-Content -Path " + postOutFile

			namespaces := []string{"np1", "np2"}
			opts := NewRenderEnvOpts(testKubeContext, namespaces)
			runner := newRenderRunner(test.hooks, &namespaces, opts, "")

			t.Override(&kubernetesclient.Client, fakeKubernetesClient)

			err := runner.RunPreHooks(context.Background(), os.Stdout)
			t.CheckNoError(err)
			preOut, err := os.ReadFile(preOutFile)
			t.CheckError(test.shouldErr, err)
			t.CheckContains(test.preHostHookOut, string(preOut))
			_, err = runner.RunPostHooks(context.Background(), manifest.ManifestList{}, os.Stdout)
			t.CheckNoError(err)
			postOut, err := os.ReadFile(postOutFile)
			t.CheckError(test.shouldErr, err)
			t.CheckContains(test.postHostHookOut, string(postOut))
		})
	}
}

func TestPostRenderHook(t *testing.T) {
	tests := []struct {
		description          string
		requireWindows       bool
		manifestList         string
		expectedManifestList string
		postHooks            []latest.PostRenderHookItem
		shouldError          bool
	}{
		{
			description:  "should change manifests with change linux",
			manifestList: SimpleManifest,
			expectedManifestList: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    test-name: after
spec:
  containers:
  - name: getting-started
    image: skaffold-example`,
			postHooks: []latest.PostRenderHookItem{
				{
					HostHook: &latest.PostRenderHostHook{
						OS:         []string{"linux", "darwin"},
						Command:    []string{"sed", "s/before/after/g"},
						WithChange: true,
					},
				},
			},
		},
		{
			description:          "should error with change does not have output linux",
			manifestList:         SimpleManifest,
			expectedManifestList: "",
			shouldError:          true,
			postHooks: []latest.PostRenderHookItem{
				{
					HostHook: &latest.PostRenderHostHook{
						OS:         []string{"linux", "darwin"},
						Command:    []string{"bash", "-c", "sleep 0.01s"},
						WithChange: true,
					},
				},
			},
		},
		{
			description:          "should not change manifests with change linux",
			manifestList:         SimpleManifest,
			expectedManifestList: SimpleManifest,
			postHooks: []latest.PostRenderHookItem{
				{
					HostHook: &latest.PostRenderHostHook{
						OS:         []string{"linux", "darwin"},
						Command:    []string{"sed", "s/before/after/g"},
						WithChange: false,
					},
				},
			},
		},
		{
			description:    "should change manifests with change windows",
			requireWindows: true,
			manifestList:   SimpleManifest,
			expectedManifestList: `apiVersion: v1
kind: Pod
metadata:
  name: getting-started
  labels:
    test-name: after
spec:
  containers:
  - name: getting-started
    image: skaffold-example`,
			postHooks: []latest.PostRenderHookItem{
				{
					HostHook: &latest.PostRenderHostHook{
						OS:         []string{"windows"},
						Command:    []string{"powershell", "-Command", `$input | ForEach-Object { $_ -replace "before", "after" }`},
						WithChange: true,
					},
				},
			},
		},
		{
			requireWindows:       true,
			description:          "should error with change does not have output windows",
			manifestList:         SimpleManifest,
			expectedManifestList: "",
			shouldError:          true,
			postHooks: []latest.PostRenderHookItem{
				{
					HostHook: &latest.PostRenderHostHook{
						OS:         []string{"windows"},
						Command:    []string{"cmd.exe", "/C", `timeout 1`},
						WithChange: true,
					},
				},
			},
		},
		{
			requireWindows:       true,
			description:          "should not change manifests with change windows",
			manifestList:         SimpleManifest,
			expectedManifestList: SimpleManifest,
			postHooks: []latest.PostRenderHookItem{
				{
					HostHook: &latest.PostRenderHostHook{
						OS:         []string{"windows"},
						Command:    []string{"pwsh", "-Command", `$input | ForEach-Object { $_ -replace "before", "after" }`},
						WithChange: false,
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.requireWindows != (runtime.GOOS == Windows) {
				t.Skip()
			}
			namespaces := []string{"np1", "np2"}
			opts := NewRenderEnvOpts(testKubeContext, namespaces)
			runner := newRenderRunner(latest.RenderHooks{PostHooks: test.postHooks}, &namespaces, opts, "")
			t.Override(&kubernetesclient.Client, fakeKubernetesClient)
			reader := strings.NewReader(SimpleManifest)
			manifestList, err := manifest.Load(reader)
			t.CheckNoError(err)
			var out bytes.Buffer
			postHookResult, err := runner.RunPostHooks(context.TODO(), manifestList, &out)
			t.CheckErrorAndDeepEqual(test.shouldError, err, test.expectedManifestList, postHookResult.String())
		})
	}
}
