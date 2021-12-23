/*
Copyright 2020 The Skaffold Authors

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

package debug

import (
	"encoding/json"
	"testing"

	"github.com/buildpacks/lifecycle/launch"
	cnb "github.com/buildpacks/lifecycle/platform"
	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsCNBImage(t *testing.T) {
	tests := []struct {
		description string
		input       ImageConfiguration
		expected    bool
	}{
		{"non-cnb image", ImageConfiguration{Entrypoint: []string{"/usr/bin/java", "-jar", "foo.jar"}}, false},
		{"implicit platform 0.3 with launcher missing label", ImageConfiguration{Entrypoint: []string{cnbLauncher}}, false},
		{"implicit platform 0.3 with launcher", ImageConfiguration{Entrypoint: []string{cnbLauncher}, Labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, true},
		{"explicit platform 0.3 with launcher", ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PLATFORM_API": "0.3"}, Labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, true},
		{"platform 0.4 with launcher", ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}, Labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, true},
		{"platform 0.4 with process executable", ImageConfiguration{Entrypoint: []string{"/cnb/process/diag"}, Arguments: []string{"arg"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}, Labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, true},
		{"platform 0.4 with non-cnb entrypoint", ImageConfiguration{Entrypoint: []string{"/usr/bin/java", "-jar", "foo.jar"}, Arguments: []string{"arg"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}, Labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, false},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, isCNBImage(test.input))
		})
	}
}
func TestHasCNBLauncherEntrypoint(t *testing.T) {
	tests := []struct {
		description string
		entrypoint  []string
		expected    bool
	}{
		{"nil", []string{}, false},
		{"empty", []string{""}, false},
		{"nonlauncher", []string{"/cnb/process/web"}, false},
		{"launcher", []string{"/cnb/lifecycle/launcher"}, true},
		{"launcher as arg", []string{"/bin/sh", "/cnb/lifecycle/launcher"}, false},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ic := ImageConfiguration{Entrypoint: test.entrypoint}
			t.CheckDeepEqual(test.expected, hasCNBLauncherEntrypoint(ic))
		})
	}
}

func TestFindCNBProcess(t *testing.T) {
	// metadata with default process type `web`
	md := cnb.BuildMetadata{Processes: []launch.Process{
		{Type: "web", Command: "webProcess arg1 arg2", Args: []string{"posArg1", "posArg2"}},
		{Type: "diag", Command: "diagProcess"},
	}}
	tests := []struct {
		description string
		input       ImageConfiguration
		found       bool
		processType string
		args        []string
	}{
		{"default is web", ImageConfiguration{Entrypoint: []string{cnbLauncher}}, true, "web", nil},
		{"platform 0.3 default is web", ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PLATFORM_API": "0.3"}}, true, "web", nil},
		{"platform 0.3 explicit", ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"diag"}}, true, "diag", nil},
		{"platform 0.3 environment", ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PROCESS_TYPE": "diag"}}, true, "diag", nil},
		{"platform 0.4 has no default", ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}}, false, "", nil},
		{"platform 0.4 process executable", ImageConfiguration{Entrypoint: []string{"/cnb/process/diag"}, Arguments: []string{"arg"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}}, true, "diag", []string{"arg"}},
		{"script-style args", ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"web", "arg"}}, false, "", nil},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			p, args, found := findCNBProcess(test.input, md)
			t.CheckDeepEqual(test.found, found)
			if found {
				t.CheckDeepEqual(test.processType, p.Type)
				t.CheckDeepEqual(test.args, args)
			}
		})
	}
}

func TestAdjustCommandLine(t *testing.T) {
	// metadata with default process type `web`
	md := cnb.BuildMetadata{Processes: []launch.Process{
		{Type: "web", Command: "webProcess arg1 arg2", Args: []string{"posArg1", "posArg2"}},
		{Type: "diag", Command: "diagProcess", Args: []string{"posArg1", "posArg2"}, Direct: true},
	}}
	tests := []struct {
		description string
		input       ImageConfiguration
		result      ImageConfiguration
		hasRewriter bool
	}{
		{
			description: "platform 0.3 default web process",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"webProcess", "arg1", "arg2"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 explicit web",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"web"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"webProcess", "arg1", "arg2"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 explicit diag",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"diag"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"diagProcess", "posArg1", "posArg2"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 environment",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PROCESS_TYPE": "diag"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"diagProcess", "posArg1", "posArg2"}, Env: map[string]string{"CNB_PROCESS_TYPE": "diag"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 invalid process (env) should be untouched",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PROCESS_TYPE": "not-found"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PROCESS_TYPE": "not-found"}},
			hasRewriter: false,
		},
		{
			description: "platform 0.3 script-style with args",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"the command line", "arg"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"the", "command", "line"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 direct with args",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"--", "the", "command", "line"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"the", "command", "line"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.4 with no default should be unchanged",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			hasRewriter: false,
		},
		{
			description: "platform 0.4 process executable",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/process/diag"}, Arguments: []string{"arg"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"diagProcess", "posArg1", "posArg2", "arg"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.4 invalid process (env) should be untouched",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PLATFORM_API": "0.4", "CNB_PROCESS_TYPE": "not-found"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Env: map[string]string{"CNB_PLATFORM_API": "0.4", "CNB_PROCESS_TYPE": "not-found"}},
			hasRewriter: false,
		},
		{
			description: "platform 0.4 direct with args",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"--", "the", "command", "line"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"the", "command", "line"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.4 script-style with args",
			input:       ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"the command line", "arg"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			result:      ImageConfiguration{Entrypoint: []string{cnbLauncher}, Arguments: []string{"the", "command", "line"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			hasRewriter: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ic, rewriter := adjustCommandLine(md, test.input)
			t.CheckDeepEqual(test.result, ic, cmp.AllowUnexported(test.result))
			if test.hasRewriter {
				// todo: can we test the rewriter?  We do exercise it in TestForCNBImage
				t.CheckNotNil(rewriter)
			} else {
				t.CheckNil(rewriter)
			}
		})
	}
}

func TestUpdateForCNBImage(t *testing.T) {
	// metadata with default process type `web`
	md := cnb.BuildMetadata{Processes: []launch.Process{
		// script-style process with positional arguments equiv to `sh -c "webProcess arg1 arg2" posArg1 posArg2`
		{Type: "web", Command: "webProcess arg1 arg2", Args: []string{"posArg1", "posArg2"}},
		{Type: "diag", Command: "diagProcess"},
		// direct process will exec `command cmdArg1`
		{Type: "direct", Command: "command", Args: []string{"cmdArg1"}, Direct: true},
		{Type: "sh-c", Command: "/bin/sh", Args: []string{"-c", "command arg1 arg2"}, Direct: true},
		// Google Buildpacks turns Procfiles into `/bin/bash -c cmdline`
		{Type: "bash-c", Command: "/bin/bash", Args: []string{"-c", "command arg1 arg2"}, Direct: true},
	}}
	mdMarshalled, _ := json.Marshal(&md)
	mdJSON := string(mdMarshalled)
	// metadata with no default process type
	mdnd := cnb.BuildMetadata{Processes: []launch.Process{
		{Type: "diag", Command: "diagProcess"},
		{Type: "direct", Command: "command", Args: []string{"cmdArg1"}, Direct: true},
	}}
	mdndMarshalled, _ := json.Marshal(&mdnd)
	mdndJSON := string(mdndMarshalled)

	tests := []struct {
		description string
		input       ImageConfiguration
		shouldErr   bool
		expected    types.ExecutableContainer
		config      types.ContainerDebugConfiguration
	}{
		{
			description: "error when missing build.metadata",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}},
			shouldErr:   true,
		},
		{
			description: "error when build.metadata missing processes",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Labels: map[string]string{"io.buildpacks.build.metadata": "{}"}},
			shouldErr:   true,
		},
		{
			description: "direct command-lines are kept as direct command-lines",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Arguments: []string{"--", "web", "arg1", "arg2"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"--", "web", "arg1", "arg2"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "defaults to web process when no process type",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "resolves to default 'web' process",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Arguments: []string{"web"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_PROCESS_TYPE=web",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Env: map[string]string{"CNB_PROCESS_TYPE": "web"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_PROCESS_TYPE=diag",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Env: map[string]string{"CNB_PROCESS_TYPE": "diag"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"diagProcess"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_PROCESS_TYPE=direct",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Env: map[string]string{"CNB_PROCESS_TYPE": "direct"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"--", "command", "cmdArg1"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "script command-line",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Arguments: []string{"python main.py"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"python main.py"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "no process and no args",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "launcher ignores image's working dir",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}, WorkingDir: "/workdir"},
			shouldErr:   false,
			expected:    types.ExecutableContainer{},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_APP_DIR used if set",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}, Env: map[string]string{"CNB_APP_DIR": "/appDir"}, WorkingDir: "/workdir"},
			shouldErr:   false,
			expected:    types.ExecutableContainer{},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/appDir"},
		},
		{
			description: "CNB_PROCESS_TYPE=sh-c (Procfile-style)",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Env: map[string]string{"CNB_PROCESS_TYPE": "sh-c"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"command arg1 arg2"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_PROCESS_TYPE=bash-c (Procfile-style)",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Env: map[string]string{"CNB_PROCESS_TYPE": "sh-c"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"command arg1 arg2"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},

		// Platform API 0.4
		{
			description: "Platform API 0.4: no default process for cnbLauncher",
			// Rather than treat this an error, we just don't do any rewriting and let the CNB launcher error instead.
			input:     ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr: false,
			expected:  types.ExecutableContainer{},
			config:    types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: direct command-lines are kept as direct command-lines",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Arguments: []string{"--", "web", "arg1", "arg2"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"--", "web", "arg1", "arg2"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: script command-line",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Arguments: []string{"python main.py"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Args: []string{"python main.py"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: launcher ignores image's working dir",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}, WorkingDir: "/workdir", Labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: CNB_APP_DIR used if set",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/lifecycle/launcher"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4", "CNB_APP_DIR": "/appDir"}, WorkingDir: "/workdir", Labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/appDir"},
		},
		{
			description: "Platform API 0.4: /cnb/process/web",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/process/web"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Command: []string{"/cnb/lifecycle/launcher"}, Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: /cnb/process/web with arguments are appended",
			input:       ImageConfiguration{Entrypoint: []string{"/cnb/process/web"}, Arguments: []string{"altArg1", "altArg2"}, Env: map[string]string{"CNB_PLATFORM_API": "0.4"}, Labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    types.ExecutableContainer{Command: []string{"/cnb/lifecycle/launcher"}, Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2", "altArg1", "altArg2"}},
			config:      types.ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
	}
	for _, test := range tests {
		// Test that when a transform modifies the command-line arguments, then
		// the changes are reflected to the launcher command-line
		testutil.Run(t, test.description+" (args changed)", func(t *testutil.T) {
			argsChangedTransform := func(a types.ContainerAdapter, ic ImageConfiguration) (types.ContainerDebugConfiguration, string, error) {
				a.GetContainer().Args = ic.Arguments
				return types.ContainerDebugConfiguration{}, "", nil
			}
			container := types.ExecutableContainer{}
			a := &testAdapter{&container}
			c, _, err := updateForCNBImage(a, test.input, argsChangedTransform)
			a.Apply()
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, container)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.config, c)
		})

		// Test that when the arguments are left unchanged, that the container is unchanged
		testutil.Run(t, test.description+" (args unchanged)", func(t *testutil.T) {
			argsUnchangedTransform := func(_ types.ContainerAdapter, ic ImageConfiguration) (types.ContainerDebugConfiguration, string, error) {
				return types.ContainerDebugConfiguration{WorkingDir: ic.WorkingDir}, "", nil
			}

			container := types.ExecutableContainer{}
			a := &testAdapter{&container}
			_, _, err := updateForCNBImage(a, test.input, argsUnchangedTransform)
			a.Apply()
			t.CheckError(test.shouldErr, err)
			if container.Args != nil {
				t.Errorf("args not nil: %v", container.Args)
			}
		})
	}
}
