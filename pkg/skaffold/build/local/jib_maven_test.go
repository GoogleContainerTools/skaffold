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

package local

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestBuildJibMavenToDocker(t *testing.T) {
	tests := []struct {
		description   string
		artifact      *latest.JibMavenArtifact
		commands      util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "build",
			artifact:    &latest.JibMavenArtifact{},
			commands: testutil.CmdRun(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --non-recursive prepare-package jib:dockerBuild -Dimage=img:tag",
			),
		},
		{
			description: "build with additional flags",
			artifact:    &latest.JibMavenArtifact{Flags: []string{"--flag1", "--flag2"}},
			commands: testutil.CmdRun(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --flag1 --flag2 --non-recursive prepare-package jib:dockerBuild -Dimage=img:tag",
			),
		},
		{
			description: "build with module",
			artifact:    &latest.JibMavenArtifact{Module: "module"},
			commands: testutil.CmdRun(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --projects module --also-make package jib:dockerBuild -Djib.containerize=module -Dimage=img:tag",
			),
		},
		{
			description: "build with module and profile",
			artifact:    &latest.JibMavenArtifact{Module: "module", Profile: "profile"},
			commands: testutil.CmdRun(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --activate-profiles profile --projects module --also-make package jib:dockerBuild -Djib.containerize=module -Dimage=img:tag",
			),
		},
		{
			description: "fail build",
			artifact:    &latest.JibMavenArtifact{},
			commands: testutil.CmdRunErr(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion="+jib.MinimumJibMavenVersion+" --non-recursive prepare-package jib:dockerBuild -Dimage=img:tag",
				errors.New("BUG"),
			),
			shouldErr:     true,
			expectedError: "maven build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			api := (&testutil.FakeAPIClient{}).Add("img:tag", "imageID")
			t.Override(&docker.NewAPIClient, func(*runcontext.RunContext) (docker.LocalDaemon, error) {
				return docker.NewLocalDaemon(api, nil, false, nil), nil
			})
			t.Override(&util.DefaultExecCommand, test.commands)

			builder, err := NewBuilder(stubRunContext(latest.LocalBuild{
				Push: util.BoolPtr(false),
			}))
			t.CheckNoError(err)

			result, err := builder.buildJibMaven(context.Background(), ioutil.Discard, ".", test.artifact, "img:tag")
			t.CheckError(test.shouldErr, err)

			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckDeepEqual("imageID", result)
			}
		})
	}
}

func TestBuildJibMavenToRegistry(t *testing.T) {
	tests := []struct {
		description   string
		artifact      *latest.JibMavenArtifact
		commands      util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "build",
			artifact:    &latest.JibMavenArtifact{},
			commands: testutil.CmdRun(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --non-recursive prepare-package jib:build -Dimage=img:tag",
			),
		},
		{
			description: "build with additional flags",
			artifact:    &latest.JibMavenArtifact{Flags: []string{"--flag1", "--flag2"}},
			commands: testutil.CmdRun(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --flag1 --flag2 --non-recursive prepare-package jib:build -Dimage=img:tag",
			),
		},
		{
			description: "build with module",
			artifact:    &latest.JibMavenArtifact{Module: "module"},
			commands: testutil.CmdRun(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --projects module --also-make package jib:build -Djib.containerize=module -Dimage=img:tag",
			),
		},
		{
			description: "build with module and profile",
			artifact:    &latest.JibMavenArtifact{Module: "module", Profile: "profile"},
			commands: testutil.CmdRun(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion=" + jib.MinimumJibMavenVersion + " --activate-profiles profile --projects module --also-make package jib:build -Djib.containerize=module -Dimage=img:tag",
			),
		},
		{
			description: "fail build",
			artifact:    &latest.JibMavenArtifact{},
			commands: testutil.CmdRunErr(
				"mvn -Djib.console=plain jib:_skaffold-fail-if-jib-out-of-date -Djib.requiredVersion="+jib.MinimumJibMavenVersion+" --non-recursive prepare-package jib:build -Dimage=img:tag",
				errors.New("BUG"),
			),
			shouldErr:     true,
			expectedError: "maven build failed",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.commands)
			t.Override(&docker.RemoteDigest, func(identifier string, _ map[string]bool) (string, error) {
				if identifier == "img:tag" {
					return "digest", nil
				}
				return "", errors.New("unknown remote tag")
			})
			t.Override(&docker.NewAPIClient, func(*runcontext.RunContext) (docker.LocalDaemon, error) {
				return docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil), nil
			})

			builder, err := NewBuilder(stubRunContext(latest.LocalBuild{
				Push: util.BoolPtr(true),
			}))
			t.CheckNoError(err)

			result, err := builder.buildJibMaven(context.Background(), ioutil.Discard, ".", test.artifact, "img:tag")
			t.CheckError(test.shouldErr, err)

			if test.shouldErr {
				t.CheckErrorContains(test.expectedError, err)
			} else {
				t.CheckDeepEqual("digest", result)
			}
		})
	}
}
