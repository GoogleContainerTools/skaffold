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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const Version string = "skaffold/v1alpha2"

type SkaffoldConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`

	Build    BuildConfig  `yaml:"build,omitempty"`
	Deploy   DeployConfig `yaml:"deploy,omitempty"`
	Profiles []Profile    `yaml:"profiles,omitempty"`
}

func (c *SkaffoldConfig) GetVersion() string {
	return c.APIVersion
}

// BuildConfig contains all the configuration for the build steps
type BuildConfig struct {
	Artifacts []*Artifact `yaml:"artifacts,omitempty"`
	TagPolicy TagPolicy   `yaml:"tagPolicy,omitempty"`
	BuildType `yaml:",inline"`
}

// TagPolicy contains all the configuration for the tagging step
type TagPolicy struct {
	GitTagger         *GitTagger         `yaml:"gitCommit"`
	ShaTagger         *ShaTagger         `yaml:"sha256"`
	EnvTemplateTagger *EnvTemplateTagger `yaml:"envTemplate"`
	DateTimeTagger    *DateTimeTagger    `yaml:"dateTime"`
}

// ShaTagger contains the configuration for the SHA tagger.
type ShaTagger struct{}

// GitTagger contains the configuration for the git tagger.
type GitTagger struct{}

// EnvTemplateTagger contains the configuration for the envTemplate tagger.
type EnvTemplateTagger struct {
	Template string `yaml:"template"`
}

// DateTimeTagger contains the configuration for the DateTime tagger.
type DateTimeTagger struct {
	Format   string `yaml:"format,omitempty"`
	TimeZone string `yaml:"timezone,omitempty"`
}

// BuildType contains the specific implementation and parameters needed
// for the build step. Only one field should be populated.
type BuildType struct {
	LocalBuild       *LocalBuild       `yaml:"local"`
	GoogleCloudBuild *GoogleCloudBuild `yaml:"googleCloudBuild"`
	KanikoBuild      *KanikoBuild      `yaml:"kaniko"`
}

// LocalBuild contains the fields needed to do a build on the local docker daemon
// and optionally push to a repository.
type LocalBuild struct {
	SkipPush *bool `yaml:"skipPush"`
}

// GoogleCloudBuild contains the fields needed to do a remote build on
// Google Container Builder.
type GoogleCloudBuild struct {
	ProjectID string `yaml:"projectId"`
}

// KanikoBuild contains the fields needed to do a on-cluster build using
// the kaniko image
type KanikoBuild struct {
	GCSBucket  string `yaml:"gcsBucket,omitempty"`
	PullSecret string `yaml:"pullSecret,omitempty"`
}

// DeployConfig contains all the configuration needed by the deploy steps
type DeployConfig struct {
	DeployType `yaml:",inline"`
}

// DeployType contains the specific implementation and parameters needed
// for the deploy step. Only one field should be populated.
type DeployType struct {
	HelmDeploy      *HelmDeploy      `yaml:"helm"`
	KubectlDeploy   *KubectlDeploy   `yaml:"kubectl"`
	KustomizeDeploy *KustomizeDeploy `yaml:"kustomize"`
}

// KubectlDeploy contains the configuration needed for deploying with `kubectl apply`
type KubectlDeploy struct {
	Manifests       []string `yaml:"manifests,omitempty"`
	RemoteManifests []string `yaml:"remoteManifests,omitempty"`
}

// HelmDeploy contains the configuration needed for deploying with helm
type HelmDeploy struct {
	Releases []HelmRelease `yaml:"releases,omitempty"`
}

type KustomizeDeploy struct{}

type HelmRelease struct {
	Name              string                 `yaml:"name"`
	ChartPath         string                 `yaml:"chartPath"`
	ValuesFilePath    string                 `yaml:"valuesFilePath"`
	Values            map[string]string      `yaml:"values,omitempty"`
	Namespace         string                 `yaml:"namespace"`
	Version           string                 `yaml:"version"`
	SetValues         map[string]string      `yaml:"setValues"`
	SetValueTemplates map[string]string      `yaml:"setValueTemplates"`
	Wait              bool                   `yaml:"wait"`
	Overrides         map[string]interface{} `yaml:"overrides"`
	Packaged          *HelmPackaged          `yaml:"packaged"`
}

// HelmPackaged represents parameters for packaging helm chart.
type HelmPackaged struct {
	// Version sets the version on the chart to this semver version.
	Version string `yaml:"version"`

	// AppVersion set the appVersion on the chart to this version
	AppVersion string `yaml:"appVersion"`
}

// Artifact represents items that need to be built, along with the context in which
// they should be built.
type Artifact struct {
	ImageName    string `yaml:"imageName"`
	Workspace    string `yaml:"workspace,omitempty"`
	ArtifactType `yaml:",inline"`
}

// Profile is additional configuration that overrides default
// configuration when it is activated.
type Profile struct {
	Name   string       `yaml:"name"`
	Build  BuildConfig  `yaml:"build,omitempty"`
	Deploy DeployConfig `yaml:"deploy,omitempty"`
}

type ArtifactType struct {
	DockerArtifact *DockerArtifact `yaml:"docker"`
	BazelArtifact  *BazelArtifact  `yaml:"bazel"`
}

type DockerArtifact struct {
	DockerfilePath string             `yaml:"dockerfilePath,omitempty"`
	BuildArgs      map[string]*string `yaml:"buildArgs,omitempty"`
}

type BazelArtifact struct {
	BuildTarget string `yaml:"target"`
}

// Parse reads a SkaffoldConfig from yaml.
func (c *SkaffoldConfig) Parse(contents []byte, useDefaults bool) error {
	if err := yaml.UnmarshalStrict(contents, c); err != nil {
		return err
	}

	if useDefaults {
		if err := c.setDefaultValues(); err != nil {
			return errors.Wrap(err, "applying default values")
		}
	}

	return nil
}

func (c *SkaffoldConfig) setDefaultValues() error {
	c.defaultToLocalBuild()
	c.defaultToDockerArtifacts()
	c.setDefaultTagger()
	c.setDefaultDockerfiles()
	c.setDefaultWorkspaces()
	return c.expandKanikoSecretPath()
}

func (c *SkaffoldConfig) defaultToLocalBuild() {
	if c.Build.BuildType != (BuildType{}) {
		return
	}

	logrus.Debugf("Defaulting build type to local build")
	c.Build.BuildType.LocalBuild = &LocalBuild{}
}

func (c *SkaffoldConfig) defaultToDockerArtifacts() {
	for _, artifact := range c.Build.Artifacts {
		if artifact.ArtifactType != (ArtifactType{}) {
			continue
		}

		artifact.ArtifactType = ArtifactType{
			DockerArtifact: &DockerArtifact{},
		}
	}
}

func (c *SkaffoldConfig) setDefaultTagger() {
	if c.Build.TagPolicy != (TagPolicy{}) {
		return
	}

	c.Build.TagPolicy = TagPolicy{GitTagger: &GitTagger{}}
}

func (c *SkaffoldConfig) setDefaultDockerfiles() {
	for _, artifact := range c.Build.Artifacts {
		if artifact.DockerArtifact != nil && artifact.DockerArtifact.DockerfilePath == "" {
			artifact.DockerArtifact.DockerfilePath = constants.DefaultDockerfilePath
		}
	}
}

func (c *SkaffoldConfig) setDefaultWorkspaces() {
	for _, artifact := range c.Build.Artifacts {
		if artifact.Workspace == "" {
			artifact.Workspace = "."
		}
	}
}

func (c *SkaffoldConfig) expandKanikoSecretPath() error {
	if c.Build.KanikoBuild == nil || c.Build.KanikoBuild.PullSecret == "" {
		return nil
	}

	absPath, err := homedir.Expand(c.Build.KanikoBuild.PullSecret)
	if err != nil {
		return fmt.Errorf("unable to expand pullSecret %s", c.Build.KanikoBuild.PullSecret)
	}

	c.Build.KanikoBuild.PullSecret = absPath
	return nil
}

// ApplyProfiles returns configuration modified by the application
// of a list of profiles.
func (c *SkaffoldConfig) ApplyProfiles(profiles []string) error {
	var err error

	byName := profilesByName(c.Profiles)
	for _, name := range profiles {
		profile, present := byName[name]
		if !present {
			return fmt.Errorf("couldn't find profile %s", name)
		}

		err = applyProfile(c, profile)
		if err != nil {
			return errors.Wrapf(err, "applying profile %s", name)
		}
	}

	c.Profiles = nil
	if err := c.setDefaultValues(); err != nil {
		return errors.Wrap(err, "applying default values")
	}

	return nil
}

func applyProfile(config *SkaffoldConfig, profile Profile) error {
	logrus.Infof("Applying profile: %s", profile.Name)

	buf, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(buf, config)
}

func profilesByName(profiles []Profile) map[string]Profile {
	byName := make(map[string]Profile)
	for _, profile := range profiles {
		byName[profile.Name] = profile
	}
	return byName
}
