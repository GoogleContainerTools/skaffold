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

package defaults

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Set makes sure default values are set on a SkaffoldConfig.
func Set(c *latest.SkaffoldConfig) error {

	defaultToLocalBuild(c)
	defaultToKubectlDeploy(c)
	setDefaultTagger(c)
	setDefaultKustomizePath(c)
	setDefaultKubectlManifests(c)

	withCloudBuildConfig(c,
		SetDefaultCloudBuildDockerImage,
		setDefaultCloudBuildMavenImage,
		setDefaultCloudBuildGradleImage,
	)

	if c.Build.Cluster != nil {
		// All artifacts should be built with kaniko
		for _, a := range c.Build.Artifacts {
			setDefaultKanikoArtifact(a)
			setDefaultKanikoArtifactImage(a)
			setDefaultKanikoArtifactBuildContext(a)
			setDefaultKanikoDockerfilePath(a)
		}
	}

	if err := withClusterConfig(c,
		setDefaultClusterNamespace,
		setDefaultClusterTimeout,
		setDefaultClusterPullSecret,
		setDefaultClusterDockerConfigSecret,
	); err != nil {
		return err
	}

	for _, a := range c.Build.Artifacts {
		setDefaultWorkspace(a)
		defaultToDockerArtifact(a)
		setDefaultDockerfile(a)
	}

	return nil
}

func defaultToLocalBuild(c *latest.SkaffoldConfig) {
	if c.Build.BuildType != (latest.BuildType{}) {
		return
	}

	logrus.Debugf("Defaulting build type to local build")
	c.Build.BuildType.LocalBuild = &latest.LocalBuild{}
}

func defaultToKubectlDeploy(c *latest.SkaffoldConfig) {
	if c.Deploy.DeployType != (latest.DeployType{}) {
		return
	}

	logrus.Debugf("Defaulting deploy type to kubectl")
	c.Deploy.DeployType.KubectlDeploy = &latest.KubectlDeploy{}
}

func withCloudBuildConfig(c *latest.SkaffoldConfig, operations ...func(kaniko *latest.GoogleCloudBuild)) {
	if gcb := c.Build.GoogleCloudBuild; gcb != nil {
		for _, operation := range operations {
			operation(gcb)
		}
	}
}

// SetDefaultCloudBuildDockerImage sets the default cloud build image if it doesn't exist
func SetDefaultCloudBuildDockerImage(gcb *latest.GoogleCloudBuild) {
	gcb.DockerImage = valueOrDefault(gcb.DockerImage, constants.DefaultCloudBuildDockerImage)
}

func setDefaultCloudBuildMavenImage(gcb *latest.GoogleCloudBuild) {
	gcb.MavenImage = valueOrDefault(gcb.MavenImage, constants.DefaultCloudBuildMavenImage)
}

func setDefaultCloudBuildGradleImage(gcb *latest.GoogleCloudBuild) {
	gcb.GradleImage = valueOrDefault(gcb.GradleImage, constants.DefaultCloudBuildGradleImage)
}

func setDefaultTagger(c *latest.SkaffoldConfig) {
	if c.Build.TagPolicy != (latest.TagPolicy{}) {
		return
	}

	c.Build.TagPolicy = latest.TagPolicy{GitTagger: &latest.GitTagger{}}
}

func setDefaultKustomizePath(c *latest.SkaffoldConfig) {
	kustomize := c.Deploy.KustomizeDeploy
	if kustomize == nil {
		return
	}

	kustomize.KustomizePath = valueOrDefault(kustomize.KustomizePath, constants.DefaultKustomizationPath)
}

func setDefaultKubectlManifests(c *latest.SkaffoldConfig) {
	if c.Deploy.KubectlDeploy != nil && len(c.Deploy.KubectlDeploy.Manifests) == 0 {
		c.Deploy.KubectlDeploy.Manifests = constants.DefaultKubectlManifests
	}
}

func defaultToDockerArtifact(a *latest.Artifact) {
	if a.ArtifactType == (latest.ArtifactType{}) {
		a.ArtifactType = latest.ArtifactType{
			DockerArtifact: &latest.DockerArtifact{},
		}
	}
}

