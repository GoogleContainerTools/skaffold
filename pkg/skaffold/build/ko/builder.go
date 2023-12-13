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

package ko

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands"
	"github.com/google/ko/pkg/commands/options"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/version"
)

func (b *Builder) newKoBuilder(ctx context.Context, a *latest.Artifact, platforms platform.Matcher, tag string) (build.Interface, error) {
	ref, err := docker.ParseReference(tag)
	if err != nil {
		return nil, fmt.Errorf("parsing image %v: %w", tag, err)
	}
	imageInfoEnv := map[string]string{
		constants.ImageRef.Repo: ref.Repo,
		constants.ImageRef.Name: ref.Name,
		constants.ImageRef.Tag:  ref.Tag,
	}
	if err != nil {
		return nil, fmt.Errorf("could not resolve skaffold runtime env for ko builder: %v", err)
	}
	bo, err := buildOptions(a, b.runMode, platforms, imageInfoEnv)
	if err != nil {
		return nil, fmt.Errorf("could not construct ko build options: %v", err)
	}
	return commands.NewBuilder(ctx, bo)
}
func buildOptions(a *latest.Artifact, runMode config.RunMode, platforms platform.Matcher, envs map[string]string) (*options.BuildOptions, error) {
	buildconfig, err := buildConfig(a, envs)
	if err != nil {
		return nil, fmt.Errorf("could not create ko build config: %v", err)
	}
	imageLabels, err := labels(a)
	if err != nil {
		return nil, fmt.Errorf("could not expand image labels: %v", err)
	}
	return &options.BuildOptions{
		BaseImage:            a.KoArtifact.BaseImage,
		BuildConfigs:         buildconfig,
		ConcurrentBuilds:     1, // we could plug in Skaffold's max builds here, but it'd be incorrect if users build more than one artifact
		DisableOptimizations: runMode == config.RunModes.Debug,
		Labels:               imageLabels,
		Platforms:            platforms.Array(),
		SBOM:                 "none", // TODO: Need design for SBOM generation to consider other builders
		Trimpath:             runMode != config.RunModes.Debug,
		UserAgent:            version.UserAgentWithClient(),
		WorkingDirectory:     filepath.Join(a.Workspace, a.KoArtifact.Dir),
	}, nil
}

// buildConfig creates the ko build config map based on the artifact config.
// A map entry is only required if the artifact config specifies fields that need to be part of ko build configs.
// If none of these are specified, we can provide an empty `BuildConfigs` map.
// In this case, ko falls back to build configs provided in `.ko.yaml`, or to the default zero config.
func buildConfig(a *latest.Artifact, envs map[string]string) (map[string]build.Config, error) {
	buildconfigs := map[string]build.Config{}
	if !koArtifactSpecifiesBuildConfig(*a.KoArtifact) {
		return buildconfigs, nil
	}
	koImportpath, err := getImportPath(a)
	if err != nil {
		return nil, fmt.Errorf("could not determine import path of image %s: %v", a.ImageName, err)
	}
	env, err := expand(a.KoArtifact.Env, envs)
	if err != nil {
		return nil, fmt.Errorf("could not expand env: %v", err)
	}
	flags, err := expand(a.KoArtifact.Flags, envs)
	if err != nil {
		return nil, fmt.Errorf("could not expand build flags: %v", err)
	}
	ldflags, err := expand(a.KoArtifact.Ldflags, envs)
	if err != nil {
		return nil, fmt.Errorf("could not expand linker flags: %v", err)
	}
	importpath := strings.TrimPrefix(koImportpath, build.StrictScheme)
	buildconfigs[importpath] = build.Config{
		ID:      a.ImageName,
		Dir:     ".",
		Env:     env,
		Flags:   flags,
		Ldflags: ldflags,
		Main:    a.KoArtifact.Main,
	}
	return buildconfigs, nil
}

func koArtifactSpecifiesBuildConfig(k latest.KoArtifact) bool {
	if k.Dir != "" && k.Dir != "." {
		return true
	}
	if k.Main != "" && k.Main != "." {
		return true
	}
	if len(k.Env) != 0 {
		return true
	}
	if len(k.Flags) != 0 {
		return true
	}
	if len(k.Ldflags) != 0 {
		return true
	}
	return false
}

func labels(a *latest.Artifact) ([]string, error) {
	rawLabels := map[string]*string{}
	for k, v := range a.KoArtifact.Labels {
		rawLabels[k] = util.Ptr(v)
	}
	expandedLabels, err := util.EvaluateEnvTemplateMapWithEnv(rawLabels, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to expand image labels: %w", err)
	}
	var labels []string
	for k, v := range expandedLabels {
		labels = append(labels, fmt.Sprintf("%s=%s", k, *v))
	}
	return labels, nil
}

func expand(dryValues []string, envs map[string]string) ([]string, error) {
	var expandedValues []string
	for _, rawValue := range dryValues {
		// support ko-style envvar templating syntax, see https://github.com/GoogleContainerTools/skaffold/issues/6916
		rawValue = strings.ReplaceAll(rawValue, "{{.Env.", "{{.")
		expandedValue, err := util.ExpandEnvTemplate(rawValue, envs)
		if err != nil {
			return nil, err
		}
		expandedValues = append(expandedValues, expandedValue)
	}
	return expandedValues, nil
}
