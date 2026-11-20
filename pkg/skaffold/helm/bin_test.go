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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

var (
	// Output strings to emulate different versions of Helm
	version20rc   = `Client: &version.Version{SemVer:"v2.0.0-rc.1", GitCommit:"92be174acf51e60a33287fb7011f4571eaa5cb98", GitTreeState:"clean"}\nError: cannot connect to Tiller\n`
	version21     = `Client: &version.Version{SemVer:"v2.15.1", GitCommit:"cf1de4f8ba70eded310918a8af3a96bfe8e7683b", GitTreeState:"clean"}\nServer: &version.Version{SemVer:"v2.16.1", GitCommit:"bbdfe5e7803a12bbdf97e94cd847859890cf4050", GitTreeState:"clean"}\n`
	version30b    = `version.BuildInfo{Version:"v3.0.0-beta.3", GitCommit:"5cb923eecbe80d1ad76399aee234717c11931d9a", GitTreeState:"clean", GoVersion:"go1.12.9"}`
	version30     = `version.BuildInfo{Version:"v3.0.0", GitCommit:"e29ce2a54e96cd02ccfce88bee4f58bb6e2a28b6", GitTreeState:"clean", GoVersion:"go1.13.4"}`
	version31     = `version.BuildInfo{Version:"v3.1.1", GitCommit:"afe70585407b420d0097d07b21c47dc511525ac8", GitTreeState:"clean", GoVersion:"go1.13.8"}`
	version35     = `version.BuildInfo{Version:"3.5.2", GitCommit:"c4e74854886b2efe3321e185578e6db9be0a6e29", GitTreeState:"clean", GoVersion:"go1.14.15"}`
	version40beta = `version.BuildInfo{Version:"v4.0.0-beta.1", GitCommit:"7dd2484aed31aaff0453b348132fe1e30c6d1db7", GitTreeState:"clean", GoVersion:"go1.24.7"}`
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
		{"Helm 4.0.0 beta", version40beta, "4.0.0-beta.1", false},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version", test.helmVersion))
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
		globalFlags []string
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
		{
			description: "helm global flags are not included in filter flags",
			result:      []string{"filter", "--kube-context", "kubecontext", "--kubeconfig", "kubeconfig"},
			globalFlags: []string{"--debug"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm version", version31))

			h := mockClient{
				enableDebug:        test.enableDebug,
				kubeContext:        "kubecontext",
				kubeConfig:         "kubeconfig",
				manifestsOverrides: map[string]string{},
				globalFlags:        test.globalFlags,
			}

			result := generateSkaffoldFilter(h, test.buildFile, []string{})
			t.CheckDeepEqual(test.result, result)
		})
	}
}

