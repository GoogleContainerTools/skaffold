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

package schema

import (
	"fmt"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	minimalConfig = ``

	simpleConfig = `
build:
  tagPolicy:
    gitCommit: {}
  artifacts:
  - image: example
deploy:
  kubectl: {}
`
	// This config has two tag policies set.
	invalidConfig = `
build:
  tagPolicy:
    sha256: {}
    gitCommit: {}
  artifacts:
  - image: example
deploy:
  name: example
`

	completeConfig = `
build:
  tagPolicy:
    sha256: {}
  artifacts:
  - image: image1
    context: ./examples/app1
    docker:
      dockerfile: Dockerfile.dev
  - image: image2
    context: ./examples/app2
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
build:
  artifacts:
  - image: image1
    context: ./examples/app1
    kaniko:
      buildContext:
        gcsBucket: demo
  cluster: {}
`
	completeKanikoConfig = `
build:
  artifacts:
  - image: image1
    context: ./examples/app1
    kaniko:
      buildContext:
        localDir: {}
  cluster:
    pullSecret: /secret.json
    pullSecretName: secret-name
    namespace: nskaniko
    timeout: 120m
    dockerConfig:
      secretName: config-name
      path: /kaniko/.docker
`
	badConfig = "bad config"

	invalidStatusCheckConfig = `
deploy:
  statusCheckDeadlineSeconds: s
`
	validStatusCheckConfig = `
deploy:
  statusCheckDeadlineSeconds: 10
`
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		apiVersion  string
		description string
		config      string
		expected    util.VersionedConfig
		shouldErr   bool
	}{
		{
			apiVersion:  latest.Version,
			description: "Minimal config",
			config:      minimalConfig,
			expected: config(
				withLocalBuild(
					withGitTagger(),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			apiVersion:  "skaffold/v1alpha1",
			description: "Old minimal config",
			config:      minimalConfig,
			expected: config(
				withLocalBuild(
					withGitTagger(),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			apiVersion:  latest.Version,
			description: "Simple config",
			config:      simpleConfig,
			expected: config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("example", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			apiVersion:  latest.Version,
			description: "Complete config",
			config:      completeConfig,
			expected: config(
				withGoogleCloudBuild("ID",
					withShaTagger(),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
					withBazelArtifact("image2", "./examples/app2", "//:example.tar"),
				),
				withKubectlDeploy("dep.yaml", "svc.yaml"),
			),
		},
		{
			apiVersion:  latest.Version,
			description: "Minimal Kaniko config",
			config:      minimalKanikoConfig,
			expected: config(
				withClusterBuild("", "/secret", "default", "", "20m",
					withGitTagger(),
					withKanikoArtifact("image1", "./examples/app1", "Dockerfile", "demo"),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			apiVersion:  latest.Version,
			description: "Complete Kaniko config",
			config:      completeKanikoConfig,
			expected: config(
				withClusterBuild("secret-name", "/secret", "nskaniko", "/secret.json", "120m",
					withGitTagger(),
					withDockerConfig("config-name", "/kaniko/.docker"),
					withKanikoArtifact("image1", "./examples/app1", "Dockerfile", ""),
				),
				withKubectlDeploy("k8s/*.yaml"),
			),
		},
		{
			apiVersion:  latest.Version,
			description: "Bad config",
			config:      badConfig,
			shouldErr:   true,
		},
		{
			apiVersion:  latest.Version,
			description: "two taggers defined",
			config:      invalidConfig,
			shouldErr:   true,
		},
		{
			apiVersion:  "",
			description: "ApiVersion not specified",
			config:      minimalConfig,
			shouldErr:   true,
		},
		{
			apiVersion:  latest.Version,
			description: "invalid statusCheckDeadline",
			config:      invalidStatusCheckConfig,
			shouldErr:   true,
		},
		{
			apiVersion:  latest.Version,
			description: "valid statusCheckDeadline",
			config:      validStatusCheckConfig,
			expected: config(
				withLocalBuild(
					withGitTagger(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withStatusCheckDeadline(10),
			),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			tmpDir := t.NewTempDir().
				Write("skaffold.yaml", fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", test.apiVersion, test.config))

			cfg, err := ParseConfig(tmpDir.Path("skaffold.yaml"), true)
			if cfg != nil {
				config := cfg.(*latest.SkaffoldConfig)
				err := defaults.Set(config)

				t.CheckNoError(err)
			}

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, cfg)
		})
	}
}

func config(ops ...func(*latest.SkaffoldConfig)) *latest.SkaffoldConfig {
	cfg := &latest.SkaffoldConfig{APIVersion: latest.Version, Kind: "Config"}
	for _, op := range ops {
		op(cfg)
	}
	return cfg
}

func withLocalBuild(ops ...func(*latest.BuildConfig)) func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		b := latest.BuildConfig{BuildType: latest.BuildType{LocalBuild: &latest.LocalBuild{}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withGoogleCloudBuild(id string, ops ...func(*latest.BuildConfig)) func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		b := latest.BuildConfig{BuildType: latest.BuildType{GoogleCloudBuild: &latest.GoogleCloudBuild{
			ProjectID:   id,
			DockerImage: "gcr.io/cloud-builders/docker",
			MavenImage:  "gcr.io/cloud-builders/mvn",
			GradleImage: "gcr.io/cloud-builders/gradle",
			KanikoImage: "gcr.io/kaniko-project/executor",
		}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withClusterBuild(secretName, mountPath, namespace, secret string, timeout string, ops ...func(*latest.BuildConfig)) func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		b := latest.BuildConfig{BuildType: latest.BuildType{Cluster: &latest.ClusterDetails{
			PullSecretName:      secretName,
			Namespace:           namespace,
			PullSecret:          secret,
			PullSecretMountPath: mountPath,
			Timeout:             timeout,
		}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withDockerConfig(secretName string, path string) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) {
		cfg.Cluster.DockerConfig = &latest.DockerConfig{
			SecretName: secretName,
			Path:       path,
		}
	}
}

func withKubectlDeploy(manifests ...string) func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		cfg.Deploy.DeployType.KubectlDeploy = &latest.KubectlDeploy{
			Manifests: manifests,
		}
	}
}

func withKubeContext(kubeContext string) func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		cfg.Deploy = latest.DeployConfig{
			KubeContext: kubeContext,
		}
	}
}

func withHelmDeploy() func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		cfg.Deploy.DeployType.HelmDeploy = &latest.HelmDeploy{}
	}
}

func withDockerArtifact(image, workspace, dockerfile string) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) {
		cfg.Artifacts = append(cfg.Artifacts, &latest.Artifact{
			ImageName: image,
			Workspace: workspace,
			ArtifactType: latest.ArtifactType{
				DockerArtifact: &latest.DockerArtifact{
					DockerfilePath: dockerfile,
				},
			},
		})
	}
}

func withBazelArtifact(image, workspace, target string) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) {
		cfg.Artifacts = append(cfg.Artifacts, &latest.Artifact{
			ImageName: image,
			Workspace: workspace,
			ArtifactType: latest.ArtifactType{
				BazelArtifact: &latest.BazelArtifact{
					BuildTarget: target,
				},
			},
		})
	}
}

func withKanikoArtifact(image, workspace, dockerfile, bucket string) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) {
		bc := &latest.KanikoBuildContext{}
		if bucket == "" {
			bc.LocalDir = &latest.LocalDir{
				InitImage: constants.DefaultBusyboxImage,
			}
		} else {
			bc.GCSBucket = bucket
		}

		cfg.Artifacts = append(cfg.Artifacts, &latest.Artifact{
			ImageName: image,
			Workspace: workspace,
			ArtifactType: latest.ArtifactType{
				KanikoArtifact: &latest.KanikoArtifact{
					DockerfilePath: dockerfile,
					BuildContext:   bc,
					Image:          constants.DefaultKanikoImage,
				},
			},
		})
	}
}

