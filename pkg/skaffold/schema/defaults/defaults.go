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

	"github.com/google/uuid"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

const (
	defaultCloudBuildDockerImage = "gcr.io/cloud-builders/docker"
	defaultCloudBuildMavenImage  = "gcr.io/cloud-builders/mvn"
	defaultCloudBuildGradleImage = "gcr.io/cloud-builders/gradle"
	defaultCloudBuildKanikoImage = constants.DefaultKanikoImage
	defaultCloudBuildPackImage   = "gcr.io/k8s-skaffold/pack"
)

// Set makes sure default values are set on a SkaffoldConfig.
func Set(c *latest.SkaffoldConfig) error {
	defaultToLocalBuild(c)
	defaultToKubectlDeploy(c)
	setDefaultTagger(c)
	setDefaultKustomizePath(c)
	setDefaultKubectlManifests(c)
	setDefaultLogsConfig(c)

	for _, a := range c.Build.Artifacts {
		setDefaultWorkspace(a)
		setDefaultSync(a)

		if c.Build.Cluster != nil && a.CustomArtifact == nil && a.BuildpackArtifact == nil {
			defaultToKanikoArtifact(a)
		} else {
			defaultToDockerArtifact(a)
		}

		switch {
		case a.DockerArtifact != nil:
			setDockerArtifactDefaults(a.DockerArtifact)

		case a.KanikoArtifact != nil:
			setKanikoArtifactDefaults(a.KanikoArtifact)

		case a.CustomArtifact != nil:
			setCustomArtifactDefaults(a.CustomArtifact)

		case a.BuildpackArtifact != nil:
			setBuildpackArtifactDefaults(a.BuildpackArtifact)
		}
	}

	withLocalBuild(c,
		setDefaultConcurrency,
	)

	withCloudBuildConfig(c,
		setDefaultCloudBuildDockerImage,
		setDefaultCloudBuildMavenImage,
		setDefaultCloudBuildGradleImage,
		setDefaultCloudBuildKanikoImage,
		setDefaultCloudBuildPackImage,
	)

	if err := withClusterConfig(c,
		setDefaultClusterNamespace,
		setDefaultClusterTimeout,
		setDefaultClusterPullSecret,
		setDefaultClusterDockerConfigSecret,
	); err != nil {
		return err
	}

	for _, pf := range c.PortForward {
		setDefaultPortForwardNamespace(pf)
		setDefaultLocalPort(pf)
		setDefaultAddress(pf)
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

func withLocalBuild(c *latest.SkaffoldConfig, operations ...func(*latest.LocalBuild)) {
	if local := c.Build.LocalBuild; local != nil {
		for _, operation := range operations {
			operation(local)
		}
	}
}

func setDefaultConcurrency(local *latest.LocalBuild) {
	if local.Concurrency == nil {
		local.Concurrency = &constants.DefaultLocalConcurrency
	}
}

func withCloudBuildConfig(c *latest.SkaffoldConfig, operations ...func(*latest.GoogleCloudBuild)) {
	if gcb := c.Build.GoogleCloudBuild; gcb != nil {
		for _, operation := range operations {
			operation(gcb)
		}
	}
}

func setDefaultCloudBuildDockerImage(gcb *latest.GoogleCloudBuild) {
	gcb.DockerImage = valueOrDefault(gcb.DockerImage, defaultCloudBuildDockerImage)
}

func setDefaultCloudBuildMavenImage(gcb *latest.GoogleCloudBuild) {
	gcb.MavenImage = valueOrDefault(gcb.MavenImage, defaultCloudBuildMavenImage)
}

func setDefaultCloudBuildGradleImage(gcb *latest.GoogleCloudBuild) {
	gcb.GradleImage = valueOrDefault(gcb.GradleImage, defaultCloudBuildGradleImage)
}

func setDefaultCloudBuildKanikoImage(gcb *latest.GoogleCloudBuild) {
	gcb.KanikoImage = valueOrDefault(gcb.KanikoImage, defaultCloudBuildKanikoImage)
}

func setDefaultCloudBuildPackImage(gcb *latest.GoogleCloudBuild) {
	gcb.PackImage = valueOrDefault(gcb.PackImage, defaultCloudBuildPackImage)
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
	if len(kustomize.KustomizePaths) == 0 {
		kustomize.KustomizePaths = []string{constants.DefaultKustomizationPath}
	}
}

func setDefaultKubectlManifests(c *latest.SkaffoldConfig) {
	if c.Deploy.KubectlDeploy != nil && len(c.Deploy.KubectlDeploy.Manifests) == 0 {
		c.Deploy.KubectlDeploy.Manifests = constants.DefaultKubectlManifests
	}
}

func setDefaultLogsConfig(c *latest.SkaffoldConfig) {
	if c.Deploy.Logs.Prefix == "" {
		c.Deploy.Logs.Prefix = "container"
	}
}

func defaultToDockerArtifact(a *latest.Artifact) {
	if a.ArtifactType == (latest.ArtifactType{}) {
		a.ArtifactType = latest.ArtifactType{
			DockerArtifact: &latest.DockerArtifact{},
		}
	}
}

func setCustomArtifactDefaults(a *latest.CustomArtifact) {
	if a.Dependencies == nil {
		a.Dependencies = &latest.CustomDependencies{
			Paths: []string{"."},
		}
	}
}

func setBuildpackArtifactDefaults(a *latest.BuildpackArtifact) {
	if a.ProjectDescriptor == "" {
		a.ProjectDescriptor = constants.DefaultProjectDescriptor
	}
	if a.Dependencies == nil {
		a.Dependencies = &latest.BuildpackDependencies{
			Paths: []string{"."},
		}
	}
}

func setDockerArtifactDefaults(a *latest.DockerArtifact) {
	a.DockerfilePath = valueOrDefault(a.DockerfilePath, constants.DefaultDockerfilePath)
}

func setDefaultWorkspace(a *latest.Artifact) {
	a.Workspace = valueOrDefault(a.Workspace, ".")
}

func setDefaultSync(a *latest.Artifact) {
	if a.Sync != nil {
		if len(a.Sync.Manual) == 0 && len(a.Sync.Infer) == 0 && a.Sync.Auto == nil {
			switch {
			case a.JibArtifact != nil || a.BuildpackArtifact != nil:
				a.Sync.Auto = &latest.Auto{}
			default:
				a.Sync.Infer = []string{"**/*"}
			}
		}
	}
}

func withClusterConfig(c *latest.SkaffoldConfig, opts ...func(*latest.ClusterDetails) error) error {
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
			return fmt.Errorf("getting current namespace: %w", err)
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
	cluster.PullSecretMountPath = valueOrDefault(cluster.PullSecretMountPath, constants.DefaultKanikoSecretMountPath)
	if cluster.PullSecretPath != "" {
		absPath, err := homedir.Expand(cluster.PullSecretPath)
		if err != nil {
			return fmt.Errorf("unable to expand pullSecretPath %s", cluster.PullSecretPath)
		}
		cluster.PullSecretPath = absPath
		random := ""
		if cluster.RandomPullSecret {
			uid, _ := uuid.NewUUID()
			random = uid.String()
		}
		cluster.PullSecretName = valueOrDefault(cluster.PullSecretName, constants.DefaultKanikoSecretName+random)
		return nil
	}
	return nil
}

func setDefaultClusterDockerConfigSecret(cluster *latest.ClusterDetails) error {
	if cluster.DockerConfig == nil {
		return nil
	}

	random := ""
	if cluster.RandomDockerConfigSecret {
		uid, _ := uuid.NewUUID()
		random = uid.String()
	}

	cluster.DockerConfig.SecretName = valueOrDefault(cluster.DockerConfig.SecretName, constants.DefaultKanikoDockerConfigSecretName+random)

	if cluster.DockerConfig.Path == "" {
		return nil
	}

	absPath, err := homedir.Expand(cluster.DockerConfig.Path)
	if err != nil {
		return fmt.Errorf("unable to expand dockerConfig.path %s", cluster.DockerConfig.Path)
	}

	cluster.DockerConfig.Path = absPath
	return nil
}

func defaultToKanikoArtifact(artifact *latest.Artifact) {
	if artifact.KanikoArtifact == nil {
		artifact.KanikoArtifact = &latest.KanikoArtifact{}
	}
}

func setKanikoArtifactDefaults(a *latest.KanikoArtifact) {
	a.Image = valueOrDefault(a.Image, constants.DefaultKanikoImage)
	a.DockerfilePath = valueOrDefault(a.DockerfilePath, constants.DefaultDockerfilePath)
	a.InitImage = valueOrDefault(a.InitImage, constants.DefaultBusyboxImage)
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

func setDefaultLocalPort(pf *latest.PortForwardResource) {
	if pf.LocalPort == 0 {
		pf.LocalPort = pf.Port
	}
}

func setDefaultPortForwardNamespace(pf *latest.PortForwardResource) {
	if pf.Namespace == "" {
		ns, err := currentNamespace()
		if err != nil {
			pf.Namespace = constants.DefaultPortForwardNamespace
			return
		}
		pf.Namespace = ns
	}
}

func setDefaultAddress(pf *latest.PortForwardResource) {
	if pf.Address == "" {
		pf.Address = constants.DefaultPortForwardAddress
	}
}