func TestPreparePostRenderer(t *testing.T) {
	tests := []struct {
		description      string
		helmVersion      semver.Version
		pluginInstallDir string
		cmdOverrides     []*testutil.ExpectedCmd
		expectedFlags    []string
		shouldErr        bool
		expectedErrSub   string
	}{
		{
			description:   "Helm v3 returns flag directly without plugin installation",
			helmVersion:   semver.MustParse("3.14.0"),
			expectedFlags: []string{"--post-renderer", "skaffold"},
			shouldErr:     false,
		},
		{
			description:      "Helm v4 with custom install dir using standard slash",
			helmVersion:      semver.MustParse("4.0.0-beta.1"),
			pluginInstallDir: "TEMPORARY-TEST-DIR/skaffold-render-12345",
			cmdOverrides: []*testutil.ExpectedCmd{
				testutil.CmdRun("helm plugin install TEMPORARY-TEST-DIR/skaffold-render-12345"),
				testutil.CmdRun("helm plugin uninstall skaffold-render-12345"),
			},
			expectedFlags: []string{"--post-renderer", "skaffold-render-12345"},
			shouldErr:     false,
		},
		{
			description:      "Helm v4 with Windows path separators (issue #10197)",
			helmVersion:      semver.MustParse("4.0.0-beta.1"),
			pluginInstallDir: `TEMPORARY-TEST-DIR\skaffold-render-win67890`,
			cmdOverrides: []*testutil.ExpectedCmd{
				testutil.CmdRun(`helm plugin install TEMPORARY-TEST-DIR\skaffold-render-win67890`),
				testutil.CmdRun("helm plugin uninstall skaffold-render-win67890"),
			},
			expectedFlags: []string{"--post-renderer", "skaffold-render-win67890"},
			shouldErr:     false,
		},
		{
			description:      "Helm v4 plugin install failure",
			helmVersion:      semver.MustParse("4.0.0-beta.1"),
			pluginInstallDir: "TEMPORARY-TEST-DIR/skaffold-render-fail",
			cmdOverrides: []*testutil.ExpectedCmd{
				testutil.CmdRunErr("helm plugin install TEMPORARY-TEST-DIR/skaffold-render-fail", fmt.Errorf("plugin install error")),
			},
			shouldErr:      true,
			expectedErrSub: "Failed to install Helm plugin",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&PluginInstallDir, test.pluginInstallDir)
			if len(test.cmdOverrides) > 0 {
				t.Override(&util.DefaultExecCommand, testutil.CmdRunCombined(test.cmdOverrides...))
			}

			h := mockClient{}
			cleanup, flags, err := PreparePostRenderer(context.Background(), h, "skaffold", test.helmVersion)

			if test.shouldErr {
				t.CheckError(true, err)
				if test.expectedErrSub != "" {
					t.CheckContains(test.expectedErrSub, err.Error())
				}
				t.CheckTrue(cleanup == nil)
			} else {
				t.CheckNoError(err)
				t.CheckDeepEqual(test.expectedFlags, flags)
				if cleanup != nil {
					cleanup()
				}
			}
		})
	}
}

func TestPreparePostRendererManifestWindowsPath(t *testing.T) {
	testutil.Run(t, "plugin.yaml parses correctly when directory has Windows backslashes", func(t *testutil.T) {
		windowsDir := `C:\Users\ContainerUser\AppData\Local\Temp\skaffold-render123456789`
		clean := strings.ReplaceAll(windowsDir, "\\", "/")
		name := filepath.Base(clean)
		t.CheckDeepEqual("skaffold-render123456789", name)

		skaffoldBinaryYaml, err := yaml.Marshal(`C:\Program Files\Skaffold\skaffold.exe`)
		t.CheckNoError(err)

		manifest := fmt.Sprintf(PostRendererTemplate, name, skaffoldBinaryYaml)

		var parsed map[string]interface{}
		err = yaml.Unmarshal([]byte(manifest), &parsed)
		t.CheckNoError(err)
		t.CheckDeepEqual("skaffold-render123456789", parsed["name"])
	})
}

func TestPreparePostRendererRealDir(t *testing.T) {
	testutil.Run(t, "Helm v4 creates valid plugin.yaml in temporary directory", func(t *testutil.T) {
		tempDir := t.NewTempDir()
		targetPluginDir := filepath.Join(tempDir.Root(), "skaffold-render-actual")
		err := os.MkdirAll(targetPluginDir, 0755)
		t.CheckNoError(err)

		t.Override(&PluginInstallDir, targetPluginDir)
		t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput(fmt.Sprintf("helm plugin install %s", targetPluginDir), ""))

		h := mockClient{}
		cleanup, flags, err := PreparePostRenderer(context.Background(), h, "/usr/local/bin/skaffold", semver.MustParse("4.0.0-beta.1"))
		t.CheckNoError(err)
		t.CheckDeepEqual([]string{"--post-renderer", "skaffold-render-actual"}, flags)

		pluginYamlPath := filepath.Join(targetPluginDir, "plugin.yaml")
		content, err := os.ReadFile(pluginYamlPath)
		t.CheckNoError(err)

		var parsed map[string]interface{}
		err = yaml.Unmarshal(content, &parsed)
		t.CheckNoError(err)
		t.CheckDeepEqual("skaffold-render-actual", parsed["name"])

		t.Override(&util.DefaultExecCommand, testutil.CmdRunWithOutput("helm plugin uninstall skaffold-render-actual", ""))
		cleanup()

		_, err = os.Stat(targetPluginDir)
		t.CheckTrue(os.IsNotExist(err))
	})
}

