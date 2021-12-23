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

package buildpacks

import (
	"context"
	"io"
	"io/ioutil"
	"testing"

	pack "github.com/buildpacks/pack/pkg/client"
	packcfg "github.com/buildpacks/pack/pkg/image"
	"github.com/docker/docker/api/types"
	"github.com/google/go-cmp/cmp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type fakePack struct {
	Opts pack.BuildOptions
}

func (f *fakePack) runPack(_ context.Context, _ io.Writer, _ docker.LocalDaemon, opts pack.BuildOptions) error {
	f.Opts = opts
	return nil
}

func TestBuild(t *testing.T) {
	tests := []struct {
		description     string
		artifact        *latestV1.Artifact
		tag             string
		api             *testutil.FakeAPIClient
		files           map[string]string
		pushImages      bool
		shouldErr       bool
		mode            config.RunMode
		resolver        ArtifactResolver
		expectedOptions *pack.BuildOptions
	}{
		{
			description: "success for debug",
			artifact:    buildpacksArtifact("my/builder", "my/run"),
			tag:         "img:tag",
			mode:        config.RunModes.Debug,
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			expectedOptions: &pack.BuildOptions{
				AppPath:  ".",
				Builder:  "my/builder",
				RunImage: "my/run",
				Env:      debugModeArgs,
				Image:    "img:latest",
			},
		},
		{
			description: "success for build",
			artifact:    buildpacksArtifact("my/builder", "my/run"),
			tag:         "img:tag",
			mode:        config.RunModes.Build,
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/builder",
				RunImage:   "my/run",
				PullPolicy: packcfg.PullNever,
				Env:        nonDebugModeArgs,
				Image:      "img:latest",
			},
		},

		{
			description: "success with buildpacks for debug",
			artifact:    withTrustedBuilder(withBuildpacks([]string{"my/buildpack", "my/otherBuildpack"}, buildpacksArtifact("my/otherBuilder", "my/otherRun"))),
			tag:         "img:tag",
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			mode:        config.RunModes.Debug,
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/otherBuilder",
				RunImage:   "my/otherRun",
				Buildpacks: []string{"my/buildpack", "my/otherBuildpack"},
				Env:        debugModeArgs,
				Image:      "img:latest",
			},
		},
		{
			description: "success with buildpacks for build",
			artifact:    withTrustedBuilder(withBuildpacks([]string{"my/buildpack", "my/otherBuildpack"}, buildpacksArtifact("my/otherBuilder", "my/otherRun"))),
			tag:         "img:tag",
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			mode:        config.RunModes.Build,
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/otherBuilder",
				RunImage:   "my/otherRun",
				Buildpacks: []string{"my/buildpack", "my/otherBuildpack"},
				PullPolicy: packcfg.PullNever,
				Env:        nonDebugModeArgs,
				Image:      "img:latest",
			},
		},
		{
			description: "project.toml",
			artifact:    buildpacksArtifact("my/builder2", "my/run2"),
			tag:         "img:tag",
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			mode:        config.RunModes.Build,
			files: map[string]string{
				"project.toml": `[[build.env]]
name = "GOOGLE_RUNTIME_VERSION"
value = "14.3.0"
[[build.buildpacks]]
id = "my/buildpack"
[[build.buildpacks]]
id = "my/otherBuildpack"
version = "1.0"
`,
			},
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/builder2",
				RunImage:   "my/run2",
				Buildpacks: []string{"my/buildpack", "my/otherBuildpack@1.0"},
				Env: addDefaultArgs(config.RunModes.Build, map[string]string{
					"GOOGLE_RUNTIME_VERSION": "14.3.0",
				}),
				Image: "img:latest",
			},
		},
		{
			description: "Buildpacks in skaffold.yaml override those in project.toml",
			artifact:    withBuildpacks([]string{"my/buildpack", "my/otherBuildpack"}, buildpacksArtifact("my/builder3", "my/run3")),
			tag:         "img:tag",
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			files: map[string]string{
				"project.toml": `[[build.buildpacks]]
id = "my/ignored"
`,
			},
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/builder3",
				RunImage:   "my/run3",
				Buildpacks: []string{"my/buildpack", "my/otherBuildpack"},
				Env:        nonDebugModeArgs,
				Image:      "img:latest",
			},
		},
		{
			description: "Combine env from skaffold.yaml and project.toml",
			artifact:    withEnv([]string{"KEY1=VALUE1"}, buildpacksArtifact("my/builder4", "my/run4")),
			tag:         "img:tag",
			mode:        config.RunModes.Build,
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			files: map[string]string{
				"project.toml": `[[build.env]]
name = "KEY2"
value = "VALUE2"
`,
			},
			expectedOptions: &pack.BuildOptions{
				AppPath:  ".",
				Builder:  "my/builder4",
				RunImage: "my/run4",
				Env: addDefaultArgs(config.RunModes.Build, map[string]string{
					"KEY1": "VALUE1",
					"KEY2": "VALUE2",
				}),
				Image: "img:latest",
			},
		},
		{
			description: "dev mode",
			artifact:    withSync(&latestV1.Sync{Auto: util.BoolPtr(true)}, buildpacksArtifact("another/builder", "another/run")),
			tag:         "img:tag",
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			mode:        config.RunModes.Dev,
			expectedOptions: &pack.BuildOptions{
				AppPath:  ".",
				Builder:  "another/builder",
				RunImage: "another/run",
				Env: addDefaultArgs(config.RunModes.Build, map[string]string{
					"GOOGLE_DEVMODE": "1",
				}),
				Image: "img:latest",
			},
		},
		{
			description: "dev mode but no sync",
			artifact:    buildpacksArtifact("my/other-builder", "my/run"),
			tag:         "img:tag",
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			mode:        config.RunModes.Dev,
			expectedOptions: &pack.BuildOptions{
				AppPath:  ".",
				Builder:  "my/other-builder",
				RunImage: "my/run",
				Env:      nonDebugModeArgs,
				Image:    "img:latest",
			},
		},
		{
			description: "invalid ref",
			artifact:    buildpacksArtifact("my/builder", "my/run"),
			tag:         "in valid ref",
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			shouldErr:   true,
		},
		{
			description: "push error",
			artifact:    buildpacksArtifact("my/builder", "my/run"),
			tag:         "img:tag",
			pushImages:  true,
			api: &testutil.FakeAPIClient{
				ErrImagePush: true,
			},
			resolver:  mockArtifactResolver{},
			shouldErr: true,
		},
		{
			description: "invalid env",
			artifact:    withEnv([]string{"INVALID"}, buildpacksArtifact("my/builder", "my/run")),
			tag:         "img:tag",
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			shouldErr:   true,
		},
		{
			description: "invalid project.toml",
			artifact:    buildpacksArtifact("my/builder2", "my/run2"),
			tag:         "img:tag",
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{},
			files: map[string]string{
				"project.toml": `INVALID`,
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("file").WriteFiles(test.files).Chdir()
			pack := &fakePack{}
			t.Override(&runPackBuildFunc, pack.runPack)

			test.api.
				Add(test.artifact.BuildpackArtifact.Builder, "builderImageID").
				Add(test.artifact.BuildpackArtifact.RunImage, "runImageID").
				Add("img:latest", "builtImageID")
			localDocker := fakeLocalDaemon(test.api)

			builder := NewArtifactBuilder(localDocker, test.pushImages, test.mode, test.resolver)
			_, err := builder.Build(context.Background(), ioutil.Discard, test.artifact, test.tag)

			t.CheckError(test.shouldErr, err)
			if test.expectedOptions != nil {
				t.CheckDeepEqual(*test.expectedOptions, pack.Opts, ignoreField("ProjectDescriptor.SchemaVersion"), ignoreField("TrustBuilder"))
			}
		})
	}
}

