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
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/defaults"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2alpha1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta14"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v2beta8"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v3alpha1"
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
	minimalClusterConfig = `
build:
  artifacts:
  - image: image1
    context: ./examples/app1
    kaniko: {}
  cluster: {}
`
	kanikoConfigMap = `
build:
  artifacts:
  - image: image1
    context: ./examples/app1
    kaniko:
      volumeMounts:
      - name: docker-config
        mountPath: /kaniko/.docker
  cluster:
    pullSecretName: "some-secret"
    volumes:
    - name: docker-config
      configMap:
        name: docker-config
`

	completeClusterConfig = `
build:
  artifacts:
  - image: image1
    context: ./examples/app1
    kaniko: {}
  cluster:
    pullSecretPath: /secret.json
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

	customLogPrefix = `
deploy:
  logs:
    prefix: none
`
)

func TestIsSkaffoldConfig(t *testing.T) {
	tests := []struct {
		description string
		contents    string
		isValid     bool
	}{
		{
			description: "valid skaffold config",
			contents: `apiVersion: skaffold/v1beta6
kind: Config
deploy:
  kustomize: {}`,
			isValid: true,
		},
		{
			description: "not a valid format",
			contents:    "test",
			isValid:     false,
		},
		{
			description: "invalid skaffold config version",
			contents: `apiVersion: skaffold/v1beta100
