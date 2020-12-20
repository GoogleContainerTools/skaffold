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

package cmd

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/initializer/config"
)

const maxFileSize = 1024 * 1024 * 512

var (
	composeFile              string
	buildpacksBuilder        string
	defaultKustomization     string
	cliArtifacts             []string
	cliKubernetesManifests   []string
	skipBuild                bool
	skipDeploy               bool
	force                    bool
	analyze                  bool
	enableJibInit            bool
	enableJibGradleInit      bool
	enableBuildpacksInit     bool
	enableNewInitFormat      bool
	enableManifestGeneration bool
)

// for testing
var initEntrypoint = initializer.DoInit

// NewCmdInit describes the CLI command to generate a Skaffold configuration.
func NewCmdInit() *cobra.Command {
	return NewCmd("init").
		WithDescription("[alpha] Generate configuration for deploying an application").
		WithCommonFlags().
		WithFlags([]*Flag{
			{Value: &skipBuild, Name: "skip-build", DefValue: false, Usage: "Skip generating build artifacts in Skaffold config", IsEnum: true},
			{Value: &skipDeploy, Name: "skip-deploy", DefValue: false, Usage: "Skip generating deploy stanza in Skaffold config", Hidden: true, IsEnum: true},
			{Value: &force, Name: "force", DefValue: false, Usage: "Force the generation of the Skaffold config", IsEnum: true},
			{Value: &composeFile, Name: "compose-file", DefValue: "", Usage: "Initialize from a docker-compose file"},
			{Value: &defaultKustomization, Name: "default-kustomization", DefValue: "", Usage: "Default Kustomization overlay path (others will be added as profiles)"},
			{Value: &cliArtifacts, Name: "artifact", Shorthand: "a", DefValue: []string{}, Usage: "'='-delimited Dockerfile/image pair, or JSON string, to generate build artifact\n(example: --artifact='{\"builder\":\"Docker\",\"payload\":{\"path\":\"/web/Dockerfile.web\"},\"image\":\"gcr.io/web-project/image\"}')"},
			{Value: &cliKubernetesManifests, Name: "kubernetes-manifest", Shorthand: "k", DefValue: []string{}, Usage: "A path or a glob pattern to kubernetes manifests (can be non-existent) to be added to the kubectl deployer (overrides detection of kubernetes manifests). Repeat the flag for multiple entries. E.g.: skaffold init -k pod.yaml -k k8s/*.yml"},
			{Value: &analyze, Name: "analyze", DefValue: false, Usage: "Print all discoverable Dockerfiles and images in JSON format to stdout", IsEnum: true},
			{Value: &enableNewInitFormat, Name: "XXenableNewInitFormat", DefValue: false, Usage: "", Hidden: true, IsEnum: true},
			{Value: &enableJibInit, Name: "XXenableJibInit", DefValue: false, Usage: "", Hidden: true, IsEnum: true},
			{Value: &enableJibGradleInit, Name: "XXenableJibGradleInit", DefValue: false, Usage: "", Hidden: true, IsEnum: true},
			{Value: &enableBuildpacksInit, Name: "XXenableBuildpacksInit", DefValue: false, Usage: "", Hidden: true, IsEnum: true},
			{Value: &buildpacksBuilder, Name: "XXdefaultBuildpacksBuilder", DefValue: "gcr.io/buildpacks/builder:v1", Usage: "", Hidden: true},
			{Value: &enableManifestGeneration, Name: "generate-manifests", DefValue: false, Usage: "Allows skaffold to try and generate basic kubernetes resources to get your project started", IsEnum: true},
		}).
		NoArgs(doInit)
}

func doInit(ctx context.Context, out io.Writer) error {
	return initEntrypoint(ctx, out, config.Config{
		BuildpacksBuilder:        buildpacksBuilder,
		ComposeFile:              composeFile,
		DefaultKustomization:     defaultKustomization,
		CliArtifacts:             cliArtifacts,
		CliKubernetesManifests:   cliKubernetesManifests,
		SkipBuild:                skipBuild,
		SkipDeploy:               skipDeploy,
		Force:                    force,
		Analyze:                  analyze,
		EnableJibInit:            enableJibInit,
		EnableJibGradleInit:      enableJibGradleInit,
		EnableBuildpacksInit:     enableBuildpacksInit,
		EnableNewInitFormat:      enableNewInitFormat || enableBuildpacksInit || enableJibInit,
		EnableManifestGeneration: enableManifestGeneration,
		Opts:                     opts,
		MaxFileSize:              maxFileSize,
	})
}
