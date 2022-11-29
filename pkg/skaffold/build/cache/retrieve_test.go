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
	"testing"

	"github.com/docker/docker/api/types"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
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

type mockArtifactStore map[string]string

func (m mockArtifactStore) GetImageTag(imageName string) (string, bool) { return m[imageName], true }
func (m mockArtifactStore) Record(a *latest.Artifact, tag string)       { m[a.ImageName] = tag }
func (m mockArtifactStore) GetArtifacts([]*latest.Artifact) ([]graph.Artifact, error) {
	return nil, nil
}

type mockBuilder struct {
	built        []*latest.Artifact
	push         bool
	dockerDaemon docker.LocalDaemon
	store        build.ArtifactStore
	cache        Cache
}

func (b *mockBuilder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, _ platform.Resolver) ([]graph.Artifact, error) {
	var built []graph.Artifact

	for _, artifact := range artifacts {
		b.built = append(b.built, artifact)
		tag := tags[artifact.ImageName]
		opts := docker.BuildOptions{Tag: tag, Mode: config.RunModes.Dev}
		_, err := b.dockerDaemon.Build(ctx, out, artifact.Workspace, artifact.ImageName, artifact.DockerArtifact, opts)
		if err != nil {
			return nil, err
		}

		if b.push {
			digest, err := b.dockerDaemon.Push(ctx, out, tag)
			if err != nil {
				return nil, err
			}

			a := graph.Artifact{
				ImageName: artifact.ImageName,
				Tag:       build.TagWithDigest(tag, digest),
			}

			built = append(built, a)
			b.cache.AddArtifact(ctx, a)
		} else {
			a := graph.Artifact{
				ImageName: artifact.ImageName,
				Tag:       tag,
			}
			built = append(built, a)
			b.store.Record(artifact, tag)
			b.cache.AddArtifact(ctx, a)
		}
	}

	return built, nil
}

type stubAuth struct{}

func (t stubAuth) GetAuthConfig(string) (types.AuthConfig, error) {
	return types.AuthConfig{}, nil
}
func (t stubAuth) GetAllAuthConfigs(context.Context) (map[string]types.AuthConfig, error) {
	return nil, nil
}

func TestCacheBuildLocal(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().
			Write("dep1", "content1").
			Write("dep2", "content2").
			Write("dep3", "content3").
			Chdir()

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
		dockerDaemon := fakeLocalDaemon(&testutil.FakeAPIClient{})
		t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
			return dockerDaemon, nil
		})

		// Mock args builder
		t.Override(&docker.EvalBuildArgs, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string) (map[string]*string, error) {
			return args, nil
		})
		t.Override(&docker.EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
			return args, nil
		})

		// Create cache
		cfg := &mockConfig{
			pipeline:  latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{TryImportMissing: false}}}},
			cacheFile: tmpDir.Path("cache"),
		}
		store := make(mockArtifactStore)
		artifactCache, err := NewCache(context.Background(), cfg, func(imageName string) (bool, error) { return true, nil }, deps, graph.ToArtifactGraph(artifacts), store)
		t.CheckNoError(err)

		// First build: Need to build both artifacts
		builder := &mockBuilder{dockerDaemon: dockerDaemon, push: false, store: store, cache: artifactCache}
		bRes, err := artifactCache.Build(context.Background(), io.Discard, tags, artifacts, platform.Resolver{}, builder.Build)

		t.CheckNoError(err)
		t.CheckDeepEqual(2, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))

		// Second build: both artifacts are read from cache
		// Artifacts should always be returned in their original order
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: false, store: store, cache: artifactCache}
		bRes, err = artifactCache.Build(context.Background(), io.Discard, tags, artifacts, platform.Resolver{}, builder.Build)

		t.CheckNoError(err)
		t.CheckEmpty(builder.built)
		t.CheckDeepEqual(2, len(bRes))
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)

		// Third build: change first artifact's dependency
		// Artifacts should always be returned in their original order
		tmpDir.Write("dep1", "new content")
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: false, store: store, cache: artifactCache}
		bRes, err = artifactCache.Build(context.Background(), io.Discard, tags, artifacts, platform.Resolver{}, builder.Build)

		t.CheckNoError(err)
		t.CheckDeepEqual(1, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)

		// Fourth build: change second artifact's dependency
		// Artifacts should always be returned in their original order
		tmpDir.Write("dep3", "new content")
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: false, store: store, cache: artifactCache}
		bRes, err = artifactCache.Build(context.Background(), io.Discard, tags, artifacts, platform.Resolver{}, builder.Build)

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
		dockerDaemon := fakeLocalDaemon(&testutil.FakeAPIClient{})
		t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
			return dockerDaemon, nil
		})
		t.Override(&docker.DefaultAuthHelper, stubAuth{})
		t.Override(&docker.RemoteDigest, func(ref string, _ docker.Config, _ []specs.Platform) (string, error) {
			switch ref {
			case "artifact1:tag1":
				return "sha256:51ae7fa00c92525c319404a3a6d400e52ff9372c5a39cb415e0486fe425f3165", nil
			case "artifact2:tag2":
				return "sha256:35bdf2619f59e6f2372a92cb5486f4a0bf9b86e0e89ee0672864db6ed9c51539", nil
			default:
				return "", errors.New("unknown remote tag")
			}
		})

		// Mock args builder
		t.Override(&docker.EvalBuildArgs, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string) (map[string]*string, error) {
			return args, nil
		})
		t.Override(&docker.EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
			return args, nil
		})

		// Create cache
		cfg := &mockConfig{
			pipeline:  latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{TryImportMissing: false}}}},
			cacheFile: tmpDir.Path("cache"),
		}
		artifactCache, err := NewCache(context.Background(), cfg, func(imageName string) (bool, error) { return false, nil }, deps, graph.ToArtifactGraph(artifacts), make(mockArtifactStore))
		t.CheckNoError(err)

		// First build: Need to build both artifacts
		builder := &mockBuilder{dockerDaemon: dockerDaemon, push: true, cache: artifactCache}
		bRes, err := artifactCache.Build(context.Background(), io.Discard, tags, artifacts, platform.Resolver{}, builder.Build)

		t.CheckNoError(err)
		t.CheckDeepEqual(2, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))
		// Artifacts should always be returned in their original order
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)

		// Second build: both artifacts are read from cache
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: true, cache: artifactCache}
		bRes, err = artifactCache.Build(context.Background(), io.Discard, tags, artifacts, platform.Resolver{}, builder.Build)

		t.CheckNoError(err)
		t.CheckEmpty(builder.built)
		t.CheckDeepEqual(2, len(bRes))
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)

		// Third build: change one artifact's dependencies
		tmpDir.Write("dep1", "new content")
		builder = &mockBuilder{dockerDaemon: dockerDaemon, push: true, cache: artifactCache}
		bRes, err = artifactCache.Build(context.Background(), io.Discard, tags, artifacts, platform.Resolver{}, builder.Build)

		t.CheckNoError(err)
		t.CheckDeepEqual(1, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)
	})
}

