/*
Copyright 2020 The Skaffold Authors

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
	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func buildpacksArtifact(builder, runImage string) *latest.Artifact {
	return &latest.Artifact{
		Workspace: ".",
		ArtifactType: latest.ArtifactType{
			BuildpackArtifact: &latest.BuildpackArtifact{
				Builder:           builder,
				RunImage:          runImage,
				ProjectDescriptor: "project.toml",
				Dependencies: &latest.BuildpackDependencies{
					Paths: []string{"."},
				},
			},
		},
	}
}

func withEnv(env []string, artifact *latest.Artifact) *latest.Artifact {
	artifact.BuildpackArtifact.Env = env
	return artifact
}

func withSync(sync *latest.Sync, artifact *latest.Artifact) *latest.Artifact {
	artifact.Sync = sync
	return artifact
}

func withTrustedBuilder(artifact *latest.Artifact) *latest.Artifact {
	artifact.BuildpackArtifact.TrustBuilder = true
	return artifact
}

func withBuildpacks(buildpacks []string, artifact *latest.Artifact) *latest.Artifact {
	artifact.BuildpackArtifact.Buildpacks = buildpacks
	return artifact
}

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, &buildpacksConfig{})
}

type buildpacksConfig struct {
	docker.Config
	devMode bool
}

func (c *buildpacksConfig) DevMode() bool { return c.devMode }
