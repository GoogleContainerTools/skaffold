/*
Copyright 2018 The Skaffold Authors

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
	"k8s.io/client-go/tools/clientcmd/api"
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
  tagPolicy:
    gitCommit: {}
  artifacts:
  - imageName: example
deploy:
  kubectl: {}
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
  kubectl:
   manifests:
   - dep.yaml
   - svc.yaml
`
	minimalKanikoConfig = `
apiVersion: skaffold/v1alpha2
kind: Config
build:
  kaniko:
    gcsBucket: demo
`
	completeKanikoConfig = `
apiVersion: skaffold/v1alpha2
kind: Config
build:
  kaniko:
    gcsBucket: demo
    pullSecret: /secret.json
    pullSecretName: secret-name
    namespace: nskaniko
    timeout: 120m
`
	badConfig = "bad config"
)

func TestParseConfig(t *testing.T) {
	cleanup := testutil.SetupFakeKubernetesContext(t, api.Config{CurrentContext: "cluster1"})
	defer cleanup()

	var tests = []struct {
		description string
		config      string
		expected    util.VersionedConfig
		badReader   bool
		shouldErr   bool
	}{
		{
			description: "Minimal config",
			config:      minimalConfig,
			expected: config(
				withLocalBuild(
					withTagPolicy(v1alpha2.TagPolicy{GitTagger: &v1alpha2.GitTagger{}}),
				),
			),
		},
		{
			description: "Simple config",
			config:      simpleConfig,
			expected: config(
				withLocalBuild(
					withTagPolicy(v1alpha2.TagPolicy{GitTagger: &v1alpha2.GitTagger{}}),
					withDockerArtifact("example", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			description: "Complete config",
			config:      completeConfig,
			expected: config(
				withGCBBuild("ID",
					withTagPolicy(v1alpha2.TagPolicy{ShaTagger: &v1alpha2.ShaTagger{}}),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
					withBazelArtifact("image2", "./examples/app2", "//:example.tar"),
				),
				withKubectlDeploy("dep.yaml", "svc.yaml"),
			),
		},
		{
			description: "Minimal Kaniko config",
			config:      minimalKanikoConfig,
			expected: config(
				withKanikoBuild("demo", "kaniko-secret", "default", "", "20m",
					withTagPolicy(v1alpha2.TagPolicy{GitTagger: &v1alpha2.GitTagger{}}),
				),
			),
		},
		{
			description: "Complete Kaniko config",
			config:      completeKanikoConfig,
			expected: config(
				withKanikoBuild("demo", "secret-name", "nskaniko", "/secret.json", "120m",
					withTagPolicy(v1alpha2.TagPolicy{GitTagger: &v1alpha2.GitTagger{}}),
				),
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
			cfg, err := GetConfig([]byte(test.config), true)
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

func withKanikoBuild(bucket, secretName, namespace, secret string, timeout string, ops ...func(*v1alpha2.BuildConfig)) func(*SkaffoldConfig) {
	return func(cfg *SkaffoldConfig) {
		b := v1alpha2.BuildConfig{BuildType: v1alpha2.BuildType{KanikoBuild: &v1alpha2.KanikoBuild{
			GCSBucket:      bucket,
			PullSecretName: secretName,
			Namespace:      namespace,
			PullSecret:     secret,
			Timeout:        timeout,
		}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withKubectlDeploy(manifests ...string) func(*SkaffoldConfig) {
	return func(cfg *SkaffoldConfig) {
		cfg.Deploy = v1alpha2.DeployConfig{
			DeployType: v1alpha2.DeployType{
				KubectlDeploy: &v1alpha2.KubectlDeploy{
					Manifests: manifests,
				},
			},
		}
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
