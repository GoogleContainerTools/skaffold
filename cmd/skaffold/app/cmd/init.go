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
	"github.com/spf13/pflag"

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
		WithFlags(func(f *pflag.FlagSet) {
			f.BoolVar(&skipBuild, "skip-build", false, "Skip generating build artifacts in Skaffold config")
			f.BoolVar(&skipDeploy, "skip-deploy", false, "Skip generating deploy stanza in Skaffold config")
			f.MarkHidden("skip-deploy")
			f.BoolVar(&force, "force", false, "Force the generation of the Skaffold config")
			f.StringVar(&composeFile, "compose-file", "", "Initialize from a docker-compose file")
			f.StringVar(&defaultKustomization, "default-kustomization", "", "Default Kustomization overlay path (others will be added as profiles)")
			f.StringArrayVarP(&cliArtifacts, "artifact", "a", nil, "'='-delimited Dockerfile/image pair, or JSON string, to generate build artifact\n(example: --artifact='{\"builder\":\"Docker\",\"payload\":{\"path\":\"/web/Dockerfile.web\"},\"image\":\"gcr.io/web-project/image\"}')")
			f.StringArrayVarP(&cliKubernetesManifests, "kubernetes-manifest", "k", nil, "A path or a glob pattern to kubernetes manifests (can be non-existent) to be added to the kubectl deployer (overrides detection of kubernetes manifests). Repeat the flag for multiple entries. E.g.: skaffold init -k pod.yaml -k k8s/*.yml")
			f.BoolVar(&analyze, "analyze", false, "Print all discoverable Dockerfiles and images in JSON format to stdout")
			f.BoolVar(&enableNewInitFormat, "XXenableNewInitFormat", false, "")
			f.MarkHidden("XXenableNewInitFormat")
			f.BoolVar(&enableJibInit, "XXenableJibInit", false, "")
			f.MarkHidden("XXenableJibInit")
			f.BoolVar(&enableJibGradleInit, "XXenableJibGradleInit", false, "")
			f.MarkHidden("XXenableJibGradleInit")
			f.BoolVar(&enableBuildpacksInit, "XXenableBuildpacksInit", false, "")
			f.MarkHidden("XXenableBuildpacksInit")
			f.StringVar(&buildpacksBuilder, "XXdefaultBuildpacksBuilder", "gcr.io/buildpacks/builder:v1", "")
			f.MarkHidden("XXdefaultBuildpacksBuilder")
			f.BoolVar(&enableManifestGeneration, "XXenableManifestGeneration", false, "")
			f.MarkHidden("XXenableManifestGeneration")
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
