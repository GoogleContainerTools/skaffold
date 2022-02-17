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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

func (b *Builder) newKoBuilder(ctx context.Context, a *latestV2.Artifact) (build.Interface, error) {
	bo, err := buildOptions(a, b.runMode)
	if err != nil {
		return nil, fmt.Errorf("could not construct ko build options: %v", err)
	}
	return commands.NewBuilder(ctx, bo)
}

func buildOptions(a *latestV2.Artifact, runMode config.RunMode) (*options.BuildOptions, error) {
	buildconfig, err := buildConfig(a)
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
		Platform:             strings.Join(a.KoArtifact.Platforms, ","),
		Trimpath:             runMode != config.RunModes.Debug,
		UserAgent:            version.UserAgentWithClient(),
		WorkingDirectory:     filepath.Join(a.Workspace, a.KoArtifact.Dir),
	}, nil
}

// buildConfig creates the ko build config map based on the artifact config.
// A map entry is only required if the artifact config specifies fields that need to be part of ko build configs.
// If none of these are specified, we can provide an empty `BuildConfigs` map.
// In this case, ko falls back to build configs provided in `.ko.yaml`, or to the default zero config.
func buildConfig(a *latestV2.Artifact) (map[string]build.Config, error) {
	buildconfigs := map[string]build.Config{}
	if !koArtifactSpecifiesBuildConfig(*a.KoArtifact) {
		return buildconfigs, nil
	}
	koImportpath, err := getImportPath(a)
	if err != nil {
		return nil, fmt.Errorf("could not determine import path of image %s: %v", a.ImageName, err)
	}
	env, err := expand(a.KoArtifact.Env)
	if err != nil {
		return nil, fmt.Errorf("could not expand env: %v", err)
	}
	flags, err := expand(a.KoArtifact.Flags)
	if err != nil {
		return nil, fmt.Errorf("could not expand build flags: %v", err)
	}
	ldflags, err := expand(a.KoArtifact.Ldflags)
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

func koArtifactSpecifiesBuildConfig(k latestV2.KoArtifact) bool {
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

func labels(a *latestV2.Artifact) ([]string, error) {
	rawLabels := map[string]*string{}
	for k, v := range a.KoArtifact.Labels {
		rawLabels[k] = util.StringPtr(v)
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

func expand(dryValues []string) ([]string, error) {
	var expandedValues []string
	for _, rawValue := range dryValues {
		// support ko-style envvar templating syntax, see https://github.com/GoogleContainerTools/skaffold/issues/6916
		rawValue = strings.ReplaceAll(rawValue, "{{.Env.", "{{.")
		expandedValue, err := util.ExpandEnvTemplate(rawValue, nil)
		if err != nil {
			return nil, fmt.Errorf("could not expand %s", rawValue)
		}
		expandedValues = append(expandedValues, expandedValue)
	}
	return expandedValues, nil
}