kind: Config
deploy:
  kustomize: {}`,
			isValid: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("skaffold.yaml", test.contents)

			isValid := IsSkaffoldConfig(tmpDir.Path("skaffold.yaml"))

			t.CheckDeepEqual(test.isValid, isValid)
		})
	}
}

func TestParseConfigAndUpgrade(t *testing.T) {
	tests := []struct {
		apiVersion  []string
		description string
		config      []string
		expected    []util.VersionedConfig
		shouldErr   bool
	}{
		{
			apiVersion:  []string{latestV1.Version},
			description: "Kaniko Volume Mount - ConfigMap",
			config:      []string{kanikoConfigMap},
			expected: []util.VersionedConfig{config(
				withClusterBuild("some-secret", "/secret", "default", "", "20m",
					withGitTagger(),
					withKanikoArtifact(),
					withKanikoVolumeMount("docker-config", "/kaniko/.docker"),
					withVolume(v1.Volume{
						Name: "docker-config",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "docker-config",
								},
							},
						},
					})),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "Minimal config",
			config:      []string{minimalConfig},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withGitTagger(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{"skaffold/v1alpha1"},
			description: "Old minimal config",
			config:      []string{minimalConfig},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withGitTagger(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "Simple config",
			config:      []string{simpleConfig},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withGitTagger(),
					withDockerArtifact("example", ".", "Dockerfile"),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "Complete config",
			config:      []string{completeConfig},
			expected: []util.VersionedConfig{config(
				withGoogleCloudBuild("ID",
					withShaTagger(),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
					withBazelArtifact(),
				),
				withKubectlDeploy("dep.yaml", "svc.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{v2beta8.Version},
			description: "Old version complete config",
			config:      []string{completeConfig},
			expected: []util.VersionedConfig{config(
				withGoogleCloudBuild("ID",
					withShaTagger(),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
					withBazelArtifact(),
				),
				withKubectlDeploy("dep.yaml", "svc.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latestV1.Version, latestV1.Version},
			description: "Multiple complete config with same API versions",
			config:      []string{completeConfig, completeClusterConfig},
			expected: []util.VersionedConfig{config(
				withGoogleCloudBuild("ID",
					withShaTagger(),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
					withBazelArtifact(),
				),
				withKubectlDeploy("dep.yaml", "svc.yaml"),
				withLogsPrefix("container"),
			), config(
				withClusterBuild("secret-name", "/secret", "nskaniko", "/secret.json", "120m",
					withGitTagger(),
					withDockerConfig("config-name", "/kaniko/.docker"),
					withKanikoArtifact(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latestV1.Version, v2beta8.Version},
			description: "Multiple complete config with different API versions",
			config:      []string{completeConfig, completeClusterConfig},
			expected: []util.VersionedConfig{config(
				withGoogleCloudBuild("ID",
					withShaTagger(),
					withDockerArtifact("image1", "./examples/app1", "Dockerfile.dev"),
					withBazelArtifact(),
				),
				withKubectlDeploy("dep.yaml", "svc.yaml"),
				withLogsPrefix("container"),
			), config(
				withClusterBuild("secret-name", "/secret", "nskaniko", "/secret.json", "120m",
					withGitTagger(),
					withDockerConfig("config-name", "/kaniko/.docker"),
					withKanikoArtifact(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "Minimal Cluster config",
			config:      []string{minimalClusterConfig},
			expected: []util.VersionedConfig{config(
				withClusterBuild("", "/secret", "default", "", "20m",
					withGitTagger(),
					withKanikoArtifact(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "Complete Cluster config",
			config:      []string{completeClusterConfig},
			expected: []util.VersionedConfig{config(
				withClusterBuild("secret-name", "/secret", "nskaniko", "/secret.json", "120m",
					withGitTagger(),
					withDockerConfig("config-name", "/kaniko/.docker"),
					withKanikoArtifact(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "Bad config",
			config:      []string{badConfig},
			shouldErr:   true,
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "two taggers defined",
			config:      []string{invalidConfig},
			shouldErr:   true,
		},
		{
			apiVersion:  []string{""},
			description: "ApiVersion not specified",
			config:      []string{minimalConfig},
			shouldErr:   true,
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "invalid statusCheckDeadline",
			config:      []string{invalidStatusCheckConfig},
			shouldErr:   true,
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "valid statusCheckDeadline",
			config:      []string{validStatusCheckConfig},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withGitTagger(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withStatusCheckDeadline(10),
				withLogsPrefix("container"),
			)},
		},
		{
			apiVersion:  []string{latestV1.Version},
			description: "custom log prefix",
			config:      []string{customLogPrefix},
			expected: []util.VersionedConfig{config(
				withLocalBuild(
					withGitTagger(),
				),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("none"),
			)},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			tmpDir := t.NewTempDir().
				Write("skaffold.yaml", format(t, test.config, test.apiVersion))

			cfgs, err := ParseConfigAndUpgrade(tmpDir.Path("skaffold.yaml"))
			for _, cfg := range cfgs {
				if _, ok := SchemaVersionsV2.Find(test.apiVersion[0]); !ok {
					// TODO(nkubala): the "defaults" package below only accept latestV2 schema.
					t.SkipNow()
				}

				err := defaults.Set(cfg.(*latestV2.SkaffoldConfig))
				t.CheckNoError(err)
			}

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, cfgs)
		})
	}
}

func TestMarshalConfig(t *testing.T) {
	tests := []struct {
		description string
		config      *latestV2.SkaffoldConfig
		shouldErr   bool
	}{
		{
			description: "Kaniko Volume Mount - ConfigMap",
			config: config(
				withClusterBuild("some-secret", "/some/secret", "default", "", "20m",
					withGitTagger(),
					withKanikoArtifact(),
					withKanikoVolumeMount("docker-config", "/kaniko/.docker"),
					withVolume(v1.Volume{
						Name: "docker-config",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "docker-config",
								},
							},
						},
					})),
				withKubectlDeploy("k8s/*.yaml"),
				withLogsPrefix("container"),
			),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.SetupFakeKubernetesContext(api.Config{CurrentContext: "cluster1"})

			actual, err := yaml.Marshal(test.config)
			t.CheckNoError(err)

			// Unmarshal the YAML and make sure it equals the original.
			// We can't compare the strings because the YAML serializer isn't deterministic.
			// TestParseConfigAndUpgrade verifies that YAML -> Go works correctly.
			// This test verifies Go -> YAML -> Go returns the original structure. Since we know
			// YAML -> Go is working this ensures Go -> YAML is correct.
			recovered := &latestV2.SkaffoldConfig{}

			err = yaml.Unmarshal(actual, recovered)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.config, recovered)
		})
	}
}

func config(ops ...func(*latestV2.SkaffoldConfig)) *latestV2.SkaffoldConfig {
	cfg := &latestV2.SkaffoldConfig{APIVersion: latestV2.Version, Kind: "Config"}
	for _, op := range ops {
		op(cfg)
	}
	return cfg
}

func format(t *testutil.T, configs []string, apiVersions []string) string {
	var str []string
	t.CheckDeepEqual(len(configs), len(apiVersions))
	for i := range configs {
		str = append(str, fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", apiVersions[i], configs[i]))
	}
	return strings.Join(str, "\n---\n")
}

func withLocalBuild(ops ...func(*latestV2.BuildConfig)) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		b := latestV2.BuildConfig{BuildType: latestV2.BuildType{LocalBuild: &latestV2.LocalBuild{}}}
		for _, op := range ops {
			op(&b)
		}
		if len(b.Artifacts) > 0 {
			b.LocalBuild.Concurrency = &constants.DefaultLocalConcurrency
		}
		cfg.Build = b
	}
}

func withGoogleCloudBuild(id string, ops ...func(*latestV2.BuildConfig)) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		b := latestV2.BuildConfig{BuildType: latestV2.BuildType{GoogleCloudBuild: &latestV2.GoogleCloudBuild{
			ProjectID:   id,
			DockerImage: "gcr.io/cloud-builders/docker",
			MavenImage:  "gcr.io/cloud-builders/mvn",
			GradleImage: "gcr.io/cloud-builders/gradle",
			KanikoImage: kaniko.DefaultImage,
			PackImage:   "gcr.io/k8s-skaffold/pack",
		}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withClusterBuild(secretName, mountPath, namespace, secret string, timeout string, ops ...func(*latestV2.BuildConfig)) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		b := latestV2.BuildConfig{BuildType: latestV2.BuildType{Cluster: &latestV2.ClusterDetails{
			PullSecretName:      secretName,
			Namespace:           namespace,
			PullSecretPath:      secret,
			PullSecretMountPath: mountPath,
			Timeout:             timeout,
		}}}
		for _, op := range ops {
			op(&b)
		}
		cfg.Build = b
	}
}

func withDockerConfig(secretName string, path string) func(*latestV2.BuildConfig) {
	return func(cfg *latestV2.BuildConfig) {
		cfg.Cluster.DockerConfig = &latestV2.DockerConfig{
			SecretName: secretName,
			Path:       path,
		}
	}
}

func withKubectlDeploy(manifests ...string) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		cfg.Deploy.DeployType.KubectlDeploy = &latestV2.KubectlDeploy{
			Manifests: manifests,
		}
	}
}

func withKubeContext(kubeContext string) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		cfg.Deploy = latestV2.DeployConfig{
			KubeContext: kubeContext,
		}
	}
}

func withHelmDeploy() func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		cfg.Deploy.DeployType.HelmDeploy = &latestV2.HelmDeploy{}
	}
}

func withDockerArtifact(image, workspace, dockerfile string) func(*latestV2.BuildConfig) {
	return func(cfg *latestV2.BuildConfig) {
		cfg.Artifacts = append(cfg.Artifacts, &latestV2.Artifact{
			ImageName: image,
			Workspace: workspace,
			ArtifactType: latestV2.ArtifactType{
				DockerArtifact: &latestV2.DockerArtifact{
					DockerfilePath: dockerfile,
				},
			},
		})
	}
}

func withBazelArtifact() func(*latestV2.BuildConfig) {
	return func(cfg *latestV2.BuildConfig) {
		cfg.Artifacts = append(cfg.Artifacts, &latestV2.Artifact{
			ImageName: "image2",
			Workspace: "./examples/app2",
			ArtifactType: latestV2.ArtifactType{
				BazelArtifact: &latestV2.BazelArtifact{
					BuildTarget: "//:example.tar",
				},
			},
		})
	}
}

func withKanikoArtifact() func(*latestV2.BuildConfig) {
	return func(cfg *latestV2.BuildConfig) {
		cfg.Artifacts = append(cfg.Artifacts, &latestV2.Artifact{
			ImageName: "image1",
			Workspace: "./examples/app1",
			ArtifactType: latestV2.ArtifactType{
				KanikoArtifact: &latestV2.KanikoArtifact{
					DockerfilePath: "Dockerfile",
					InitImage:      constants.DefaultBusyboxImage,
					Image:          kaniko.DefaultImage,
				},
			},
		})
	}
}

// withKanikoVolumeMount appends a volume mount to the latest Kaniko artifact
func withKanikoVolumeMount(name, mountPath string) func(*latestV2.BuildConfig) {
	return func(cfg *latestV2.BuildConfig) {
		if cfg.Artifacts[len(cfg.Artifacts)-1].KanikoArtifact.VolumeMounts == nil {
			cfg.Artifacts[len(cfg.Artifacts)-1].KanikoArtifact.VolumeMounts = []v1.VolumeMount{}
		}

		cfg.Artifacts[len(cfg.Artifacts)-1].KanikoArtifact.VolumeMounts = append(
			cfg.Artifacts[len(cfg.Artifacts)-1].KanikoArtifact.VolumeMounts,
			v1.VolumeMount{
				Name:      name,
				MountPath: mountPath,
			},
		)
	}
}

// withVolume appends a volume to the cluster
func withVolume(v v1.Volume) func(*latestV2.BuildConfig) {
	return func(cfg *latestV2.BuildConfig) {
		cfg.Cluster.Volumes = append(cfg.Cluster.Volumes, v)
	}
}

func withTagPolicy(tagPolicy latestV2.TagPolicy) func(*latestV2.BuildConfig) {
	return func(cfg *latestV2.BuildConfig) { cfg.TagPolicy = tagPolicy }
}

func withGitTagger() func(*latestV2.BuildConfig) {
	return withTagPolicy(latestV2.TagPolicy{GitTagger: &latestV2.GitTagger{}})
}

func withShaTagger() func(*latestV2.BuildConfig) {
	return withTagPolicy(latestV2.TagPolicy{ShaTagger: &latestV2.ShaTagger{}})
}

func withProfiles(profiles ...latestV2.Profile) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		cfg.Profiles = profiles
	}
}

func withTests(testCases ...*latestV2.TestCase) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		cfg.Test = testCases
	}
}

func withPortForward(portForward ...*latestV2.PortForwardResource) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		cfg.PortForward = portForward
	}
}

func withStatusCheckDeadline(deadline int) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		cfg.Deploy.StatusCheckDeadlineSeconds = deadline
	}
}

func withLogsPrefix(prefix string) func(*latestV2.SkaffoldConfig) {
	return func(cfg *latestV2.SkaffoldConfig) {
		cfg.Deploy.Logs.Prefix = prefix
	}
}

func TestUpgradeToNextVersion(t *testing.T) {
	for _, versions := range []Versions{SchemaVersions} {
		for i, schemaVersion := range versions[0 : len(versions)-2] {
			// TODO(nkubala)[11/12/21]: Upgrade from v2 to v3 config not supported yet
			if schemaVersion.APIVersion == LatestV1Version.APIVersion {
				t.SkipNow()
			}
			from := schemaVersion
			to := versions[i+1]
			description := fmt.Sprintf("Upgrade from %s to %s", from.APIVersion, to.APIVersion)

			testutil.Run(t, description, func(t *testutil.T) {
				factory, _ := versions.Find(from.APIVersion)

				newer, err := factory().Upgrade()

				t.CheckNoError(err)
				t.CheckDeepEqual(to.APIVersion, newer.GetVersion())
			})
		}
	}
}

func TestCantUpgradeFromLatestV1Version(t *testing.T) {
	factory, present := SchemaVersions.Find(latestV1.Version)
	testutil.CheckDeepEqual(t, true, present)

	_, err := factory().Upgrade()
	testutil.CheckError(t, true, err)
}

func TestCantUpgradeFromLatestV2Version(t *testing.T) {
	factory, present := SchemaVersions.Find(latestV2.Version)
	testutil.CheckDeepEqual(t, true, present)

	_, err := factory().Upgrade()
	testutil.CheckError(t, true, err)
}

func TestParseConfigAndUpgradeToOlderVersion(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.NewTempDir().
			Write("skaffold.yaml", fmt.Sprintf("apiVersion: %s\nkind: Config\n%s", latestV1.Version, minimalConfig)).
			Chdir()

		cfgs, err := ParseConfig("skaffold.yaml")
		t.CheckNoError(err)
		_, err = UpgradeTo(cfgs, "skaffold/v1alpha1")
		t.CheckErrorContains(`is more recent than target version "skaffold/v1alpha1": upgrade Skaffold`, err)
	})
}

func TestGetLatestFromCompatibilityCheck(t *testing.T) {
	tests := []struct {
		description string
		apiVersions []util.VersionedConfig
		expected    string
		shouldErr   bool
		err         error
	}{
		{
			apiVersions: []util.VersionedConfig{
				&v1alpha1.SkaffoldConfig{APIVersion: v1alpha1.Version},
				&v1beta1.SkaffoldConfig{APIVersion: v1beta1.Version},
				&v2alpha1.SkaffoldConfig{APIVersion: v2alpha1.Version},
				&v2beta1.SkaffoldConfig{APIVersion: v2beta1.Version},
			},
			description: "valid compatibility check for all v1 schemas releases",
			expected:    latestV1.Version,
			shouldErr:   false,
		},
		{

			apiVersions: []util.VersionedConfig{
				&v3alpha1.SkaffoldConfig{APIVersion: v3alpha1.Version},
			},
			description: "valid compatibility check for all v2 schemas releases",
			expected:    latestV2.Version,
			shouldErr:   false,
		},
		{
			apiVersions: []util.VersionedConfig{
				&v1alpha1.SkaffoldConfig{APIVersion: v1alpha1.Version},
				&v3alpha1.SkaffoldConfig{APIVersion: v3alpha1.Version},
			},
			description: "invalid compatibility among v1 and v2 versions",
			shouldErr:   true,
			err: fmt.Errorf("detected incompatible versions:%v are incompatible with %v",
				[]string{latestV1.Version, v1alpha1.Version}, []string{v3alpha1.Version}),
		},
		{
			apiVersions: []util.VersionedConfig{
				&v1alpha1.SkaffoldConfig{APIVersion: "vXalphaY"},
			},
			description: "invalid api version",
			shouldErr:   true,
			err:         fmt.Errorf("unknown apiVersion vXalpaY"),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			upToDateVersion, err := getLatestFromCompatibilityCheck(test.apiVersions)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, upToDateVersion)
		})
	}
}

func TestIsCompatibleWith(t *testing.T) {
	tests := []struct {
		description string
		apiVersions []util.VersionedConfig
		toVersion   string
		shouldErr   bool
		err         error
	}{
		{
			apiVersions: []util.VersionedConfig{
				&v1alpha1.SkaffoldConfig{APIVersion: v1alpha1.Version},
				&v1beta1.SkaffoldConfig{APIVersion: v1beta1.Version},
				&v2alpha1.SkaffoldConfig{APIVersion: v2alpha1.Version},
				&v2beta1.SkaffoldConfig{APIVersion: v2beta1.Version},
			},
			description: "v1 schemas are compatible to a v1 schema",
			toVersion:   v2beta14.Version,
			shouldErr:   false,
		},
		{
			apiVersions: []util.VersionedConfig{
				&v3alpha1.SkaffoldConfig{APIVersion: v3alpha1.Version},
			},
			description: "v2 schemas are compatible to a v2 schema",
			toVersion:   latestV2.Version,
			shouldErr:   false,
		},
		{
			apiVersions: []util.VersionedConfig{
				&v1alpha1.SkaffoldConfig{APIVersion: v1alpha1.Version},
				&v1beta1.SkaffoldConfig{APIVersion: v1beta1.Version},
			},
			description: "v1 schemas cannot upgrade to v2.",
			toVersion:   latestV2.Version,
			shouldErr:   true,
			err: fmt.Errorf("the following versions are incompatible with target version %v. upgrade aborted",
				[]string{v1alpha1.Version, v1beta1.Version}),
		},
		{
			apiVersions: []util.VersionedConfig{
				&v3alpha1.SkaffoldConfig{APIVersion: v3alpha1.Version},
				&latestV2.SkaffoldConfig{APIVersion: latestV2.Version},
			},
			description: "v2 schemas are incompatible with v1.",
			toVersion:   latestV1.Version,
			shouldErr:   true,
			err: fmt.Errorf("the following versions are incompatible with target version %v. upgrade aborted",
				[]string{v3alpha1.Version, latestV2.Version}),
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := IsCompatibleWith(test.apiVersions, test.toVersion)
			t.CheckError(test.shouldErr, err)
		})
	}
}
