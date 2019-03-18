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
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/docker/docker/api/types"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
)

type mockLocalDaemon struct {
}

func (d *mockLocalDaemon) Close() error {
	return nil
}

func (d *mockLocalDaemon) ExtraEnv() []string {
	return nil
}

func (d *mockLocalDaemon) ServerVersion(ctx context.Context) (types.Version, error) {
	return types.Version{}, nil
}

func (d *mockLocalDaemon) ConfigFile(ctx context.Context, image string) (*v1.ConfigFile, error) {
	return nil, nil
}

func (d *mockLocalDaemon) Build(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, ref string) (string, error) {
	return "", nil
}

func (d *mockLocalDaemon) Push(ctx context.Context, out io.Writer, ref string) (string, error) {
	return "", nil
}

func (d *mockLocalDaemon) Pull(ctx context.Context, out io.Writer, ref string) error {
	return nil
}

func (d *mockLocalDaemon) Load(ctx context.Context, out io.Writer, input io.Reader, ref string) (string, error) {
	return "", nil
}

func (d *mockLocalDaemon) Tag(ctx context.Context, image, ref string) error {
	return nil
}

func (d *mockLocalDaemon) ImageID(ctx context.Context, ref string) (string, error) {
	return "image id from " + ref, nil
}

func (d *mockLocalDaemon) RepoDigest(ctx context.Context, ref string) (string, error) {
	return "", nil
}

func (d *mockLocalDaemon) ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error) {
	return nil, nil
}

func (d *mockLocalDaemon) ImageExists(ctx context.Context, ref string) bool {
	return false
}

func TestRunGradleCommand(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRun("gradle foo bar")
	util.SkipWrapperCheck = true

	b := &GradleBuilder{
		LocalDocker: &mockLocalDaemon{},
		opts:        &config.SkaffoldOptions{},
	}
	err := b.runGradleCommand(context.Background(), &bytes.Buffer{}, ".", []string{"foo", "bar"})

	testutil.CheckError(t, false, err)
}

func TestBuildJibGradleToDocker(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	defer func(previous bool) { util.SkipWrapperCheck = previous }(util.SkipWrapperCheck)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRun(
		"gradle :proj:jibDockerBuild --image=my-tag --foo --bar")
	util.SkipWrapperCheck = true

	b := &GradleBuilder{
		LocalDocker: &mockLocalDaemon{},
		opts:        &config.SkaffoldOptions{},
	}
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

	b := &GradleBuilder{
		LocalDocker: &mockLocalDaemon{},
		opts:        &config.SkaffoldOptions{},
	}
	a := &latest.JibGradleArtifact{Project: "proj", Flags: []string{"--foo", "--bar"}}

	imageID, err := b.buildJibGradleToRegistry(
		context.Background(), &bytes.Buffer{}, ".", a, "my-tag")

	testutil.CheckErrorAndDeepEqual(t, true, err, "", imageID)
}
