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
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	schemautil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	defaultCloudBuildDockerImage = "gcr.io/cloud-builders/docker"
	defaultCloudBuildMavenImage  = "gcr.io/cloud-builders/mvn"
	defaultCloudBuildGradleImage = "gcr.io/cloud-builders/gradle"
	defaultCloudBuildKanikoImage = kaniko.DefaultImage
	defaultCloudBuildPackImage   = "gcr.io/k8s-skaffold/pack"
)

// Set makes sure default values are set on a SkaffoldConfig.
func Set(c *latestV1.SkaffoldConfig) error {
	defaultToLocalBuild(c)
	setDefaultTagger(c)
	setDefaultKustomizePath(c)
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

		for _, d := range a.Dependencies {
			setDefaultArtifactDependencyAlias(d)
		}
	}

	withLocalBuild(c, func(lb *latestV1.LocalBuild) {
		// don't set build concurrency if there are no artifacts in the current config
		if len(c.Build.Artifacts) > 0 {
			setDefaultConcurrency(lb)
		}
	})

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

	for i, pf := range c.PortForward {
		if pf == nil {
			return fmt.Errorf("portForward[%d] of config with name '%s' is empty, Please check if it has valid values", i, c.Metadata.Name)
		}
		setDefaultLocalPort(pf)
		setDefaultAddress(pf)
	}

	setDefaultTestWorkspace(c)
	return nil
}

// SetDefaultDeployer adds a default kubectl deploy configuration.
func SetDefaultDeployer(c *latestV1.SkaffoldConfig) {
	defaultToKubectlDeploy(c)
	setDefaultKubectlManifests(c)
}

func defaultToLocalBuild(c *latestV1.SkaffoldConfig) {
	if c.Build.BuildType != (latestV1.BuildType{}) {
		return
	}

	logrus.Debugf("Defaulting build type to local build")
	c.Build.BuildType.LocalBuild = &latestV1.LocalBuild{}
}

func defaultToKubectlDeploy(c *latestV1.SkaffoldConfig) {
	if c.Deploy.DeployType != (latestV1.DeployType{}) {
		return
	}

	logrus.Debugf("Defaulting deploy type to kubectl")
	c.Deploy.DeployType.KubectlDeploy = &latestV1.KubectlDeploy{}
}

func withLocalBuild(c *latestV1.SkaffoldConfig, operations ...func(*latestV1.LocalBuild)) {
	if local := c.Build.LocalBuild; local != nil {
		for _, operation := range operations {
			operation(local)
		}
	}
}

func setDefaultConcurrency(local *latestV1.LocalBuild) {
	if local.Concurrency == nil {
		local.Concurrency = &constants.DefaultLocalConcurrency
	}
}

func withCloudBuildConfig(c *latestV1.SkaffoldConfig, operations ...func(*latestV1.GoogleCloudBuild)) {
	if gcb := c.Build.GoogleCloudBuild; gcb != nil {
		for _, operation := range operations {
			operation(gcb)
		}
	}
}

func setDefaultCloudBuildDockerImage(gcb *latestV1.GoogleCloudBuild) {
	gcb.DockerImage = valueOrDefault(gcb.DockerImage, defaultCloudBuildDockerImage)
}

func setDefaultCloudBuildMavenImage(gcb *latestV1.GoogleCloudBuild) {
	gcb.MavenImage = valueOrDefault(gcb.MavenImage, defaultCloudBuildMavenImage)
}

func setDefaultCloudBuildGradleImage(gcb *latestV1.GoogleCloudBuild) {
	gcb.GradleImage = valueOrDefault(gcb.GradleImage, defaultCloudBuildGradleImage)
}

func setDefaultCloudBuildKanikoImage(gcb *latestV1.GoogleCloudBuild) {
	gcb.KanikoImage = valueOrDefault(gcb.KanikoImage, defaultCloudBuildKanikoImage)
}

func setDefaultCloudBuildPackImage(gcb *latestV1.GoogleCloudBuild) {
	gcb.PackImage = valueOrDefault(gcb.PackImage, defaultCloudBuildPackImage)
}

func setDefaultTagger(c *latestV1.SkaffoldConfig) {
	if c.Build.TagPolicy != (latestV1.TagPolicy{}) {
		return
	}

	c.Build.TagPolicy = latestV1.TagPolicy{GitTagger: &latestV1.GitTagger{}}
}

func setDefaultKustomizePath(c *latestV1.SkaffoldConfig) {
	kustomize := c.Deploy.KustomizeDeploy
	if kustomize == nil {
		return
	}
	if len(kustomize.KustomizePaths) == 0 {
		kustomize.KustomizePaths = []string{constants.DefaultKustomizationPath}
	}
}

