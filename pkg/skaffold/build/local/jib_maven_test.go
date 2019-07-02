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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func TestBuildJibMavenToDocker(t *testing.T) {
	var tests = []struct {
		description   string
		artifact      *latest.JibMavenArtifact
		cmd           util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "build",
			artifact:    &latest.JibMavenArtifact{},
			cmd:         testutil.FakeRun(t, "mvn -Djib.console=plain --non-recursive prepare-package jib:dockerBuild -Dimage=img:tag"),
		},
		{
			description: "build with additional flags",
			artifact:    &latest.JibMavenArtifact{Flags: []string{"--flag1", "--flag2"}},
			cmd:         testutil.FakeRun(t, "mvn -Djib.console=plain --flag1 --flag2 --non-recursive prepare-package jib:dockerBuild -Dimage=img:tag"),
		},
		{
			description: "build with module",
			artifact:    &latest.JibMavenArtifact{Module: "module"},
			cmd: testutil.
				FakeRunOut(t, "mvn --quiet --projects module jib:_skaffold-package-goals", "dockerBuild").
				WithRun("mvn -Djib.console=plain --projects module --also-make package -Dimage=img:tag"),
		},
		{
			description: "build with module and profile",
			artifact:    &latest.JibMavenArtifact{Module: "module", Profile: "profile"},
			cmd: testutil.
				FakeRunOut(t, "mvn --quiet --projects module jib:_skaffold-package-goals --activate-profiles profile", "dockerBuild").
				WithRun("mvn -Djib.console=plain --activate-profiles profile --projects module --also-make package -Dimage=img:tag"),
		},
		{
			description:   "fail build",
			artifact:      &latest.JibMavenArtifact{},
			cmd:           testutil.FakeRunErr(t, "mvn -Djib.console=plain --non-recursive prepare-package jib:dockerBuild -Dimage=img:tag", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "maven build failed",
		},
		{
			description:   "fail to check package goals",
			artifact:      &latest.JibMavenArtifact{Module: "module"},
			cmd:           testutil.FakeRunOutErr(t, "mvn --quiet --projects module jib:_skaffold-package-goals", "", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "could not obtain jib package goals",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.cmd)

			api := &testutil.FakeAPIClient{
				TagToImageID: map[string]string{"img:tag": "imageID"},
			}

			builder := &Builder{
				pushImages:  false,
				localDocker: docker.NewLocalDaemon(api, nil, false, map[string]bool{}),
			}
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
	var tests = []struct {
		description   string
		artifact      *latest.JibMavenArtifact
		cmd           util.Command
		shouldErr     bool
		expectedError string
	}{
		{
			description: "build",
			artifact:    &latest.JibMavenArtifact{},
			cmd:         testutil.FakeRun(t, "mvn -Djib.console=plain --non-recursive prepare-package jib:build -Dimage=img:tag"),
		},
		{
			description: "build with additional flags",
			artifact:    &latest.JibMavenArtifact{Flags: []string{"--flag1", "--flag2"}},
			cmd:         testutil.FakeRun(t, "mvn -Djib.console=plain --flag1 --flag2 --non-recursive prepare-package jib:build -Dimage=img:tag"),
		},
		{
			description: "build with module",
			artifact:    &latest.JibMavenArtifact{Module: "module"},
			cmd: testutil.
				FakeRunOut(t, "mvn --quiet --projects module jib:_skaffold-package-goals", "build").
				WithRun("mvn -Djib.console=plain --projects module --also-make package -Dimage=img:tag"),
		},
		{
			description: "build with module and profile",
			artifact:    &latest.JibMavenArtifact{Module: "module", Profile: "profile"},
			cmd: testutil.
				FakeRunOut(t, "mvn --quiet --projects module jib:_skaffold-package-goals --activate-profiles profile", "build").
				WithRun("mvn -Djib.console=plain --activate-profiles profile --projects module --also-make package -Dimage=img:tag"),
		},
		{
			description:   "fail build",
			artifact:      &latest.JibMavenArtifact{},
			cmd:           testutil.FakeRunErr(t, "mvn -Djib.console=plain --non-recursive prepare-package jib:build -Dimage=img:tag", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "maven build failed",
		},
		{
			description:   "fail to check package goals",
			artifact:      &latest.JibMavenArtifact{Module: "module"},
			cmd:           testutil.FakeRunOutErr(t, "mvn --quiet --projects module jib:_skaffold-package-goals", "", errors.New("BUG")),
			shouldErr:     true,
			expectedError: "could not obtain jib package goals",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.DefaultExecCommand, test.cmd)
			t.Override(&getRemoteDigest, func(identifier string, _ map[string]bool) (string, error) {
				if identifier == "img:tag" {
					return "digest", nil
				}
				return "", errors.New("unknown remote tag")
			})

			builder := &Builder{
				pushImages:  true,
				localDocker: docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, map[string]bool{}),
			}
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

func TestMavenVerifyJibPackageGoal(t *testing.T) {
	var tests = []struct {
		description  string
		requiredGoal string
		mavenOutput  string
		shouldErr    bool
	}{
		{
			description:  "no goals should fail",
			requiredGoal: "xxx",
			mavenOutput:  "",
			shouldErr:    true,
		},
		{
			description:  "no goals should fail; newline stripped",
			requiredGoal: "xxx",
			mavenOutput:  "\n",
			shouldErr:    true,
		},
		{
			description:  "valid goal",
			requiredGoal: "dockerBuild",
			mavenOutput:  "dockerBuild",
			shouldErr:    false,
		},
		{
			description:  "newline stripped",
			requiredGoal: "dockerBuild",
			mavenOutput:  "dockerBuild\n",
			shouldErr:    false,
		},
		{
			description:  "invalid goal",
			requiredGoal: "dockerBuild",
			mavenOutput:  "build\n",
			shouldErr:    true,
		},
		{
			description:  "too many goals",
			requiredGoal: "dockerBuild",
			mavenOutput:  "build\ndockerBuild\n",
			shouldErr:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&util.SkipWrapperCheck, true)
			t.Override(&util.DefaultExecCommand, t.FakeRunOut(
				"mvn --quiet --projects module jib:_skaffold-package-goals",
				test.mavenOutput,
			))

			err := verifyJibPackageGoal(context.Background(), test.requiredGoal, ".", &latest.JibMavenArtifact{Module: "module"})

			t.CheckError(test.shouldErr, err)
		})
	}
}
