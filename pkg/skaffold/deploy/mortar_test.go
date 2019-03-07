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

package deploy

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestMortarDeploy(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *latest.MortarDeploy
		builds      []build.Artifact
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "deploy success",
			cfg: &latest.MortarDeploy{
				Name:   "test-shot",
				Source: "manifests/",
			},
			command: testutil.NewFakeCmd(t).
				WithRun("mortar fire manifests/ test-shot"),
			builds: []build.Artifact{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:123",
				},
			},
		},
		{
			description: "deploy success with config file",
			cfg: &latest.MortarDeploy{
				Name:   "test-shot",
				Source: "manifests/",
				Config: "shot-test.yml",
			},
			command: testutil.NewFakeCmd(t).
				WithRun("mortar fire -c shot-test.yml manifests/ test-shot"),
			builds: []build.Artifact{
				{
					ImageName: "leeroy-web",
					Tag:       "leeroy-web:123",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			m := NewMortarDeployer(test.cfg)
			err := m.Deploy(context.Background(), ioutil.Discard, test.builds, nil)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}

func TestMortarCleanup(t *testing.T) {
	var tests = []struct {
		description string
		cfg         *latest.MortarDeploy
		command     util.Command
		shouldErr   bool
	}{
		{
			description: "cleanup success",
			cfg: &latest.MortarDeploy{
				Name:   "test-shot",
				Source: "manifests/",
			},
			command: testutil.NewFakeCmd(t).WithRun("mortar yank --force test-shot"),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			m := NewMortarDeployer(test.cfg)
			err := m.Cleanup(context.Background(), ioutil.Discard)

			testutil.CheckError(t, test.shouldErr, err)
		})
	}
}
