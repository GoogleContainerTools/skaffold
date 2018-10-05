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

package latest

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SetDefaultValues makes sure default values are set.
func (c *SkaffoldConfig) SetDefaultValues() error {
	c.defaultToLocalBuild()
	c.defaultToKubectlDeploy()
	c.setDefaultCloudBuildDockerImage()
	c.setDefaultTagger()
	c.setDefaultKustomizePath()
	c.setDefaultKubectlManifests()

	if err := c.withKanikoConfig(
		setDefaultKanikoTimeout,
		setDefaultKanikoImage,
		setDefaultKanikoNamespace,
		setDefaultKanikoSecret,
	); err != nil {
		return err
	}

	for _, a := range c.Build.Artifacts {
		c.defaultToDockerArtifact(a)
		c.setDefaultDockerfile(a)
		c.setDefaultWorkspace(a)
	}

	return nil
}

func (c *SkaffoldConfig) defaultToLocalBuild() {
	if c.Build.BuildType != (BuildType{}) {
		return
	}

	logrus.Debugf("Defaulting build type to local build")
	c.Build.BuildType.LocalBuild = &LocalBuild{}
}

func (c *SkaffoldConfig) defaultToKubectlDeploy() {
	if c.Deploy.DeployType != (DeployType{}) {
		return
	}

	logrus.Debugf("Defaulting deploy type to kubectl")
	c.Deploy.DeployType.KubectlDeploy = &KubectlDeploy{}
}

func (c *SkaffoldConfig) setDefaultCloudBuildDockerImage() {
	cloudBuild := c.Build.BuildType.GoogleCloudBuild
	if cloudBuild == nil {
		return
	}

	cloudBuild.DockerImage = valueOrDefault(cloudBuild.DockerImage, constants.DefaultCloudBuildDockerImage)
}

func (c *SkaffoldConfig) setDefaultTagger() {
	if c.Build.TagPolicy != (TagPolicy{}) {
		return
	}

	c.Build.TagPolicy = TagPolicy{GitTagger: &GitTagger{}}
}

func (c *SkaffoldConfig) setDefaultKustomizePath() {
	kustomize := c.Deploy.KustomizeDeploy
	if kustomize == nil {
		return
	}

	kustomize.KustomizePath = valueOrDefault(kustomize.KustomizePath, constants.DefaultKustomizationPath)
}

func (c *SkaffoldConfig) setDefaultKubectlManifests() {
	if c.Deploy.KubectlDeploy != nil && len(c.Deploy.KubectlDeploy.Manifests) == 0 {
		c.Deploy.KubectlDeploy.Manifests = constants.DefaultKubectlManifests
	}
}

func (c *SkaffoldConfig) defaultToDockerArtifact(a *Artifact) {
	if a.ArtifactType == (ArtifactType{}) {
		a.ArtifactType = ArtifactType{
			DockerArtifact: &DockerArtifact{},
		}
	}
}

func (c *SkaffoldConfig) setDefaultDockerfile(a *Artifact) {
	if a.DockerArtifact != nil {
		a.DockerArtifact.DockerfilePath = valueOrDefault(a.DockerArtifact.DockerfilePath, constants.DefaultDockerfilePath)
	}
}

func (c *SkaffoldConfig) setDefaultWorkspace(a *Artifact) {
	a.Workspace = valueOrDefault(a.Workspace, ".")
}

func (c *SkaffoldConfig) withKanikoConfig(operations ...func(kaniko *KanikoBuild) error) error {
	if kaniko := c.Build.KanikoBuild; kaniko != nil {
		for _, operation := range operations {
			if err := operation(kaniko); err != nil {
				return err
			}
		}
	}

	return nil
}

func setDefaultKanikoNamespace(kaniko *KanikoBuild) error {
	if kaniko.Namespace == "" {
		ns, err := currentNamespace()
		if err != nil {
			return errors.Wrap(err, "getting current namespace")
		}

		kaniko.Namespace = ns
	}

	return nil
}

func setDefaultKanikoTimeout(kaniko *KanikoBuild) error {
	kaniko.Timeout = valueOrDefault(kaniko.Timeout, constants.DefaultKanikoTimeout)
	return nil
}

func setDefaultKanikoImage(kaniko *KanikoBuild) error {
	kaniko.Image = valueOrDefault(kaniko.Image, constants.DefaultKanikoImage)
	return nil
}

func setDefaultKanikoSecret(kaniko *KanikoBuild) error {
	kaniko.PullSecretName = valueOrDefault(kaniko.PullSecretName, constants.DefaultKanikoSecretName)

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
