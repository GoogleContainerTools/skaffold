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

import "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"

// Config contains all the parameters for the initializer package
type Config struct {
	BuildpacksBuilder        string
	ComposeFile              string
	DefaultKustomization     string
	CliArtifacts             []string
	CliKubernetesManifests   []string
	SkipBuild                bool
	SkipDeploy               bool
	Force                    bool
	Analyze                  bool
	EnableJibInit            bool // TODO: Remove this parameter
	EnableJibGradleInit      bool
	EnableBuildpacksInit     bool
	EnableNewInitFormat      bool
	EnableManifestGeneration bool
	Opts                     config.SkaffoldOptions
	MaxFileSize              int64
}