func TestCacheFindMissing(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().
			Write("dep1", "content1").
			Write("dep2", "content2").
			Write("dep3", "content3").
			Chdir()

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
		dockerDaemon := fakeLocalDaemon(&testutil.FakeAPIClient{})
		t.Override(&docker.NewAPIClient, func(context.Context, docker.Config) (docker.LocalDaemon, error) {
			return dockerDaemon, nil
		})
		t.Override(&docker.DefaultAuthHelper, stubAuth{})
		t.Override(&docker.RemoteDigest, func(ref string, _ docker.Config, _ []specs.Platform) (string, error) {
			switch ref {
			case "artifact1:tag1":
				return "sha256:51ae7fa00c92525c319404a3a6d400e52ff9372c5a39cb415e0486fe425f3165", nil
			case "artifact2:tag2":
				return "sha256:35bdf2619f59e6f2372a92cb5486f4a0bf9b86e0e89ee0672864db6ed9c51539", nil
			default:
				return "", errors.New("unknown remote tag")
			}
		})

		// Mock args builder
		t.Override(&docker.EvalBuildArgs, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string) (map[string]*string, error) {
			return args, nil
		})
		t.Override(&docker.EvalBuildArgsWithEnv, func(_ config.RunMode, _ string, _ string, args map[string]*string, _ map[string]*string, _ map[string]string) (map[string]*string, error) {
			return args, nil
		})

		// Create cache
		cfg := &mockConfig{
			pipeline:  latest.Pipeline{Build: latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{TryImportMissing: true}}}},
			cacheFile: tmpDir.Path("cache"),
		}
		artifactCache, err := NewCache(context.Background(), cfg, func(imageName string) (bool, error) { return false, nil }, deps, graph.ToArtifactGraph(artifacts), make(mockArtifactStore))
		t.CheckNoError(err)

		// Because the artifacts are in the docker registry, we expect them to be imported correctly.
		builder := &mockBuilder{dockerDaemon: dockerDaemon, push: false, store: make(mockArtifactStore)}
		bRes, err := artifactCache.Build(context.Background(), io.Discard, tags, artifacts, platform.Resolver{}, builder.Build)

		t.CheckNoError(err)
		t.CheckDeepEqual(0, len(builder.built))
		t.CheckDeepEqual(2, len(bRes))
		// Artifacts should always be returned in their original order
		t.CheckDeepEqual("artifact1", bRes[0].ImageName)
		t.CheckDeepEqual("artifact2", bRes[1].ImageName)
	})
}

type mockConfig struct {
	runcontext.RunContext // Embedded to provide the default values.
	cacheFile             string
	mode                  config.RunMode
	pipeline              latest.Pipeline
}

func (c *mockConfig) CacheArtifacts() bool                            { return true }
func (c *mockConfig) CacheFile() string                               { return c.cacheFile }
func (c *mockConfig) Mode() config.RunMode                            { return c.mode }
func (c *mockConfig) PipelineForImage(string) (latest.Pipeline, bool) { return c.pipeline, true }