func TestBuildWithArtifactDependencies(t *testing.T) {
	tests := []struct {
		description     string
		artifact        *latestV1.Artifact
		tag             string
		api             *testutil.FakeAPIClient
		files           map[string]string
		pushImages      bool
		shouldErr       bool
		mode            config.RunMode
		resolver        ArtifactResolver
		expectedOptions *pack.BuildOptions
	}{
		{
			description: "custom builder image only with no push",
			artifact:    withRequiredArtifacts([]*latestV1.ArtifactDependency{{ImageName: "builder-image", Alias: "BUILDER_IMAGE"}}, buildpacksArtifact("BUILDER_IMAGE", "my/run")),
			tag:         "img:tag",
			pushImages:  false,
			mode:        config.RunModes.Build,
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{m: map[string]string{"builder-image": "my/custom-builder"}},
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/custom-builder",
				RunImage:   "my/run",
				PullPolicy: packcfg.PullIfNotPresent,
				Env:        nonDebugModeArgs,
				Image:      "img:latest",
			},
		},
		{
			description: "custom run image only with no push",
			artifact:    withRequiredArtifacts([]*latestV1.ArtifactDependency{{ImageName: "run-image", Alias: "RUN_IMAGE"}}, buildpacksArtifact("my/builder", "RUN_IMAGE")),
			tag:         "img:tag",
			pushImages:  false,
			mode:        config.RunModes.Build,
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{m: map[string]string{"run-image": "my/custom-run"}},
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/builder",
				RunImage:   "my/custom-run",
				PullPolicy: packcfg.PullIfNotPresent,
				Env:        nonDebugModeArgs,
				Image:      "img:latest",
			},
		},
		{
			description: "custom builder image only with push",
			artifact:    withRequiredArtifacts([]*latestV1.ArtifactDependency{{ImageName: "builder-image", Alias: "BUILDER_IMAGE"}}, buildpacksArtifact("BUILDER_IMAGE", "my/run")),
			tag:         "img:tag",
			pushImages:  true,
			mode:        config.RunModes.Build,
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{m: map[string]string{"builder-image": "my/custom-builder"}},
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/custom-builder",
				RunImage:   "my/run",
				PullPolicy: packcfg.PullIfNotPresent,
				Env:        nonDebugModeArgs,
				Image:      "img:latest",
			},
		},
		{
			description: "custom run image only with push",
			artifact:    withRequiredArtifacts([]*latestV1.ArtifactDependency{{ImageName: "run-image", Alias: "RUN_IMAGE"}}, buildpacksArtifact("my/builder", "RUN_IMAGE")),
			tag:         "img:tag",
			pushImages:  true,
			mode:        config.RunModes.Build,
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{m: map[string]string{"run-image": "my/custom-run"}},
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/builder",
				RunImage:   "my/custom-run",
				PullPolicy: packcfg.PullIfNotPresent,
				Env:        nonDebugModeArgs,
				Image:      "img:latest",
			},
		},
		{
			description: "custom run image and custom builder image with push",
			artifact:    withRequiredArtifacts([]*latestV1.ArtifactDependency{{ImageName: "run-image", Alias: "RUN_IMAGE"}, {ImageName: "builder-image", Alias: "BUILDER_IMAGE"}}, buildpacksArtifact("BUILDER_IMAGE", "RUN_IMAGE")),
			tag:         "img:tag",
			pushImages:  true,
			mode:        config.RunModes.Build,
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{m: map[string]string{"builder-image": "my/custom-builder", "run-image": "my/custom-run"}},
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/custom-builder",
				RunImage:   "my/custom-run",
				PullPolicy: packcfg.PullNever,
				Env:        nonDebugModeArgs,
				Image:      "img:latest",
			},
		},
		{
			description: "custom run image and custom builder image with no push",
			artifact:    withRequiredArtifacts([]*latestV1.ArtifactDependency{{ImageName: "run-image", Alias: "RUN_IMAGE"}, {ImageName: "builder-image", Alias: "BUILDER_IMAGE"}}, buildpacksArtifact("BUILDER_IMAGE", "RUN_IMAGE")),
			tag:         "img:tag",
			pushImages:  false,
			mode:        config.RunModes.Build,
			api:         &testutil.FakeAPIClient{},
			resolver:    mockArtifactResolver{m: map[string]string{"builder-image": "my/custom-builder", "run-image": "my/custom-run"}},
			expectedOptions: &pack.BuildOptions{
				AppPath:    ".",
				Builder:    "my/custom-builder",
				RunImage:   "my/custom-run",
				PullPolicy: packcfg.PullNever,
				Env:        nonDebugModeArgs,
				Image:      "img:latest",
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().Touch("file").WriteFiles(test.files).Chdir()
			pack := &fakePack{}
			t.Override(&runPackBuildFunc, pack.runPack)
			images.images = map[imageTuple]bool{}
			test.api.
				Add(test.artifact.BuildpackArtifact.Builder, "builderImageID").
				Add(test.artifact.BuildpackArtifact.RunImage, "runImageID").
				Add("img:latest", "builtImageID")
			t.Override(&docker.DefaultAuthHelper, testAuthHelper{})

			localDocker := fakeLocalDaemon(test.api)

			builder := NewArtifactBuilder(localDocker, test.pushImages, test.mode, test.resolver)
			_, err := builder.Build(context.Background(), ioutil.Discard, test.artifact, test.tag)

			t.CheckError(test.shouldErr, err)
			if test.expectedOptions != nil {
				t.CheckDeepEqual(*test.expectedOptions, pack.Opts, ignoreField("ProjectDescriptor.SchemaVersion"), ignoreField("TrustBuilder"))
			}
		})
	}
}
func buildpacksArtifact(builder, runImage string) *latestV1.Artifact {
	return &latestV1.Artifact{
		Workspace: ".",
		ArtifactType: latestV1.ArtifactType{
			BuildpackArtifact: &latestV1.BuildpackArtifact{
				Builder:           builder,
				RunImage:          runImage,
				ProjectDescriptor: "project.toml",
				Dependencies: &latestV1.BuildpackDependencies{
					Paths: []string{"."},
				},
			},
		},
	}
}

