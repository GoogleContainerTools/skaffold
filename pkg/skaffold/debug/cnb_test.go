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

	cnb "github.com/buildpacks/lifecycle"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestIsCNBImage(t *testing.T) {
	tests := []struct {
		description string
		input       imageConfiguration
		expected    bool
	}{
		{"non-cnb image", imageConfiguration{entrypoint: []string{"/usr/bin/java", "-jar", "foo.jar"}}, false},
		{"implicit platform 0.3 with launcher missing label", imageConfiguration{entrypoint: []string{cnbLauncher}}, false},
		{"implicit platform 0.3 with launcher", imageConfiguration{entrypoint: []string{cnbLauncher}, labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, true},
		{"explicit platform 0.3 with launcher", imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PLATFORM_API": "0.3"}, labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, true},
		{"platform 0.4 with launcher", imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}, labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, true},
		{"platform 0.4 with process executable", imageConfiguration{entrypoint: []string{"/cnb/process/diag"}, arguments: []string{"arg"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}, labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, true},
		{"platform 0.4 with non-cnb entrypoint", imageConfiguration{entrypoint: []string{"/usr/bin/java", "-jar", "foo.jar"}, arguments: []string{"arg"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}, labels: map[string]string{"io.buildpacks.stack.id": "not checked"}}, false},
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
			ic := imageConfiguration{entrypoint: test.entrypoint}
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
		input       imageConfiguration
		found       bool
		processType string
		args        []string
	}{
		{"default is web", imageConfiguration{entrypoint: []string{cnbLauncher}}, true, "web", nil},
		{"platform 0.3 default is web", imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PLATFORM_API": "0.3"}}, true, "web", nil},
		{"platform 0.3 explicit", imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"diag"}}, true, "diag", nil},
		{"platform 0.3 environment", imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PROCESS_TYPE": "diag"}}, true, "diag", nil},
		{"platform 0.4 has no default", imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}}, false, "", nil},
		{"platform 0.4 process executable", imageConfiguration{entrypoint: []string{"/cnb/process/diag"}, arguments: []string{"arg"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}}, true, "diag", []string{"arg"}},
		{"script-style args", imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"web", "arg"}}, false, "", nil},
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
		input       imageConfiguration
		result      imageConfiguration
		hasRewriter bool
	}{
		{
			description: "platform 0.3 default web process",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"webProcess", "arg1", "arg2"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 explicit web",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"web"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"webProcess", "arg1", "arg2"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 explicit diag",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"diag"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"diagProcess", "posArg1", "posArg2"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 environment",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PROCESS_TYPE": "diag"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"diagProcess", "posArg1", "posArg2"}, env: map[string]string{"CNB_PROCESS_TYPE": "diag"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 invalid process (env) should be untouched",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PROCESS_TYPE": "not-found"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PROCESS_TYPE": "not-found"}},
			hasRewriter: false,
		},
		{
			description: "platform 0.3 script-style with args",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"the command line", "arg"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"the", "command", "line"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.3 direct with args",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"--", "the", "command", "line"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"the", "command", "line"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.4 with no default should be unchanged",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			hasRewriter: false,
		},
		{
			description: "platform 0.4 process executable",
			input:       imageConfiguration{entrypoint: []string{"/cnb/process/diag"}, arguments: []string{"arg"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"diagProcess", "posArg1", "posArg2", "arg"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.4 invalid process (env) should be untouched",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PLATFORM_API": "0.4", "CNB_PROCESS_TYPE": "not-found"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, env: map[string]string{"CNB_PLATFORM_API": "0.4", "CNB_PROCESS_TYPE": "not-found"}},
			hasRewriter: false,
		},
		{
			description: "platform 0.4 direct with args",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"--", "the", "command", "line"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"the", "command", "line"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			hasRewriter: true,
		},
		{
			description: "platform 0.4 script-style with args",
			input:       imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"the command line", "arg"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
			result:      imageConfiguration{entrypoint: []string{cnbLauncher}, arguments: []string{"the", "command", "line"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}},
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
		input       imageConfiguration
		shouldErr   bool
		expected    v1.Container
		config      ContainerDebugConfiguration
	}{
		{
			description: "error when missing build.metadata",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}},
			shouldErr:   true,
		},
		{
			description: "error when build.metadata missing processes",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, labels: map[string]string{"io.buildpacks.build.metadata": "{}"}},
			shouldErr:   true,
		},
		{
			description: "direct command-lines are kept as direct command-lines",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, arguments: []string{"--", "web", "arg1", "arg2"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"--", "web", "arg1", "arg2"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "defaults to web process when no process type",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "resolves to default 'web' process",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, arguments: []string{"web"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_PROCESS_TYPE=web",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PROCESS_TYPE": "web"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_PROCESS_TYPE=diag",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PROCESS_TYPE": "diag"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"diagProcess"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_PROCESS_TYPE=direct",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PROCESS_TYPE": "direct"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"--", "command", "cmdArg1"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "script command-line",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, arguments: []string{"python main.py"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"python main.py"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "no process and no args",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}},
			shouldErr:   false,
			expected:    v1.Container{},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "launcher ignores image's working dir",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}, workingDir: "/workdir"},
			shouldErr:   false,
			expected:    v1.Container{},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_APP_DIR used if set",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}, env: map[string]string{"CNB_APP_DIR": "/appDir"}, workingDir: "/workdir"},
			shouldErr:   false,
			expected:    v1.Container{},
			config:      ContainerDebugConfiguration{WorkingDir: "/appDir"},
		},
		{
			description: "CNB_PROCESS_TYPE=sh-c (Procfile-style)",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PROCESS_TYPE": "sh-c"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"command arg1 arg2"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "CNB_PROCESS_TYPE=bash-c (Procfile-style)",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PROCESS_TYPE": "sh-c"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"command arg1 arg2"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},

		// Platform API 0.4
		{
			description: "Platform API 0.4: no default process for cnbLauncher",
			// Rather than treat this an error, we just don't do any rewriting and let the CNB launcher error instead.
			input:     imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr: false,
			expected:  v1.Container{},
			config:    ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: direct command-lines are kept as direct command-lines",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, arguments: []string{"--", "web", "arg1", "arg2"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"--", "web", "arg1", "arg2"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: script command-line",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, arguments: []string{"python main.py"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"python main.py"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: launcher ignores image's working dir",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}, workingDir: "/workdir", labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}},
			shouldErr:   false,
			expected:    v1.Container{},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: CNB_APP_DIR used if set",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PLATFORM_API": "0.4", "CNB_APP_DIR": "/appDir"}, workingDir: "/workdir", labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}},
			shouldErr:   false,
			expected:    v1.Container{},
			config:      ContainerDebugConfiguration{WorkingDir: "/appDir"},
		},
		{
			description: "Platform API 0.4: /cnb/process/web",
			input:       imageConfiguration{entrypoint: []string{"/cnb/process/web"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Command: []string{"/cnb/lifecycle/launcher"}, Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
		{
			description: "Platform API 0.4: /cnb/process/web with arguments are appended",
			input:       imageConfiguration{entrypoint: []string{"/cnb/process/web"}, arguments: []string{"altArg1", "altArg2"}, env: map[string]string{"CNB_PLATFORM_API": "0.4"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Command: []string{"/cnb/lifecycle/launcher"}, Args: []string{"webProcess arg1 arg2", "posArg1", "posArg2", "altArg1", "altArg2"}},
			config:      ContainerDebugConfiguration{WorkingDir: "/workspace"},
		},
	}
	for _, test := range tests {
		// Test that when a transform modifies the command-line arguments, then
		// the changes are reflected to the launcher command-line
		testutil.Run(t, test.description+" (args changed)", func(t *testutil.T) {
			argsChangedTransform := func(c *v1.Container, ic imageConfiguration) (ContainerDebugConfiguration, string, error) {
				c.Args = ic.arguments
				return ContainerDebugConfiguration{}, "", nil
			}
			copy := v1.Container{}
			c, _, err := updateForCNBImage(&copy, test.input, argsChangedTransform)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, copy)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.config, c)
		})

		// Test that when the arguments are left unchanged, that the container is unchanged
		testutil.Run(t, test.description+" (args unchanged)", func(t *testutil.T) {
			argsUnchangedTransform := func(c *v1.Container, ic imageConfiguration) (ContainerDebugConfiguration, string, error) {
				return ContainerDebugConfiguration{WorkingDir: ic.workingDir}, "", nil
			}

			copy := v1.Container{}
			_, _, err := updateForCNBImage(&copy, test.input, argsUnchangedTransform)
			t.CheckError(test.shouldErr, err)
			if copy.Args != nil {
				t.Errorf("args not nil: %v", copy.Args)
			}
		})
	}
}
