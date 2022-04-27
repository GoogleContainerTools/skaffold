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

package runner

import (
	"context"
	"io"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/renderer"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	timeutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/time"
)

// WithTimings creates a deployer that logs the duration of each phase.
func WithTimings(b build.Builder, t test.Tester, r renderer.Renderer, d deploy.Deployer, cacheArtifacts bool) (build.Builder, test.Tester, renderer.Renderer, deploy.Deployer) {
	w := withTimings{
		Builder:        b,
		Tester:         t,
		Renderer:       r,
		Deployer:       d,
		cacheArtifacts: cacheArtifacts,
	}

	return w, w, w, w
}

type withTimings struct {
	build.Builder
	test.Tester
	renderer.Renderer
	deploy.Deployer
	cacheArtifacts bool
}

func (w withTimings) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, platforms platform.Resolver, artifacts []*latest.Artifact) ([]graph.Artifact, error) {
	if len(artifacts) == 0 && w.cacheArtifacts {
		return nil, nil
	}
	start := time.Now()
	output.Default.Fprintln(out, "Starting build...")

	bRes, err := w.Builder.Build(ctx, out, tags, platforms, artifacts)
	if err != nil {
		return nil, err
	}
	log.Entry(ctx).Infoln("Build completed in", timeutil.Humanize(time.Since(start)))
	return bRes, nil
}

func (w withTimings) Test(ctx context.Context, out io.Writer, builds []graph.Artifact) error {
	start := time.Now()
	output.Default.Fprintln(out, "Starting test...")

	err := w.Tester.Test(ctx, out, builds)
	if err != nil {
		return err
	}
	log.Entry(ctx).Infoln("Test completed in", timeutil.Humanize(time.Since(start)))
	return nil
}

func (w withTimings) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, filepath string) (manifest.ManifestList, error) {
	start := time.Now()
	output.Default.Fprintln(out, "Starting render...")

	manifestsLists, err := w.Renderer.Render(ctx, out, builds, offline, filepath)
	if err != nil {
		return nil, err
	}
	log.Entry(context.TODO()).Infoln("Render completed in", timeutil.Humanize(time.Since(start)))
	return manifestsLists, nil
}

func (w withTimings) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact, l manifest.ManifestList) error {
	start := time.Now()
	output.Default.Fprintln(out, "Starting deploy...")

	err := w.Deployer.Deploy(ctx, out, builds, l)
	if err != nil {
		return err
	}
	log.Entry(ctx).Infoln("Deploy completed in", timeutil.Humanize(time.Since(start)))
	return err
}

func (w withTimings) Cleanup(ctx context.Context, out io.Writer, dryRun bool, list manifest.ManifestList) error {
	start := time.Now()
	output.Default.Fprintln(out, "Cleaning up...")

	err := w.Deployer.Cleanup(ctx, out, dryRun, nil)
	if err != nil {
		return err
	}
	log.Entry(ctx).Infoln("Cleanup completed in", timeutil.Humanize(time.Since(start)))
	return nil
}

func (w withTimings) Prune(ctx context.Context, out io.Writer) error {
	start := time.Now()
	output.Default.Fprintln(out, "Pruning images...")

	err := w.Builder.Prune(ctx, out)
	if err != nil {
		return err
	}
	log.Entry(ctx).Infoln("Image prune completed in", timeutil.Humanize(time.Since(start)))
	return nil
}
