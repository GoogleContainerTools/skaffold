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
	bo := buildOptions(a, b.runMode)
	return commands.NewBuilder(ctx, bo)
}

func buildOptions(a *latestV1.Artifact, runMode config.RunMode) *options.BuildOptions {
	workingDirectory := filepath.Join(a.Workspace, a.KoArtifact.Dir)
	return &options.BuildOptions{
		BaseImage: a.KoArtifact.BaseImage,
		BuildConfigs: map[string]build.Config{
			a.Workspace: {
				ID:      a.ImageName,
				Dir:     workingDirectory,
				Env:     a.KoArtifact.Env,
				Flags:   a.KoArtifact.Flags,
				Ldflags: a.KoArtifact.Ldflags,
				Main:    a.KoArtifact.Main,
			},
		},
		ConcurrentBuilds:     1,
		DisableOptimizations: runMode == config.RunModes.Debug,
		Labels:               labels(a),
		Platform:             strings.Join(a.KoArtifact.Platforms, ","),
		UserAgent:            version.UserAgentWithClient(),
		WorkingDirectory:     workingDirectory,
	}
}

func labels(a *latestV1.Artifact) []string {
	labels := []string{}
	for k, v := range a.KoArtifact.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}
	return labels
}
