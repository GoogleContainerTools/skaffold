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

package defaults

import (
	"fmt"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Set makes sure default values are set on a SkaffoldPipeline.
func Set(c *latest.SkaffoldPipeline) error {
	defaultToLocalBuild(c)
	defaultToKubectlDeploy(c)
	setDefaultCloudBuildDockerImage(c)
	setDefaultTagger(c)
	setDefaultKustomizePath(c)
	setDefaultKubectlManifests(c)

	if err := withKanikoConfig(c,
		setDefaultKanikoTimeout,
		setDefaultKanikoImage,
		setDefaultKanikoNamespace,
		setDefaultKanikoSecret,
		setDefaultKanikoBuildContext,
	); err != nil {
		return err
	}

	for _, a := range c.Build.Artifacts {
		defaultToDockerArtifact(a)
		setDefaultDockerfile(a)
		setDefaultWorkspace(a)
	}

	return nil
}

func defaultToLocalBuild(c *latest.SkaffoldPipeline) {
	if c.Build.BuildType != (latest.BuildType{}) {
		return
	}

	logrus.Debugf("Defaulting build type to local build")
	c.Build.BuildType.LocalBuild = &latest.LocalBuild{}
}

func defaultToKubectlDeploy(c *latest.SkaffoldPipeline) {
	if c.Deploy.DeployType != (latest.DeployType{}) {
		return
	}

	logrus.Debugf("Defaulting deploy type to kubectl")
	c.Deploy.DeployType.KubectlDeploy = &latest.KubectlDeploy{}
}

func setDefaultCloudBuildDockerImage(c *latest.SkaffoldPipeline) {
	cloudBuild := c.Build.BuildType.GoogleCloudBuild
	if cloudBuild == nil {
		return
	}

	cloudBuild.DockerImage = valueOrDefault(cloudBuild.DockerImage, constants.DefaultCloudBuildDockerImage)
}

func setDefaultTagger(c *latest.SkaffoldPipeline) {
	if c.Build.TagPolicy != (latest.TagPolicy{}) {
		return
	}

	c.Build.TagPolicy = latest.TagPolicy{GitTagger: &latest.GitTagger{}}
}

func setDefaultKustomizePath(c *latest.SkaffoldPipeline) {
	kustomize := c.Deploy.KustomizeDeploy
	if kustomize == nil {
		return
	}

	kustomize.KustomizePath = valueOrDefault(kustomize.KustomizePath, constants.DefaultKustomizationPath)
}

func setDefaultKubectlManifests(c *latest.SkaffoldPipeline) {
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
		a.DockerArtifact.DockerfilePath = valueOrDefault(a.DockerArtifact.DockerfilePath, constants.DefaultDockerfilePath)
	}
}

func setDefaultWorkspace(a *latest.Artifact) {
	a.Workspace = valueOrDefault(a.Workspace, ".")
}

func withKanikoConfig(c *latest.SkaffoldPipeline, operations ...func(kaniko *latest.KanikoBuild) error) error {
	if kaniko := c.Build.KanikoBuild; kaniko != nil {
		for _, operation := range operations {
			if err := operation(kaniko); err != nil {
				return err
			}
		}
	}

	return nil
}

func setDefaultKanikoNamespace(kaniko *latest.KanikoBuild) error {
	if kaniko.Namespace == "" {
		ns, err := currentNamespace()
		if err != nil {
			return errors.Wrap(err, "getting current namespace")
		}

		kaniko.Namespace = ns
	}

	return nil
}

func setDefaultKanikoTimeout(kaniko *latest.KanikoBuild) error {
	kaniko.Timeout = valueOrDefault(kaniko.Timeout, constants.DefaultKanikoTimeout)
	return nil
}

func setDefaultKanikoImage(kaniko *latest.KanikoBuild) error {
	kaniko.Image = valueOrDefault(kaniko.Image, constants.DefaultKanikoImage)
	return nil
}

func setDefaultKanikoSecret(kaniko *latest.KanikoBuild) error {
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

func setDefaultKanikoBuildContext(kaniko *latest.KanikoBuild) error {
	if kaniko.BuildContext == nil {
		kaniko.BuildContext = &latest.KanikoBuildContext{
			LocalDir: &latest.LocalDir{},
		}
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
