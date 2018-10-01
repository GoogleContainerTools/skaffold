/*
Copyright 2018 The Skaffold Authors

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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	latest "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
)

// WithTimings creates a deployer that logs the duration of each phase.
func WithTimings(b build.Builder, d deploy.Deployer) (build.Builder, deploy.Deployer) {
	w := withTimings{
		Builder:  b,
		Deployer: d,
	}

	return w, w
}

type withTimings struct {
	build.Builder
	deploy.Deployer
}

func (w withTimings) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	start := time.Now()
	color.Default.Fprintln(out, "Starting build...")

	bRes, err := w.Builder.Build(ctx, out, tagger, artifacts)
	if err == nil {
		color.Default.Fprintln(out, "Build complete in", time.Since(start))
	}
	return bRes, err
}

func (w withTimings) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]deploy.Artifact, error) {
	start := time.Now()
	color.Default.Fprintln(out, "Starting deploy...")

	dRes, err := w.Deployer.Deploy(ctx, out, builds)
	if err == nil {
		color.Default.Fprintln(out, "Deploy complete in", time.Since(start))
	}
	return dRes, err
}

func (w withTimings) Labels() map[string]string {
	labels := map[string]string{}
	for k, v := range w.Builder.Labels() {
		labels[k] = v
	}
	for k, v := range w.Deployer.Labels() {
		labels[k] = v
	}
	return labels
}

func (w withTimings) Cleanup(ctx context.Context, out io.Writer) error {
	start := time.Now()
	color.Default.Fprintln(out, "Cleaning up...")

	err := w.Deployer.Cleanup(ctx, out)
	if err == nil {
		color.Default.Fprintln(out, "Cleanup complete in", time.Since(start))
	}
	return err
}
