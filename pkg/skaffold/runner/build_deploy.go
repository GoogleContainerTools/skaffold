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
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// BuildAndTest builds and tests a list of artifacts.
func (r *SkaffoldRunner) BuildAndTest(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	tags, err := r.imageTags(ctx, out, artifacts)
	if err != nil {
		return nil, err
	}

	// In dry-run mode, we don't build anything, just return the tag for each artifact.
	if r.runCtx.Opts.DryRun {
		var bRes []build.Artifact

		for _, artifact := range artifacts {
			bRes = append(bRes, build.Artifact{
				ImageName: artifact.ImageName,
				Tag:       tags[artifact.ImageName],
			})
		}

		return bRes, nil
	}

	bRes, err := r.cache.Build(ctx, out, tags, artifacts, func(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
		if len(artifacts) == 0 {
			return nil, nil
		}

		r.hasBuilt = true

		bRes, err := r.builder.Build(ctx, out, tags, artifacts)
		if err != nil {
			return nil, err
		}

		if !r.runCtx.Opts.SkipTests {
			if err = r.tester.Test(ctx, out, bRes); err != nil {
				return nil, err
			}
		}

		return bRes, nil
	})
	if err != nil {
		return nil, err
	}

	// Update which images are logged.
	for _, build := range bRes {
		r.podSelector.Add(build.Tag)
	}

	// Make sure all artifacts are redeployed. Not only those that were just built.
	r.builds = build.MergeWithPreviousBuilds(bRes, r.builds)

	return bRes, nil
}

// DeployAndLog deploys a list of already built artifacts and optionally show the logs.
func (r *SkaffoldRunner) DeployAndLog(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
	if !r.runCtx.Opts.Tail && !r.runCtx.Opts.PortForward.Enabled {
		return r.Deploy(ctx, out, artifacts)
	}

	var imageNames []string
	for _, artifact := range artifacts {
		imageNames = append(imageNames, artifact.ImageName)
		r.podSelector.Add(artifact.Tag)
	}

	r.createLoggerForImages(out, imageNames)
	defer r.logger.Stop()

	r.createForwarder(out)
	defer r.forwarderManager.Stop()

	// Logs should be retrieved up to just before the deploy
	r.logger.SetSince(time.Now())

	// First deploy
	if err := r.Deploy(ctx, out, artifacts); err != nil {
		return err
	}

	if r.runCtx.Opts.PortForward.Enabled {
		if err := r.forwarderManager.Start(ctx); err != nil {
			logrus.Warnln("Error starting port forwarding:", err)
		}
	}

	// Start printing the logs after deploy is finished
	if r.runCtx.Opts.Tail {
		if err := r.logger.Start(ctx); err != nil {
			return fmt.Errorf("starting logger: %w", err)
		}
	}

	<-ctx.Done()

	return nil
}

type tagErr struct {
	tag string
	err error
}

// imageTags generates tags for a list of artifacts
func (r *SkaffoldRunner) imageTags(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) (tag.ImageTags, error) {
	start := time.Now()
	color.Default.Fprintln(out, "Generating tags...")

	defaultRepo, err := config.GetDefaultRepo(r.runCtx.Opts.GlobalConfig, r.runCtx.Opts.DefaultRepo.Value())
	if err != nil {
		return nil, fmt.Errorf("getting default repo: %w", err)
	}

	tagErrs := make([]chan tagErr, len(artifacts))

	for i := range artifacts {
		tagErrs[i] = make(chan tagErr, 1)

		i := i
		go func() {
			tag, err := r.tagger.GenerateFullyQualifiedImageName(artifacts[i].Workspace, artifacts[i].ImageName)
			tagErrs[i] <- tagErr{tag: tag, err: err}
		}()
	}

	imageTags := make(tag.ImageTags, len(artifacts))
	showWarning := false

	for i, artifact := range artifacts {
		imageName := artifact.ImageName
		color.Default.Fprintf(out, " - %s -> ", imageName)

		select {
		case <-ctx.Done():
			return nil, context.Canceled

		case t := <-tagErrs[i]:
			err := t.err

			if err != nil {
				logrus.Debugln(err)
				logrus.Debugln("Using a fall-back tagger")

				fallbackTagger := &tag.ChecksumTagger{}
				t.tag, err = fallbackTagger.GenerateFullyQualifiedImageName(artifact.Workspace, imageName)
				if err != nil {
					return nil, fmt.Errorf("generating checksum as fall-back tag for %q: %w", imageName, err)
				}

				showWarning = true
			}

			tag, err := docker.SubstituteDefaultRepoIntoImage(defaultRepo, t.tag)
			if err != nil {
				return nil, fmt.Errorf("applying default repo to %q: %w", t.tag, t.err)
			}

			fmt.Fprintln(out, tag)
			imageTags[imageName] = tag
		}
	}

	if showWarning {
		color.Yellow.Fprintln(out, "Some taggers failed. Rerun with -vdebug for errors.")
	}

	logrus.Infoln("Tags generated in", time.Since(start))
	return imageTags, nil
}