func setDefaultDockerfile(a *latest.Artifact) {
	if a.DockerArtifact != nil {
		SetDefaultDockerArtifact(a.DockerArtifact)
	}
}

// SetDefaultDockerArtifact sets defaults on docker artifacts
func SetDefaultDockerArtifact(a *latest.DockerArtifact) {
	a.DockerfilePath = valueOrDefault(a.DockerfilePath, constants.DefaultDockerfilePath)
}

func setDefaultWorkspace(a *latest.Artifact) {
	a.Workspace = valueOrDefault(a.Workspace, ".")
}

func withClusterConfig(c *latest.SkaffoldConfig, opts ...func(cluster *latest.ClusterDetails) error) error {
	clusterDetails := c.Build.BuildType.Cluster
	if clusterDetails == nil {
		return nil
	}
	for _, o := range opts {
		if err := o(clusterDetails); err != nil {
			return err
		}
	}
	return nil
}

func setDefaultClusterNamespace(cluster *latest.ClusterDetails) error {
	if cluster.Namespace == "" {
		ns, err := currentNamespace()
		if err != nil {
			return errors.Wrap(err, "getting current namespace")
		}
		cluster.Namespace = ns
	}
	return nil
}

func setDefaultClusterTimeout(cluster *latest.ClusterDetails) error {
	cluster.Timeout = valueOrDefault(cluster.Timeout, constants.DefaultKanikoTimeout)
	return nil
}

func setDefaultClusterPullSecret(cluster *latest.ClusterDetails) error {
	cluster.PullSecretName = valueOrDefault(cluster.PullSecretName, constants.DefaultKanikoSecretName)
	if cluster.PullSecret != "" {
		absPath, err := homedir.Expand(cluster.PullSecret)
		if err != nil {
			return fmt.Errorf("unable to expand pullSecret %s", cluster.PullSecret)
		}
		cluster.PullSecret = absPath
		return nil
	}
	return nil
}

func setDefaultClusterDockerConfigSecret(cluster *latest.ClusterDetails) error {
	if cluster.DockerConfig == nil {
		return nil
	}

	cluster.DockerConfig.SecretName = valueOrDefault(cluster.DockerConfig.SecretName, constants.DefaultKanikoDockerConfigSecretName)

	if cluster.DockerConfig.Path != "" {
		absPath, err := homedir.Expand(cluster.DockerConfig.Path)
		if err != nil {
			return fmt.Errorf("unable to expand dockerConfig.path %s", cluster.DockerConfig.Path)
		}

		cluster.DockerConfig.Path = absPath
		return nil
	}

	return nil
}

func setDefaultKanikoArtifact(artifact *latest.Artifact) {
	if artifact.KanikoArtifact == nil {
		artifact.KanikoArtifact = &latest.KanikoArtifact{}
	}
}

func setDefaultKanikoDockerfilePath(artifact *latest.Artifact) {
	artifact.KanikoArtifact.DockerfilePath = valueOrDefault(artifact.KanikoArtifact.DockerfilePath, constants.DefaultDockerfilePath)
}

func setDefaultKanikoArtifactBuildContext(artifact *latest.Artifact) {
	if artifact.KanikoArtifact.BuildContext == nil {
		artifact.KanikoArtifact.BuildContext = &latest.KanikoBuildContext{
			LocalDir: &latest.LocalDir{},
		}
	}
	localDir := artifact.KanikoArtifact.BuildContext.LocalDir
	if localDir != nil {
		localDir.InitImage = valueOrDefault(localDir.InitImage, constants.DefaultBusyboxImage)
	}
}

func setDefaultKanikoArtifactImage(artifact *latest.Artifact) {
	kanikoArtifact := artifact.KanikoArtifact
	artifact.KanikoArtifact.Image = valueOrDefault(kanikoArtifact.Image, constants.DefaultKanikoImage)
}

func valueOrDefault(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

func currentNamespace() (string, error) {
	cfg, err := kubectx.CurrentConfig()
	if err != nil {
		return "", err
	}

	current, present := cfg.Contexts[cfg.CurrentContext]
	if present {
		if current.Namespace != "" {
			return current.Namespace, nil
		}
	}

	return "default", nil
}
