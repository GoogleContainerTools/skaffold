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

package jib

import (
	"bytes"
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/pkg/errors"
)

func NewTestMavenBuilder() *MavenBuilder {
	localDaemon := docker.NewLocalDaemon(&testutil.FakeAPIClient{
		TagToImageID: map[string]string{
			"my-tag": "image id from my-tag",
		}}, nil, true)
	return &MavenBuilder{
		LocalDocker: localDaemon,
		opts:        &config.SkaffoldOptions{},
	}
}

func TestMavenVerifyJibPackageGoal(t *testing.T) {
	var testCases = []struct {
		requiredGoal string
		mavenOutput  string
		shouldError  bool
	}{
		{"xxx", "", true},   // no goals should fail
		{"xxx", "\n", true}, // no goals should fail; newline stripped
		{"dockerBuild", "dockerBuild", false},
		{"dockerBuild", "dockerBuild\n", false}, // newline stripped
		{"dockerBuild", "build\n", true},
		{"dockerBuild", "build\ndockerBuild\n", true},
	}

	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.SkipWrapperCheck = true

	for _, tt := range testCases {
		util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunOut("mvn --quiet --projects module jib:_skaffold-package-goals", tt.mavenOutput)

		err := verifyJibPackageGoal(context.Background(), tt.requiredGoal, ".", &latest.JibMavenArtifact{Module: "module"})
		if hasError := err != nil; tt.shouldError != hasError {
			t.Error("Unexpected return result")
		}
	}
}

func TestRunMavenCommand(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRun("mvn foo bar")
	util.SkipWrapperCheck = true

	b := NewTestMavenBuilder()
	err := b.runMavenCommand(context.Background(), &bytes.Buffer{}, ".", []string{"foo", "bar"})

	testutil.CheckError(t, false, err)
}

func TestBuildJibMavenToDocker(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRun(
		"mvn -Djib.console=plain --foo --bar --non-recursive prepare-package jib:dockerBuild -Dimage=my-tag")
	util.SkipWrapperCheck = true

	b := NewTestMavenBuilder()
	a := &latest.JibMavenArtifact{Flags: []string{"--foo", "--bar"}}
	imageID, err := b.buildJibMavenToDocker(context.Background(), &bytes.Buffer{}, ".", a, "my-tag")

	testutil.CheckErrorAndDeepEqual(t, false, err, "image id from my-tag", imageID)
}

func TestBuildJibMavenToRegsitry(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunErr(
		"mvn -Djib.console=plain --foo --bar --non-recursive prepare-package jib:build -Dimage=my-tag",
		errors.New("fake error"),
	)
	util.SkipWrapperCheck = true

	b := NewTestMavenBuilder()
	a := &latest.JibMavenArtifact{Flags: []string{"--foo", "--bar"}}
	imageID, err := b.buildJibMavenToRegistry(context.Background(), &bytes.Buffer{}, ".", a, "my-tag")

	testutil.CheckErrorAndDeepEqual(t, true, err, "", imageID)
}