func withTagPolicy(tagPolicy latest.TagPolicy) func(*latest.BuildConfig) {
	return func(cfg *latest.BuildConfig) { cfg.TagPolicy = tagPolicy }
}

func withGitTagger() func(*latest.BuildConfig) {
	return withTagPolicy(latest.TagPolicy{GitTagger: &latest.GitTagger{}})
}

func withShaTagger() func(*latest.BuildConfig) {
	return withTagPolicy(latest.TagPolicy{ShaTagger: &latest.ShaTagger{}})
}

func withProfiles(profiles ...latest.Profile) func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		cfg.Profiles = profiles
	}
}

func withTests(testCases ...*latest.TestCase) func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		cfg.Test = testCases
	}
}

func withPortForward(portForward ...*latest.PortForwardResource) func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		cfg.PortForward = portForward
	}
}

func withStatusCheckDeadline(deadline int) func(*latest.SkaffoldConfig) {
	return func(cfg *latest.SkaffoldConfig) {
		cfg.Deploy.StatusCheckDeadlineSeconds = deadline
	}
}

func TestUpgradeToNextVersion(t *testing.T) {
	for i, schemaVersion := range SchemaVersions[0 : len(SchemaVersions)-2] {
		from := schemaVersion
		to := SchemaVersions[i+1]
		description := fmt.Sprintf("Upgrade from %s to %s", from.APIVersion, to.APIVersion)

		testutil.Run(t, description, func(t *testutil.T) {
			factory, _ := SchemaVersions.Find(from.APIVersion)

			newer, err := factory().Upgrade()

			t.CheckNoError(err)
			t.CheckDeepEqual(to.APIVersion, newer.GetVersion())
		})
	}
}

func TestCantUpgradeFromLatestVersion(t *testing.T) {
	factory, present := SchemaVersions.Find(latest.Version)
	testutil.CheckDeepEqual(t, true, present)

	_, err := factory().Upgrade()
	testutil.CheckError(t, true, err)
}
