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
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
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

	var tests = []struct {
		description    string
		args           []string
		expectedOutput []string
	}{
		{
			description:    "list for test-context",
			args:           []string{"-k", "test-context"},
			expectedOutput: []string{"default-repo: context-local-repository"},
		},
		{
			description: "list all",
			args:        []string{"--all"},
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
			args := append([]string{"list", "-c", cfg}, test.args...)
			rawOut := skaffold.Config(args...).RunOrFail(t)

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

	var tests = []struct {
		description string
		setArgs     []string
		listArgs    []string
		key         string
		shouldErr   bool
	}{
		{
			description: "set default-repo for context",
			setArgs:     []string{"-k", "test-context"},
			listArgs:    []string{"-k", "test-context"},
			key:         "default-repo",
		},
		{
			description: "set global default-repo",
			setArgs:     []string{"--global"},
			listArgs:    []string{"--all"},
			key:         "default-repo",
		},
		{
			description: "fail to set unrecognized value",
			setArgs:     []string{"--global"},
			listArgs:    []string{"--all"},
			key:         "doubt-this-will-ever-be-a-config-value",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			value := util.RandomID()

			args := append([]string{"set", test.key, value, "-c", cfg}, test.setArgs...)
			_, err := skaffold.Config(args...).Run(t)
			if err != nil {
				if test.shouldErr {
					return
				}
				t.Error(err)
			}

			args = append([]string{"list", "-c", cfg}, test.listArgs...)
			out := skaffold.Config(args...).RunOrFail(t)

			if !strings.Contains(string(out), fmt.Sprintf("%s: %s", test.key, value)) {
				t.Errorf("value %s not set correctly", test.key)
			}
		})
	}
}
