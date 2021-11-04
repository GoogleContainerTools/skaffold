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
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

func (b *Builder) newKoBuilder(ctx context.Context, a *latestV1.Artifact) (build.Interface, error) {
	bo, err := buildOptions(a, b.runMode)
	if err != nil {
		return nil, fmt.Errorf("could not construct ko build options: %v", err)
	}
	return commands.NewBuilder(ctx, bo)
}

func buildOptions(a *latestV1.Artifact, runMode config.RunMode) (*options.BuildOptions, error) {
	koImportpath, err := getImportPath(a)
	if err != nil {
		return nil, fmt.Errorf("could not determine import path: %v", err)
	}
	importpath := strings.TrimPrefix(koImportpath, build.StrictScheme)
	return &options.BuildOptions{
		BaseImage: a.KoArtifact.BaseImage,
		BuildConfigs: map[string]build.Config{
			importpath: {
				ID:      a.ImageName,
				Dir:     ".",
				Env:     a.KoArtifact.Env,
				Flags:   a.KoArtifact.Flags,
				Ldflags: a.KoArtifact.Ldflags,
				Main:    a.KoArtifact.Main,
			},
		},
		ConcurrentBuilds:     1, // we could plug in Skaffold's max builds here, but it'd be incorrect if users build more than one artifact
		DisableOptimizations: runMode == config.RunModes.Debug,
		Labels:               labels(a),
		Platform:             strings.Join(a.KoArtifact.Platforms, ","),
		UserAgent:            version.UserAgentWithClient(),
		WorkingDirectory:     filepath.Join(a.Workspace, a.KoArtifact.Dir),
	}, nil
}

func labels(a *latestV1.Artifact) []string {
	var labels []string
	for k, v := range a.KoArtifact.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}
	return labels
}