func withEnv(env []string, artifact *latestV1.Artifact) *latestV1.Artifact {
	artifact.BuildpackArtifact.Env = env
	return artifact
}

func withSync(sync *latestV1.Sync, artifact *latestV1.Artifact) *latestV1.Artifact {
	artifact.Sync = sync
	return artifact
}

func withTrustedBuilder(artifact *latestV1.Artifact) *latestV1.Artifact {
	artifact.BuildpackArtifact.TrustBuilder = true
	return artifact
}

func withRequiredArtifacts(deps []*latestV1.ArtifactDependency, artifact *latestV1.Artifact) *latestV1.Artifact {
	artifact.Dependencies = deps
	return artifact
}

func withBuildpacks(buildpacks []string, artifact *latestV1.Artifact) *latestV1.Artifact {
	artifact.BuildpackArtifact.Buildpacks = buildpacks
	return artifact
}

type mockArtifactResolver struct {
	m map[string]string
}

func (r mockArtifactResolver) GetImageTag(imageName string) (string, bool) {
	if r.m == nil {
		return "", false
	}
	val, found := r.m[imageName]
	return val, found
}

type testAuthHelper struct{}

func (t testAuthHelper) GetAuthConfig(string) (types.AuthConfig, error) {
	return types.AuthConfig{}, nil
}
func (t testAuthHelper) GetAllAuthConfigs(context.Context) (map[string]types.AuthConfig, error) {
	return nil, nil
}

func ignoreField(path string) cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		return p.String() == path
	}, cmp.Ignore())
}
