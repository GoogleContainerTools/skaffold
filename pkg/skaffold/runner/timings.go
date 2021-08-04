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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// WithTimings creates a deployer that logs the duration of each phase.
func WithTimings(b build.Builder, t test.Tester, d deploy.Deployer, cacheArtifacts bool) (build.Builder, test.Tester, deploy.Deployer) {
	w := withTimings{
		Builder:        b,
		Tester:         t,
		Deployer:       d,
		cacheArtifacts: cacheArtifacts,
	}

	return w, w, w
}

type withTimings struct {
	build.Builder
	test.Tester
	deploy.Deployer
	cacheArtifacts bool
}

func (w withTimings) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latestV1.Artifact) ([]graph.Artifact, error) {
	if len(artifacts) == 0 && w.cacheArtifacts {
		return nil, nil
	}
	start := time.Now()
	output.Default.Fprintln(out, "Starting build...")

	bRes, err := w.Builder.Build(ctx, out, tags, artifacts)
	if err != nil {
		return nil, err
	}
	logrus.Infoln("Build completed in", util.ShowHumanizeTime(time.Since(start)))
	return bRes, nil
}

func (w withTimings) Test(ctx context.Context, out io.Writer, builds []graph.Artifact) error {
	start := time.Now()
	output.Default.Fprintln(out, "Starting test...")

	err := w.Tester.Test(ctx, out, builds)
	if err != nil {
		return err
	}
	logrus.Infoln("Test completed in", util.ShowHumanizeTime(time.Since(start)))
	return nil
}

func (w withTimings) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact) error {
	start := time.Now()
	output.Default.Fprintln(out, "Starting deploy...")

	err := w.Deployer.Deploy(ctx, out, builds)
	if err != nil {
		return err
	}
	logrus.Infoln("Deploy completed in", util.ShowHumanizeTime(time.Since(start)))
	return err
}

func (w withTimings) Cleanup(ctx context.Context, out io.Writer) error {
	start := time.Now()
	output.Default.Fprintln(out, "Cleaning up...")

	err := w.Deployer.Cleanup(ctx, out)
	if err != nil {
		return err
	}
	logrus.Infoln("Cleanup completed in", util.ShowHumanizeTime(time.Since(start)))
	return nil
}

func (w withTimings) Prune(ctx context.Context, out io.Writer) error {
	start := time.Now()
	output.Default.Fprintln(out, "Pruning images...")

	err := w.Builder.Prune(ctx, out)
	if err != nil {
		return err
	}
	logrus.Infoln("Image prune completed in", util.ShowHumanizeTime(time.Since(start)))
	return nil
}
