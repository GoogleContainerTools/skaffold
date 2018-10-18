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

package v1alpha2

import (
	"fmt"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
)

// SetDefaultValues makes sure default values are set.
func (c *SkaffoldPipeline) SetDefaultValues() error {
	c.defaultToLocalBuild()
	c.defaultToKubectlDeploy()
	c.setDefaultCloudBuildDockerImage()
	c.setDefaultTagger()
	c.setDefaultKustomizePath()
	c.setDefaultKubectlManifests()
	c.setDefaultKanikoTimeout()
	if err := c.setDefaultKanikoNamespace(); err != nil {
		return err
	}
	if err := c.setDefaultKanikoSecret(); err != nil {
		return err
	}

	for _, a := range c.Build.Artifacts {
		c.defaultToDockerArtifact(a)
		c.setDefaultDockerfile(a)
		c.setDefaultWorkspace(a)
	}

	return nil
}

func (c *SkaffoldPipeline) defaultToLocalBuild() {
	if c.Build.BuildType != (BuildType{}) {
		return
	}

	logrus.Debugf("Defaulting build type to local build")
	c.Build.BuildType.LocalBuild = &LocalBuild{}
}

func (c *SkaffoldPipeline) defaultToKubectlDeploy() {
	if c.Deploy.DeployType != (DeployType{}) {
		return
	}

	logrus.Debugf("Defaulting deploy type to kubectl")
	c.Deploy.DeployType.KubectlDeploy = &KubectlDeploy{}
}

func (c *SkaffoldPipeline) setDefaultCloudBuildDockerImage() {
	cloudBuild := c.Build.BuildType.GoogleCloudBuild
	if cloudBuild == nil {
		return
	}

	if cloudBuild.DockerImage == "" {
		cloudBuild.DockerImage = constants.DefaultCloudBuildDockerImage
	}
}

func (c *SkaffoldPipeline) setDefaultTagger() {
	if c.Build.TagPolicy != (TagPolicy{}) {
		return
	}

	c.Build.TagPolicy = TagPolicy{GitTagger: &GitTagger{}}
}

func (c *SkaffoldPipeline) setDefaultKustomizePath() {
	if c.Deploy.KustomizeDeploy != nil && c.Deploy.KustomizeDeploy.KustomizePath == "" {
		c.Deploy.KustomizeDeploy.KustomizePath = constants.DefaultKustomizationPath
	}
}

func (c *SkaffoldPipeline) setDefaultKubectlManifests() {
	if c.Deploy.KubectlDeploy != nil && len(c.Deploy.KubectlDeploy.Manifests) == 0 {
		c.Deploy.KubectlDeploy.Manifests = constants.DefaultKubectlManifests
	}
}

func (c *SkaffoldPipeline) defaultToDockerArtifact(a *Artifact) {
	if a.ArtifactType == (ArtifactType{}) {
		a.ArtifactType = ArtifactType{
			DockerArtifact: &DockerArtifact{},
		}
	}
}

func (c *SkaffoldPipeline) setDefaultDockerfile(a *Artifact) {
	if a.DockerArtifact != nil && a.DockerArtifact.DockerfilePath == "" {
		a.DockerArtifact.DockerfilePath = constants.DefaultDockerfilePath
	}
}

func (c *SkaffoldPipeline) setDefaultWorkspace(a *Artifact) {
	if a.Workspace == "" {
		a.Workspace = "."
	}
}

func (c *SkaffoldPipeline) setDefaultKanikoNamespace() error {
	kaniko := c.Build.KanikoBuild
	if kaniko == nil {
		return nil
	}

	if kaniko.Namespace == "" {
		ns, err := currentNamespace()
		if err != nil {
			return errors.Wrap(err, "getting current namespace")
		}

		kaniko.Namespace = ns
	}

	return nil
}

func (c *SkaffoldPipeline) setDefaultKanikoTimeout() {
	kaniko := c.Build.KanikoBuild
	if kaniko == nil {
		return
	}

	if kaniko.Timeout == "" {
		kaniko.Timeout = constants.DefaultKanikoTimeout
	}
}

func (c *SkaffoldPipeline) setDefaultKanikoSecret() error {
	kaniko := c.Build.KanikoBuild
	if kaniko == nil {
		return nil
	}

	if kaniko.PullSecretName == "" {
		kaniko.PullSecretName = constants.DefaultKanikoSecretName
	}

	if kaniko.PullSecret != "" {
		absPath, err := homedir.Expand(kaniko.PullSecret)
		if err != nil {
			return fmt.Errorf("unable to expand pullSecret %s", kaniko.PullSecret)
		}

		kaniko.PullSecret = absPath
		return nil
	}

	return nil
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
