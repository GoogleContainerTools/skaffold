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
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

var (
	// Output strings to emulate different versions of Helm
	version20rc = `Client: &version.Version{SemVer:"v2.0.0-rc.1", GitCommit:"92be174acf51e60a33287fb7011f4571eaa5cb98", GitTreeState:"clean"}\nError: cannot connect to Tiller\n`
	version21   = `Client: &version.Version{SemVer:"v2.15.1", GitCommit:"cf1de4f8ba70eded310918a8af3a96bfe8e7683b", GitTreeState:"clean"}\nServer: &version.Version{SemVer:"v2.16.1", GitCommit:"bbdfe5e7803a12bbdf97e94cd847859890cf4050", GitTreeState:"clean"}\n`
	version30b  = `version.BuildInfo{Version:"v3.0.0-beta.3", GitCommit:"5cb923eecbe80d1ad76399aee234717c11931d9a", GitTreeState:"clean", GoVersion:"go1.12.9"}`
	version30   = `version.BuildInfo{Version:"v3.0.0", GitCommit:"e29ce2a54e96cd02ccfce88bee4f58bb6e2a28b6", GitTreeState:"clean", GoVersion:"go1.13.4"}`
	version31   = `version.BuildInfo{Version:"v3.1.1", GitCommit:"afe70585407b420d0097d07b21c47dc511525ac8", GitTreeState:"clean", GoVersion:"go1.13.8"}`
	version35   = `version.BuildInfo{Version:"3.5.2", GitCommit:"c4e74854886b2efe3321e185578e6db9be0a6e29", GitTreeState:"clean", GoVersion:"go1.14.15"}`
)

type mockClient struct {
	enableDebug        bool
	overrideProtocols  []string
	configFile         string
	kubeContext        string
	kubeConfig         string
	labels             map[string]string
	manifestsOverrides map[string]string
	globalFlags        []string
}

func (h mockClient) ManifestOverrides() map[string]string {
	return h.manifestsOverrides
}

func (h mockClient) EnableDebug() bool           { return h.enableDebug }
func (h mockClient) OverrideProtocols() []string { return h.overrideProtocols }
func (h mockClient) ConfigFile() string          { return h.configFile }
func (h mockClient) KubeContext() string         { return h.kubeContext }
func (h mockClient) KubeConfig() string          { return h.kubeConfig }
func (h mockClient) Labels() map[string]string   { return h.labels }
func (h mockClient) GlobalFlags() []string       { return h.globalFlags }

func TestBinVer(t *testing.T) {
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
			ver, err := BinVer(context.Background())

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, ver.String())
		})
	}
}

func TestGenerateSkaffoldFilter(t *testing.T) {
	tests := []struct {
		description string
		enableDebug bool
		buildFile   string
		result      []string
	}{
		{
			description: "empty buildfile is skipped",
			buildFile:   "",
			result:      []string{"filter", "--kube-context", "kubecontext", "--kubeconfig", "kubeconfig"},
		},
		{
			description: "buildfile is added",
			buildFile:   "buildfile",
			result:      []string{"filter", "--kube-context", "kubecontext", "--build-artifacts", "buildfile", "--kubeconfig", "kubeconfig"},
		},
		{
			description: "debugging brings --debugging",
			enableDebug: true,
			result:      []string{"filter", "--kube-context", "kubecontext", "--debugging", "--kubeconfig", "kubeconfig"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version --client", version31))

			h := mockClient{
				enableDebug:        test.enableDebug,
				kubeContext:        "kubecontext",
				kubeConfig:         "kubeconfig",
				manifestsOverrides: map[string]string{},
			}

			result := generateSkaffoldFilter(h, test.buildFile)
			t.CheckDeepEqual(test.result, result)
		})
	}
}
