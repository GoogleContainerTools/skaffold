/*
Copyright 2021 The Skaffold Authors

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

package v2

import (
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

const Version string = "skaffold/v3alpha1"

// NewSkaffoldConfig creates a SkaffoldConfig
func NewSkaffoldConfig() util.VersionedConfig {
	return new(SkaffoldConfig)
}

func (c *SkaffoldConfig) GetVersion() string {
	return c.APIVersion
}

// SkaffoldConfig holds the fields parsed from the Skaffold configuration file (skaffold.yaml).
type SkaffoldConfig struct {
	// APIVersion is the version of the configuration.
	APIVersion string `yaml:"apiVersion" yamltags:"required"`

	// Kind is always `Config`. Defaults to `Config`.
	Kind string `yaml:"kind" yamltags:"required"`

	// Metadata holds additional information about the config.
	Metadata Metadata `yaml:"metadata,omitempty"`

	// Dependencies describes a list of other required configs for the current config.
	Dependencies []latestV1.ConfigDependency `yaml:"requires,omitempty"`

	// Pipeline defines the Build/Test/Deploy phases.
	Pipeline `yaml:",inline"`

	// Profiles *beta* can override be used to `build`, `test` or `deploy` configuration.
	Profiles []Profile `yaml:"profiles,omitempty"`
}

// Metadata holds an optional name of the project.
type Metadata struct {
	// Name is an identifier for the project.
	Name string `yaml:"name,omitempty"`
}

// Pipeline describes a Skaffold pipeline.
type Pipeline struct {
	// Build describes how images are built.
	Build latestV1.BuildConfig `yaml:"build,omitempty"`

	// Test describes how images are tested.
	Test []*latestV1.TestCase `yaml:"test,omitempty"`

	// Render describes how the original manifests are hydrated, validated and transformed.
	Render RenderConfig `yaml:"manifests,omitempty"`

	// Deploy describes how the manifests are deployed.
	Deploy DeployConfig `yaml:"deploy,omitempty"`

	// PortForward describes user defined resources to port-forward.
	PortForward []*latestV1.PortForwardResource `yaml:"portForward,omitempty"`
}

// RenderConfig contains all the configuration needed by the render steps.
type RenderConfig struct {

	// Generate defines the dry manifests from a variety of sources.
	Generate *Generate `yaml:"generate,omitempty"`

	// Transform defines a set of transformation operations to run in series
	Transform *[]Transformer `yaml:"transform,omitempty"`

	// Validate defines a set of validator operations to run in series.
	Validate *[]Validator `yaml:"validate,omitempty"`

	// Output is the path to the hydrated directory.
	Output string `yaml:"output,omitempty"`
}

// Generate defines the dry manifests from a variety of sources.
type Generate struct {

	// Manifests contains the raw kubernetes manifest paths and kustomize paths.
	Manifests []string `yaml:"manifests,omitempty"`

	// TODO: Implement the "HelmCharts" and "RemoteResoruces" fields.
}

// Transformer describes the supported kpt transformers.
type Transformer struct {
	// Name is the transformer name. Can only accept skaffold whitelisted tools.
	Name string `yaml:"name" yamltags:"required"`
}

// Transformer describes the supported kpt transformers.
type Validator struct {
	// Name is the Validator name. Can only accept skaffold whitelisted tools.
	Name string `yaml:"name" yamltags:"required"`
}

// DeployConfig contains all the configuration needed by the deploy steps.
type DeployConfig struct {

	// Dir is equivalent to the dir in `kpt live apply <dir>`. If not provided, skaffold renders the raw manifests
	// and store them to a a hidden directory `.kpt-hydrated`, and deploys the hidden directory.
	Dir string `yaml:"dir,omitempty"`

	// InventoryID *alpha* is the identifier for a group of applied resources.
	// This value is only needed when the `kpt live` is working on a pre-applied cluster resources.
	InventoryID string `yaml:"inventoryID,omitempty"`
	// InventoryNamespace *alpha* sets the inventory namespace.
	InventoryNamespace string `yaml:"inventoryNamespace,omitempty"`

	// StatusCheckDeadlineSeconds sets for the polling period for resource statuses. Default to 2s. Values can be "2s", "1min", "3h", etc
	StatusCheckDeadlineSeconds string `statusCheckDeadlineSeconds:"pollPeriod,omitempty"`
	// PrunePropagationPolicy sets the propagation policy for pruning.
	// Possible settings are Background, Foreground, Orphan.
	// Default to "Background".
	PrunePropagationPolicy string `yaml:"prunePropagationPolicy,omitempty"`
	// PruneTimeout sets the time threshold to wait for all pruned resources to be deleted.
	PruneTimeout string `yaml:"pruneTimeout,omitempty"`
	// ReconcileTimeout sets the time threshold to wait for all resources to reach the current status.
	ReconcileTimeout string `yaml:"reconcileTimeout,omitempty"`

	// KubeContext is the Kubernetes context that Skaffold should deploy to.
	// For example: `minikube`.
	KubeContext string `yaml:"kubeContext,omitempty"`

	// Logs configures how container logs are printed as a result of a deployment.
	Logs LogsConfig `yaml:"logs,omitempty"`
}

// LogsConfig configures how container logs are printed as a result of a deployment.
type LogsConfig struct {
	// Prefix defines the prefix shown on each log line. Valid values are
	// `container`: prefix logs lines with the name of the container.
	// `podAndContainer`: prefix logs lines with the names of the pod and of the container.
	// `auto`: same as `podAndContainer` except that the pod name is skipped if it's the same as the container name.
	// `none`: don't add a prefix.
	// Defaults to `auto`.
	Prefix string `yaml:"prefix,omitempty"`
}

// Profile is used to override any `build`, `test` or `deploy` configuration.
type Profile struct {
	// Name is a unique profile name.
	// For example: `profile-prod`.
	Name string `yaml:"name,omitempty" yamltags:"required"`

	// Activation criteria by which a profile can be auto-activated.
	// The profile is auto-activated if any one of the activations are triggered.
	// An activation is triggered if all of the criteria (env, kubeContext, command) are triggered.
	Activation []latestV1.Activation `yaml:"activation,omitempty"`

	// Patches lists patches applied to the configuration.
	// Patches use the JSON patch notation.
	Patches []latestV1.JSONPatch `yaml:"patches,omitempty"`

	// Pipeline contains the definitions to replace the default skaffold pipeline.
	Pipeline `yaml:",inline"`
}
