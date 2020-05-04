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

package cache

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func depLister(files map[string][]string) DependencyLister {
	return func(_ context.Context, artifact *latest.Artifact) ([]string, error) {
		list, found := files[artifact.ImageName]
		if !found {
			return nil, errors.New("unknown artifact")
		}
		return list, nil
	}
}

type mockBuilder struct {
	built        []*latest.Artifact
	push         bool
	dockerDaemon docker.LocalDaemon
}

func (b *mockBuilder) BuildAndTest(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	var built []build.Artifact

	for _, artifact := range artifacts {
		b.built = append(b.built, artifact)
		tag := tags[artifact.ImageName]

		_, err := b.dockerDaemon.Build(ctx, out, artifact.Workspace, artifact.DockerArtifact, tag)
		if err != nil {
			return nil, err
		}

		if b.push {
			digest, err := b.dockerDaemon.Push(ctx, out, tag)
			if err != nil {
				return nil, err
			}

			built = append(built, build.Artifact{
				ImageName: artifact.ImageName,
				Tag:       build.TagWithDigest(tag, digest),
			})
		} else {
			built = append(built, build.Artifact{
				ImageName: artifact.ImageName,
				Tag:       tag,
			})
		}
	}

	return built, nil
}

type stubAuth struct{}

func (t stubAuth) GetAuthConfig(string) (types.AuthConfig, error)          { return types.AuthConfig{}, nil }
func (t stubAuth) GetAllAuthConfigs() (map[string]types.AuthConfig, error) { return nil, nil }

func TestCacheBuildLocal(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().
			Write("dep1", "content1").
			Write("dep2", "content2").
			Write("dep3", "content3").
			Chdir()

		runCtx := &runcontext.RunContext{
			Opts: config.SkaffoldOptions{
				CacheArtifacts: true,
				CacheFile:      tmpDir.Path("cache"),
			},
		}
		tags := map[string]string{
			"artifact1": "artifact1:tag1",
			"artifact2": "artifact2:tag2",
		}
		artifacts := []*latest.Artifact{
			{ImageName: "artifact1", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}},
			{ImageName: "artifact2", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}},
		}
		deps := depLister(map[string][]string{
			"artifact1": {"dep1", "dep2"},
			"artifact2": {"dep3"},
		})

		// Mock Docker
		t.Override(&docker.DefaultAuthHelper, stubAuth{})
		dockerDaemon := docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil)
		t.Override(&docker.NewAPIClient, func(*runcontext.RunContext) (docker.LocalDaemon, error) {
			return dockerDaemon, nil
		})

		// Create cache
		artifactCache, err := NewCache(runCtx, true, deps)
		t.CheckNoError(err)

		// First build: Need to build both artifacts
		builder := &mockBuilder{dockerDaemon: dockerDaemon, push: false}
		bRes, err := artifactCache.Build(context.Background(), ioutil.Discard, tags, artifacts, builder.BuildAndTest)

		t.CheckNoError(err)
		t.CheckDeepEqual(2, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))

		// Second build: both artifacts are read from cache
		// Artifacts should always be returned in their original order
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: false}
		bRes, err = artifactCache.Build(context.Background(), ioutil.Discard, tags, artifacts, builder.BuildAndTest)

		t.CheckNoError(err)
		t.CheckEmpty(builder.built)
		t.CheckDeepEqual(2, len(bRes))
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)

		// Third build: change first artifact's dependency
		// Artifacts should always be returned in their original order
		tmpDir.Write("dep1", "new content")
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: false}
		bRes, err = artifactCache.Build(context.Background(), ioutil.Discard, tags, artifacts, builder.BuildAndTest)

		t.CheckNoError(err)
		t.CheckDeepEqual(1, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)

		// Fourth build: change second artifact's dependency
		// Artifacts should always be returned in their original order
		tmpDir.Write("dep3", "new content")
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: false}
		bRes, err = artifactCache.Build(context.Background(), ioutil.Discard, tags, artifacts, builder.BuildAndTest)

		t.CheckNoError(err)
		t.CheckDeepEqual(1, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)
	})
}

func TestCacheBuildRemote(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().
			Write("dep1", "content1").
			Write("dep2", "content2").
			Write("dep3", "content3").
			Chdir()

		runCtx := &runcontext.RunContext{
			Opts: config.SkaffoldOptions{
				CacheArtifacts: true,
				CacheFile:      tmpDir.Path("cache"),
			},
		}
		tags := map[string]string{
			"artifact1": "artifact1:tag1",
			"artifact2": "artifact2:tag2",
		}
		artifacts := []*latest.Artifact{
			{ImageName: "artifact1", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}},
			{ImageName: "artifact2", ArtifactType: latest.ArtifactType{DockerArtifact: &latest.DockerArtifact{}}},
		}
		deps := depLister(map[string][]string{
			"artifact1": {"dep1", "dep2"},
			"artifact2": {"dep3"},
		})

		// Mock Docker
		dockerDaemon := docker.NewLocalDaemon(&testutil.FakeAPIClient{}, nil, false, nil)
		t.Override(&docker.NewAPIClient, func(*runcontext.RunContext) (docker.LocalDaemon, error) {
			return dockerDaemon, nil
		})
		t.Override(&docker.DefaultAuthHelper, stubAuth{})
		t.Override(&docker.RemoteDigest, func(ref string, _ map[string]bool) (string, error) {
			switch ref {
			case "artifact1:tag1":
				return "sha256:51ae7fa00c92525c319404a3a6d400e52ff9372c5a39cb415e0486fe425f3165", nil
			case "artifact2:tag2":
				return "sha256:35bdf2619f59e6f2372a92cb5486f4a0bf9b86e0e89ee0672864db6ed9c51539", nil
			default:
				return "", errors.New("unknown remote tag")
			}
		})

		// Create cache
		artifactCache, err := NewCache(runCtx, false, deps)
		t.CheckNoError(err)

		// First build: Need to build both artifacts
		builder := &mockBuilder{dockerDaemon: dockerDaemon, push: true}
		bRes, err := artifactCache.Build(context.Background(), ioutil.Discard, tags, artifacts, builder.BuildAndTest)

		t.CheckNoError(err)
		t.CheckDeepEqual(2, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))
		// Artifacts should always be returned in their original order
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)

		// Second build: both artifacts are read from cache
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: true}
		bRes, err = artifactCache.Build(context.Background(), ioutil.Discard, tags, artifacts, builder.BuildAndTest)

		t.CheckNoError(err)
		t.CheckEmpty(builder.built)
		t.CheckDeepEqual(2, len(bRes))
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)

		// Third build: change one artifact's dependencies
		tmpDir.Write("dep1", "new content")
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: true}
		bRes, err = artifactCache.Build(context.Background(), ioutil.Discard, tags, artifacts, builder.BuildAndTest)

		t.CheckNoError(err)
		t.CheckDeepEqual(1, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)
	})
}
