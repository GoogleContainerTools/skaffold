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
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
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
func Set(c *latestV2.SkaffoldConfig) error {
	defaultToLocalBuild(c)
	setDefaultTagger(c)
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

	withLocalBuild(c, func(lb *latestV2.LocalBuild) {
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
	SetDefaultRenderer(c)
	SetDefaultDeployer(c)
	return nil
}

// SetDefaultRenderer sets the default manifests to rawYaml.
func SetDefaultRenderer(c *latestV2.SkaffoldConfig) {
	if len(c.Render.Generate.Kpt) > 0 {
		return
	}
	if len(c.Render.Generate.RawK8s) > 0 {
		return
	}
	if len(c.Render.Generate.Kustomize) > 0 {
		return
	}
	if c.Render.Generate.Helm != nil {
		return
	}
	if c.Deploy.HelmDeploy != nil {
		return
	}
	// Set default manifests to "k8s/*.yaml", same as v1.
	c.Render.Generate.RawK8s = constants.DefaultKubectlManifests
}

// SetDefaultDeployer adds a default kubectl deploy configuration.
func SetDefaultDeployer(c *latestV2.SkaffoldConfig) {
	if c.Deploy.DeployType != (latestV2.DeployType{}) {
		return
	}

	log.Entry(context.TODO()).Debug("Defaulting deploy type to kubectl")
	c.Deploy.DeployType.KubectlDeploy = &latestV2.KubectlDeploy{}
}

func defaultToLocalBuild(c *latestV2.SkaffoldConfig) {
	if c.Build.BuildType != (latestV2.BuildType{}) {
		return
	}

	log.Entry(context.TODO()).Debug("Defaulting build type to local build")
	c.Build.BuildType.LocalBuild = &latestV2.LocalBuild{}
}

func withLocalBuild(c *latestV2.SkaffoldConfig, operations ...func(*latestV2.LocalBuild)) {
	if local := c.Build.LocalBuild; local != nil {
		for _, operation := range operations {
			operation(local)
		}
	}
}

func setDefaultConcurrency(local *latestV2.LocalBuild) {
	if local.Concurrency == nil {
		local.Concurrency = &constants.DefaultLocalConcurrency
	}
}

func withCloudBuildConfig(c *latestV2.SkaffoldConfig, operations ...func(*latestV2.GoogleCloudBuild)) {
	if gcb := c.Build.GoogleCloudBuild; gcb != nil {
		for _, operation := range operations {
			operation(gcb)
		}
	}
}

func setDefaultCloudBuildDockerImage(gcb *latestV2.GoogleCloudBuild) {
	gcb.DockerImage = valueOrDefault(gcb.DockerImage, defaultCloudBuildDockerImage)
}

func setDefaultCloudBuildMavenImage(gcb *latestV2.GoogleCloudBuild) {
	gcb.MavenImage = valueOrDefault(gcb.MavenImage, defaultCloudBuildMavenImage)
}

func setDefaultCloudBuildGradleImage(gcb *latestV2.GoogleCloudBuild) {
	gcb.GradleImage = valueOrDefault(gcb.GradleImage, defaultCloudBuildGradleImage)
}

func setDefaultCloudBuildKanikoImage(gcb *latestV2.GoogleCloudBuild) {
	gcb.KanikoImage = valueOrDefault(gcb.KanikoImage, defaultCloudBuildKanikoImage)
}

func setDefaultCloudBuildPackImage(gcb *latestV2.GoogleCloudBuild) {
	gcb.PackImage = valueOrDefault(gcb.PackImage, defaultCloudBuildPackImage)
}

func setDefaultTagger(c *latestV2.SkaffoldConfig) {
	if c.Build.TagPolicy != (latestV2.TagPolicy{}) {
		return
	}

	c.Build.TagPolicy = latestV2.TagPolicy{GitTagger: &latestV2.GitTagger{}}
}

func setDefaultLogsConfig(c *latestV2.SkaffoldConfig) {
	if c.Deploy.Logs.Prefix == "" {
		c.Deploy.Logs.Prefix = "container"
	}
}

func defaultToDockerArtifact(a *latestV2.Artifact) {
	if a.ArtifactType == (latestV2.ArtifactType{}) {
		a.ArtifactType = latestV2.ArtifactType{
			DockerArtifact: &latestV2.DockerArtifact{},
		}
	}
}

func setCustomArtifactDefaults(a *latestV2.CustomArtifact) {
	if a.Dependencies == nil {
		a.Dependencies = &latestV2.CustomDependencies{
			Paths: []string{"."},
		}
	}
}

func setBuildpackArtifactDefaults(a *latestV2.BuildpackArtifact) {
	if a.ProjectDescriptor == "" {
		a.ProjectDescriptor = constants.DefaultProjectDescriptor
	}
	if a.Dependencies == nil {
		a.Dependencies = &latestV2.BuildpackDependencies{
			Paths: []string{"."},
		}
	}
}

func setDockerArtifactDefaults(a *latestV2.DockerArtifact) {
	a.DockerfilePath = valueOrDefault(a.DockerfilePath, constants.DefaultDockerfilePath)
}

func setDefaultWorkspace(a *latestV2.Artifact) {
	a.Workspace = valueOrDefault(a.Workspace, ".")
}

func setDefaultSync(a *latestV2.Artifact) {
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
		a.Sync = &latestV2.Sync{Auto: util.BoolPtr(true)}
	}
}

func withClusterConfig(c *latestV2.SkaffoldConfig, opts ...func(*latestV2.ClusterDetails) error) error {
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

func setDefaultClusterNamespace(cluster *latestV2.ClusterDetails) error {
	if cluster.Namespace == "" {
		ns, err := currentNamespace()
		if err != nil {
			return fmt.Errorf("getting current namespace: %w", err)
		}
		cluster.Namespace = ns
	}
	return nil
}

func setDefaultClusterTimeout(cluster *latestV2.ClusterDetails) error {
	cluster.Timeout = valueOrDefault(cluster.Timeout, kaniko.DefaultTimeout)
	return nil
}

func setDefaultClusterPullSecret(cluster *latestV2.ClusterDetails) error {
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

func setDefaultClusterDockerConfigSecret(cluster *latestV2.ClusterDetails) error {
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

func defaultToKanikoArtifact(artifact *latestV2.Artifact) {
	if artifact.KanikoArtifact == nil {
		artifact.KanikoArtifact = &latestV2.KanikoArtifact{}
	}
}

func setKanikoArtifactDefaults(a *latestV2.KanikoArtifact) {
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

func setDefaultLocalPort(pf *latestV2.PortForwardResource) {
	if pf.LocalPort == 0 {
		if pf.Port.Type == schemautil.Int {
			pf.LocalPort = pf.Port.IntVal
		}
	}
}

func setDefaultAddress(pf *latestV2.PortForwardResource) {
	if pf.Address == "" {
		pf.Address = constants.DefaultPortForwardAddress
	}
}

func setDefaultArtifactDependencyAlias(d *latestV2.ArtifactDependency) {
	if d.Alias == "" {
		d.Alias = d.ImageName
	}
}

func setDefaultTestWorkspace(c *latestV2.SkaffoldConfig) {
	for _, tc := range c.Test {
		if tc == nil {
			continue
		}
		tc.Workspace = valueOrDefault(tc.Workspace, ".")
	}
}
