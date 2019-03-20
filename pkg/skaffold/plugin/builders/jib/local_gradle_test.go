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

func NewTestGradleBuilder() *GradleBuilder {
	localDaemon := docker.NewLocalDaemon(&testutil.FakeAPIClient{
		TagToImageID: map[string]string{
			"my-tag": "image id from my-tag",
		}}, nil)
	return &GradleBuilder{
		LocalDocker: localDaemon,
		opts:        &config.SkaffoldOptions{},
	}
}

func TestRunGradleCommand(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRun("gradle foo bar")
	util.SkipWrapperCheck = true

	b := NewTestGradleBuilder()
	err := b.runGradleCommand(context.Background(), &bytes.Buffer{}, ".", []string{"foo", "bar"})

	testutil.CheckError(t, false, err)
}

func TestBuildJibGradleToDocker(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRun(
		"gradle :proj:jibDockerBuild --image=my-tag --foo --bar")
	util.SkipWrapperCheck = true

	b := NewTestGradleBuilder()
	a := &latest.JibGradleArtifact{Project: "proj", Flags: []string{"--foo", "--bar"}}
	imageID, err := b.buildJibGradleToDocker(context.Background(), &bytes.Buffer{}, ".", a, "my-tag")

	testutil.CheckErrorAndDeepEqual(t, false, err, "image id from my-tag", imageID)
}

func TestBuildJibGradleToRegsitry(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunErr(
		"gradle :proj:jib --image=my-tag --foo --bar",
		errors.New("fake error"),
	)
	util.SkipWrapperCheck = true

	b := NewTestGradleBuilder()
	a := &latest.JibGradleArtifact{Project: "proj", Flags: []string{"--foo", "--bar"}}

	imageID, err := b.buildJibGradleToRegistry(
		context.Background(), &bytes.Buffer{}, ".", a, "my-tag")

	testutil.CheckErrorAndDeepEqual(t, true, err, "", imageID)
}