func setDefaultKubectlManifests(c *latestV1.SkaffoldConfig) {
	if c.Deploy.KubectlDeploy != nil && len(c.Deploy.KubectlDeploy.Manifests) == 0 && len(c.Deploy.KubectlDeploy.RemoteManifests) == 0 {
		c.Deploy.KubectlDeploy.Manifests = constants.DefaultKubectlManifests
	}
}

func setDefaultLogsConfig(c *latestV1.SkaffoldConfig) {
	if c.Deploy.Logs.Prefix == "" {
		c.Deploy.Logs.Prefix = "container"
	}
}

func defaultToDockerArtifact(a *latestV1.Artifact) {
	if a.ArtifactType == (latestV1.ArtifactType{}) {
		a.ArtifactType = latestV1.ArtifactType{
			DockerArtifact: &latestV1.DockerArtifact{},
		}
	}
}

func setCustomArtifactDefaults(a *latestV1.CustomArtifact) {
	if a.Dependencies == nil {
		a.Dependencies = &latestV1.CustomDependencies{
			Paths: []string{"."},
		}
	}
}

func setBuildpackArtifactDefaults(a *latestV1.BuildpackArtifact) {
	if a.ProjectDescriptor == "" {
		a.ProjectDescriptor = constants.DefaultProjectDescriptor
	}
	if a.Dependencies == nil {
		a.Dependencies = &latestV1.BuildpackDependencies{
			Paths: []string{"."},
		}
	}
}

func setDockerArtifactDefaults(a *latestV1.DockerArtifact) {
	a.DockerfilePath = valueOrDefault(a.DockerfilePath, constants.DefaultDockerfilePath)
}

func setDefaultWorkspace(a *latestV1.Artifact) {
	a.Workspace = valueOrDefault(a.Workspace, ".")
}

func setDefaultSync(a *latestV1.Artifact) {
	if a.Sync != nil {
		if len(a.Sync.Manual) == 0 && len(a.Sync.Infer) == 0 && a.Sync.Auto == nil {
			switch {
			case a.JibArtifact != nil || a.BuildpackArtifact != nil:
				a.Sync.Auto = util.BoolPtr(true)
			default:
				a.Sync.Infer = []string{"**/*"}
			}
		}
	} else if a.BuildpackArtifact != nil {
		a.Sync = &latestV1.Sync{Auto: util.BoolPtr(true)}
	}
}

func withClusterConfig(c *latestV1.SkaffoldConfig, opts ...func(*latestV1.ClusterDetails) error) error {
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

func setDefaultClusterNamespace(cluster *latestV1.ClusterDetails) error {
	if cluster.Namespace == "" {
		ns, err := currentNamespace()
		if err != nil {
			return fmt.Errorf("getting current namespace: %w", err)
		}
		cluster.Namespace = ns
	}
	return nil
}

func setDefaultClusterTimeout(cluster *latestV1.ClusterDetails) error {
	cluster.Timeout = valueOrDefault(cluster.Timeout, kaniko.DefaultTimeout)
	return nil
}

func setDefaultClusterPullSecret(cluster *latestV1.ClusterDetails) error {
	cluster.PullSecretMountPath = valueOrDefault(cluster.PullSecretMountPath, kaniko.DefaultSecretMountPath)
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
		cluster.PullSecretName = valueOrDefault(cluster.PullSecretName, kaniko.DefaultSecretName+random)
		return nil
	}
	return nil
}

func setDefaultClusterDockerConfigSecret(cluster *latestV1.ClusterDetails) error {
	if cluster.DockerConfig == nil {
		return nil
	}

	random := ""
	if cluster.RandomDockerConfigSecret {
		uid, _ := uuid.NewUUID()
		random = uid.String()
	}

	cluster.DockerConfig.SecretName = valueOrDefault(cluster.DockerConfig.SecretName, kaniko.DefaultDockerConfigSecretName+random)

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

func defaultToKanikoArtifact(artifact *latestV1.Artifact) {
	if artifact.KanikoArtifact == nil {
		artifact.KanikoArtifact = &latestV1.KanikoArtifact{}
	}
}

func setKanikoArtifactDefaults(a *latestV1.KanikoArtifact) {
	a.Image = valueOrDefault(a.Image, kaniko.DefaultImage)
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

func setDefaultLocalPort(pf *latestV1.PortForwardResource) {
	if pf.LocalPort == 0 {
		if pf.Port.Type == schemautil.Int {
			pf.LocalPort = pf.Port.IntVal
		}
	}
}

func setDefaultAddress(pf *latestV1.PortForwardResource) {
	if pf.Address == "" {
		pf.Address = constants.DefaultPortForwardAddress
	}
}

func setDefaultArtifactDependencyAlias(d *latestV1.ArtifactDependency) {
	if d.Alias == "" {
		d.Alias = d.ImageName
	}
}

func setDefaultTestWorkspace(c *latestV1.SkaffoldConfig) {
	for _, tc := range c.Test {
		if tc == nil {
			continue
		}
		tc.Workspace = valueOrDefault(tc.Workspace, ".")
	}
}
