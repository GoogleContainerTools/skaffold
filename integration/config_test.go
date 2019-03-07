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

package integration

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	yaml "gopkg.in/yaml.v2"
)

func TestListConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	baseConfig := &config.Config{
		Global: &config.ContextConfig{
			DefaultRepo: "global-repository",
		},
		ContextConfigs: []*config.ContextConfig{
			{
				Kubecontext: "test-context",
				DefaultRepo: "context-local-repository",
			},
		},
	}

	c, _ := yaml.Marshal(*baseConfig)
	cfg, teardown := testutil.TempFile(t, "config", c)
	defer teardown()

	type testListCase struct {
		description    string
		kubectx        string
		expectedOutput []string
	}

	var tests = []testListCase{
		{
			description:    "list for test-context",
			kubectx:        "test-context",
			expectedOutput: []string{"default-repo: context-local-repository"},
		},
		{
			description: "list all",
			expectedOutput: []string{
				"global:",
				"default-repo: global-repository",
				"kube-context: test-context",
				"default-repo: context-local-repository",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			args := []string{"config", "list", "-c", cfg}
			if test.kubectx != "" {
				args = append(args, "-k", test.kubectx)
			} else {
				args = append(args, "--all")
			}
			cmd := exec.Command("skaffold", args...)
			rawOut, err := util.RunCmdOut(cmd)
			if err != nil {
				t.Error(err)
			}
			out := string(rawOut)
			for _, output := range test.expectedOutput {
				if !strings.Contains(out, output) {
					t.Errorf("expected output %s not found in output: %s", output, out)
				}
			}
		})
	}
}

func TestSetConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	baseConfig := &config.Config{
		Global: &config.ContextConfig{
			DefaultRepo: "global-repository",
		},
		ContextConfigs: []*config.ContextConfig{
			{
				Kubecontext: "test-context",
				DefaultRepo: "context-local-repository",
			},
		},
	}

	c, _ := yaml.Marshal(*baseConfig)
	cfg, teardown := testutil.TempFile(t, "config", c)
	defer teardown()

	type testSetCase struct {
		description string
		kubectx     string
		key         string
		shouldErr   bool
	}

	var tests = []testSetCase{
		{
			description: "set default-repo for context",
			kubectx:     "test-context",
			key:         "default-repo",
		},
		{
			description: "set global default-repo",
			key:         "default-repo",
		},
		{
			description: "fail to set unrecognized value",
			key:         "doubt-this-will-ever-be-a-config-value",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			value := util.RandomID()
			args := []string{"config", "set", test.key, value}
			args = append(args, "-c", cfg)
			if test.kubectx != "" {
				args = append(args, "-k", test.kubectx)
			} else {
				args = append(args, "--global")
			}
			cmd := exec.Command("skaffold", args...)
			if err := util.RunCmd(cmd); err != nil {
				if test.shouldErr {
					return
				}
				t.Error(err)
			}

			listArgs := []string{"config", "list", "-c", cfg}
			if test.kubectx != "" {
				listArgs = append(listArgs, "-k", test.kubectx)
			} else {
				listArgs = append(listArgs, "--all")
			}
			listCmd := exec.Command("skaffold", listArgs...)
			out, err := util.RunCmdOut(listCmd)
			if err != nil {
				t.Error(err)
			}
			t.Log(string(out))
			if !strings.Contains(string(out), fmt.Sprintf("%s: %s", test.key, value)) {
				t.Errorf("value %s not set correctly", test.key)
			}
		})
	}
}
