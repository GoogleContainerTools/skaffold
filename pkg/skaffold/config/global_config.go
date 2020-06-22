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

package config

// GlobalConfig is the top level struct for the global Skaffold config
// It is unrelated to the SkaffoldConfig object (parsed from the skaffold.yaml)
type GlobalConfig struct {
	Global         *ContextConfig   `yaml:"global,omitempty"`
	ContextConfigs []*ContextConfig `yaml:"kubeContexts"`
}

// ContextConfig is the context-specific config information provided in
// the global Skaffold config.
type ContextConfig struct {
	Kubecontext        string   `yaml:"kube-context,omitempty"`
	DefaultRepo        string   `yaml:"default-repo,omitempty"`
	LocalCluster       *bool    `yaml:"local-cluster,omitempty"`
	InsecureRegistries []string `yaml:"insecure-registries,omitempty"`
	// DebugHelpersRegistry is the registry from which the debug helper images are used.
	DebugHelpersRegistry string        `yaml:"debug-helpers-registry,omitempty"`
	UpdateCheck          *bool         `yaml:"update-check,omitempty"`
	Survey               *SurveyConfig `yaml:"survey,omitempty"`
}

// SurveyConfig is the survey config information
type SurveyConfig struct {
	DisablePrompt *bool  `yaml:"disable-prompt,omitempty"`
	LastTaken     string `yaml:"last-taken,omitempty"`
	LastPrompted  string `yaml:"last-prompted,omitempty"`
}
