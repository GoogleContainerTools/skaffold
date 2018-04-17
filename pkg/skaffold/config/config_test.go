/*
Copyright 2018 Google LLC

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

package config

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	minimalConfig = `
apiVersion: skaffold/v1alpha2
kind: Config
`
	simpleConfig = `
apiVersion: skaffold/v1alpha2
kind: Config
build:
  artifacts:
  - imageName: example
deploy:
  name: example
`
	completeConfig = `
apiVersion: skaffold/v1alpha2
kind: Config
build:
  tagPolicy:
    sha256: {}
  artifacts:
  - imageName: image1
    workspace: ./examples/app1
    docker:
      dockerfilePath: Dockerfile.dev
  - imageName: image2
    workspace: ./examples/app2
    bazel:
      target: //:example.tar
  googleCloudBuild:
    projectId: ID
deploy:
  name: example
`
	badConfig = "bad config"
)

func TestParseConfig(t *testing.T) {
	var tests = []struct {
		description string
		config      string
		dev         bool
		expected    util.VersionedConfig
		badReader   bool
		shouldErr   bool
	}{
		{
			description: "Minimal config for dev",
			config:      minimalConfig,
			dev:         true,
			expected: config(
				withLocalBuild(
					withTagPolicy(v1alpha2.TagPolicy{ShaTagger: &v1alpha2.ShaTagger{}}),
				),
			),
		},
		{
			description: "Minimal config for run",
			config:      minimalConfig,
			dev:         false,
			expected: config(
				withLocalBuild(
					withTagPolicy(v1alpha2.TagPolicy{GitTagger: &v1alpha2.GitTagger{}}),
				),
			),
		},
		{
			description: "Simple config for dev",
			config:      simpleConfig,
			dev:         true,
			expected: config(
				withLocalBuild(
					withTagPolicy(v1alpha2.TagPolicy{ShaTagger: &v1alpha2.ShaTagger{}}),
					withDockerArtifact("example", ".", "Dockerfile"),
				),
				withDeploy("example"),
			),
		},
		{
			description: "Simple config for run",
			config:      simpleConfig,
			dev:         false,
			expected: config(
				withLocalBuild(
					withTagPolicy(v1alpha2.TagPolicy{GitTagger: &v1alpha2.GitTagger{}}),
					withDockerArtifact("example", ".", "Dockerfile"),
				),
				withDeploy("example"),
			),
		},
		{
			description: "Complete config for dev",
			config:      completeConfig,
			dev:         true,
			expected: config(
				withGCBBuild("ID",
					withTagPolicy(v1alpha2.TagPolicy{ShaTagger: &v1alpha2.ShaTagger{}}),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
					withBazelArtifact("image2", "./examples/app2", "//:example.tar"),
				),
				withDeploy("example"),
			),
		},
		{
			description: "Complete config for run",
			config:      completeConfig,
			dev:         false,
			expected: config(
				withGCBBuild("ID",
					withTagPolicy(v1alpha2.TagPolicy{ShaTagger: &v1alpha2.ShaTagger{}}),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
					withBazelArtifact("image2", "./examples/app2", "//:example.tar"),
				),
				withDeploy("example"),
			),
		},
		{
			description: "Bad config",
			config:      badConfig,
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg, err := GetConfig([]byte(test.config), true, test.dev)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, cfg)
		})
	}
}

func config(ops ...func(*SkaffoldConfig)) *SkaffoldConfig {
	cfg := &SkaffoldConfig{APIVersion: "skaffold/v1alpha2", Kind: "Config"}
	for _, op := range ops {
		op(cfg)
	}
	return cfg
}

func withLocalBuild(ops ...func(*v1alpha2.BuildConfig)) func(*SkaffoldConfig) {
	return func(cfg *SkaffoldConfig) {
		b := v1alpha2.BuildConfig{BuildType: v1alpha2.BuildType{LocalBuild: &v1alpha2.LocalBuild{}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withGCBBuild(id string, ops ...func(*v1alpha2.BuildConfig)) func(*SkaffoldConfig) {
	return func(cfg *SkaffoldConfig) {
		b := v1alpha2.BuildConfig{BuildType: v1alpha2.BuildType{GoogleCloudBuild: &v1alpha2.GoogleCloudBuild{ProjectID: id}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withDeploy(name string, ops ...func(*v1alpha2.DeployConfig)) func(*SkaffoldConfig) {
	return func(cfg *SkaffoldConfig) {
		d := v1alpha2.DeployConfig{Name: name}
		for _, op := range ops {
			op(&d)
		}
		cfg.Deploy = d
	}
}

func withDockerArtifact(image, workspace, dockerfile string) func(*v1alpha2.BuildConfig) {
	return func(cfg *v1alpha2.BuildConfig) {
		cfg.Artifacts = append(cfg.Artifacts, &v1alpha2.Artifact{
			ImageName: image,
			Workspace: workspace,
			ArtifactType: v1alpha2.ArtifactType{
				DockerArtifact: &v1alpha2.DockerArtifact{
					DockerfilePath: dockerfile,
				},
			},
		})
	}
}

func withBazelArtifact(image, workspace, target string) func(*v1alpha2.BuildConfig) {
	return func(cfg *v1alpha2.BuildConfig) {
		cfg.Artifacts = append(cfg.Artifacts, &v1alpha2.Artifact{
			ImageName: image,
			Workspace: workspace,
			ArtifactType: v1alpha2.ArtifactType{
				BazelArtifact: &v1alpha2.BazelArtifact{
					BuildTarget: target,
				},
			},
		})
	}
}

func withTagPolicy(tagPolicy v1alpha2.TagPolicy) func(*v1alpha2.BuildConfig) {
	return func(cfg *v1alpha2.BuildConfig) { cfg.TagPolicy = tagPolicy }
}
