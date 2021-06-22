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
	"strings"

	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands"
	"github.com/google/ko/pkg/commands/options"

	// latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

func (b *Builder) newKoBuilder(ctx context.Context, a *latestV1.Artifact) (build.Interface, error) {
	bo := buildOptions(a.KoArtifact.BaseImage, a.KoArtifact.Platforms, a.Workspace)
	return commands.NewBuilder(ctx, bo)
}

func buildOptions(baseImage string, platforms []string, workspace string) *options.BuildOptions {
	return &options.BuildOptions{
		BaseImage:        baseImage,
		ConcurrentBuilds: 1,
		Platform:         strings.Join(platforms, ","),
		UserAgent:        version.UserAgentWithClient(),
		WorkingDirectory: workspace,
	}
}
