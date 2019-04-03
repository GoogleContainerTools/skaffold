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

package local

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/plugin/builders/jib"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func (b *Builder) buildJibGradle(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	builder := jib.NewGradleBuilder()
	builder.LocalBuild = b.cfg
	builder.LocalDocker = b.localDocker
	builder.KubeContext = b.kubeContext
	builder.PushImages = b.pushImages
	builder.PluginMode = false

	builder.Init(&runcontext.RunContext{
		Opts: &config.SkaffoldOptions{
			SkipTests: b.skipTests,
		},
		Cfg: &latest.Pipeline{
			Build: latest.BuildConfig{
				ExecutionEnvironment: &latest.ExecutionEnvironment{},
			},
		},
	})
	return builder.BuildArtifact(ctx, out, a, tag)
}
