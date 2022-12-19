/*
Copyright 2022 The Skaffold Authors

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

package gcb

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/cloudbuild/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schema "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/v2beta28"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

// koBuildSpec creates a Cloud Build configuration using the `ko` builder.
//
// Because Skaffold embeds ko as a library, the `skaffold` binary builds the container image on Cloud Build.
//
// The method uses the artifact input argument to generate a Skaffold Config manifest.
func (b *Builder) koBuildSpec(ctx context.Context, artifact *latest.Artifact, tag string, platforms platform.Matcher) (cloudbuild.Build, error) {
	imageName, imageTag := splitImageNameAndTag(tag)
	insecureRegistries := getKeys(b.cfg.GetInsecureRegistries())
	skaffoldConfig := createSkaffoldConfig(artifact, imageName, platforms.Array(), insecureRegistries)
	skaffoldYaml, err := yaml.Marshal(skaffoldConfig)
	if err != nil {
		return cloudbuild.Build{}, fmt.Errorf("marshalling Skaffold Config YAML for %s: %w", tag, err)
	}
	log.Entry(ctx).Debugf("Skaffold config for Cloud Build:\n%s\n", skaffoldYaml)
	verbosity := log.GetLevel().String()
	return cloudBuildConfig(b.KoImage, skaffoldYaml, imageTag, verbosity, artifact.KoArtifact.Env...), nil
}

func splitImageNameAndTag(imageNameWithTag string) (string, string) {
	nameAndTag := strings.Split(imageNameWithTag, ":")
	imageName := nameAndTag[0]
	imageTag := "latest"
	if len(nameAndTag) > 1 {
		imageTag = nameAndTag[1]
	}
	return imageName, imageTag
}

func getKeys(in map[string]bool) []string {
	var keys []string
	for v := range in {
		keys = append(keys, v)
	}
	return keys
}

// createSkaffoldConfig creates a Skaffold Config manifest for use in a build step on Cloud Build.
//
// The manifest uses a specific schema version that is known to be supported by a public Skaffold image on GCR/AR.
// The schema version must be a version supported by a public Skaffold image.
// The latest schema version is not used for two reasons:
//
// 1. There could be a mismatch between the versions of Skaffold used locally and on Cloud Build.
//
//	For instance, a team could choose to use an LTS image for builds on Cloud Build.
//
// 2. The local version of Skaffold may not be available as a public image on GCR/AR.
//
// The manifest does not include artifact fields that are irrelevant for a remote build, such as Dependencies, LifecycleHooks, and Sync.
func createSkaffoldConfig(artifact *latest.Artifact, imageName string, platforms []string, insecureRegistries []string) *schema.SkaffoldConfig {
	return &schema.SkaffoldConfig{
		APIVersion: schema.Version,
		Kind:       "Config",
		Pipeline: schema.Pipeline{
			Build: schema.BuildConfig{
				Artifacts: []*schema.Artifact{{
					// Replace `ImageName` since we need the fully resolved name (with the Skaffold default repo).
					ImageName: imageName,
					// Copy values from the `artifact` function argument.
					ArtifactType: schema.ArtifactType{
						KoArtifact: &schema.KoArtifact{
							BaseImage: artifact.KoArtifact.BaseImage,
							Dir:       artifact.KoArtifact.Dir,
							Env:       artifact.KoArtifact.Env,
							Flags:     artifact.KoArtifact.Flags,
							Labels:    artifact.KoArtifact.Labels,
							Ldflags:   artifact.KoArtifact.Ldflags,
							Main:      artifact.KoArtifact.Main,
						},
					},
					Platforms: artifact.Platforms, // platforms defined for the artifact
					Workspace: artifact.Workspace,
				}},
				InsecureRegistries: insecureRegistries,
				Platforms:          platforms, // platforms provided via command-line flag, or envvar
			},
		},
	}
}

// cloudBuildConfig creates a single step build configuraration using the provided image and Skaffold config.
//
// The build step writes out the generated Skaffold Config manifest to a temporary file.
// Skaffold uses this Config to build and push the image.
func cloudBuildConfig(koImage string, skaffoldYaml []byte, imageTag string, verbosity string, env ...string) cloudbuild.Build {
	return cloudbuild.Build{
		Steps: []*cloudbuild.BuildStep{{
			Name:       koImage,
			Entrypoint: "sh",
			Args: []string{"-c", strings.Join(
				[]string{
					"skaffoldConfigFile=$(mktemp)",
					// here document with quoted end marker to ensure no subsitution or expansion
					fmt.Sprintf("cat << 'EOF' > $skaffoldConfigFile\n%s\nEOF", skaffoldYaml),
					fmt.Sprintf("skaffold build --filename $skaffoldConfigFile --tag %s --verbosity %s", imageTag, verbosity),
				},
				"\n",
			)},
			Env: skaffoldGCBEnv(env...),
		}},
	}
}

func skaffoldGCBEnv(env ...string) []string {
	defaultEnv := []string{
		"SKAFFOLD_DETECT_MINIKUBE=false",
		"SKAFFOLD_INTERACTIVE=false",
		"SKAFFOLD_UPDATE_CHECK=false",
	}
	return append(defaultEnv, env...)
}
