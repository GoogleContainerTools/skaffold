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
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// Build builds a list of artifacts.
func (r *SkaffoldRunner) Build(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	// Use tags directly from the Kubernetes manifests.
	if r.runCtx.DigestSource() == noneDigestSource {
		return []build.Artifact{}, nil
	}

	if err := checkWorkspaces(artifacts); err != nil {
		return nil, err
	}

	tags, err := r.imageTags(ctx, out, artifacts)
	if err != nil {
		return nil, err
	}

	// In dry-run mode or with --digest-source  set to 'remote', we don't build anything, just return the tag for each artifact.
	if r.runCtx.DryRun() || (r.runCtx.DigestSource() == remoteDigestSource) {
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

		return bRes, nil
	})
	if err != nil {
		return nil, err
	}

	// Update which images are logged.
	r.addTagsToPodSelector(bRes)

	// Make sure all artifacts are redeployed. Not only those that were just built.
	r.builds = build.MergeWithPreviousBuilds(bRes, r.builds)

	return bRes, nil
}

// Test tests a list of already built artifacts.
func (r *SkaffoldRunner) Test(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
	if err := r.tester.Test(ctx, out, artifacts); err != nil {
		return err
	}

	return nil
}

// DeployAndLog deploys a list of already built artifacts and optionally show the logs.
func (r *SkaffoldRunner) DeployAndLog(ctx context.Context, out io.Writer, artifacts []build.Artifact) error {
	// Update which images are logged.
	r.addTagsToPodSelector(artifacts)

	logger := r.createLogger(out, artifacts)
	defer logger.Stop()

	// Logs should be retrieved up to just before the deploy
	logger.SetSince(time.Now())

	// First deploy
	if err := r.Deploy(ctx, out, artifacts); err != nil {
		return err
	}

	forwarderManager := r.createForwarder(out)
	defer forwarderManager.Stop()

	if err := forwarderManager.Start(ctx); err != nil {
		logrus.Warnln("Error starting port forwarding:", err)
	}

	// Start printing the logs after deploy is finished
	if err := logger.Start(ctx); err != nil {
		return fmt.Errorf("starting logger: %w", err)
	}

	if r.runCtx.Tail() || r.runCtx.PortForward() {
		color.Yellow.Fprintln(out, "Press Ctrl+C to exit")
		<-ctx.Done()
	}

	return nil
}

// Update which images are logged.
func (r *SkaffoldRunner) addTagsToPodSelector(artifacts []build.Artifact) {
	for _, artifact := range artifacts {
		r.podSelector.Add(artifact.Tag)
	}
}

type tagErr struct {
	tag string
	err error
}

// ApplyDefaultRepo applies the default repo to a given image tag.
func (r *SkaffoldRunner) ApplyDefaultRepo(tag string) (string, error) {
	return deployutil.ApplyDefaultRepo(r.runCtx.GlobalConfig(), r.runCtx.DefaultRepo(), tag)
}

// imageTags generates tags for a list of artifacts
func (r *SkaffoldRunner) imageTags(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) (tag.ImageTags, error) {
	start := time.Now()
	color.Default.Fprintln(out, "Generating tags...")

	tagErrs := make([]chan tagErr, len(artifacts))

	for i := range artifacts {
		tagErrs[i] = make(chan tagErr, 1)

		i := i
		go func() {
			tag, err := tag.GenerateFullyQualifiedImageName(r.tagger, artifacts[i].Workspace, artifacts[i].ImageName)
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
			if t.err != nil {
				logrus.Debugln(t.err)
				logrus.Debugln("Using a fall-back tagger")

				fallbackTag, err := tag.GenerateFullyQualifiedImageName(&tag.ChecksumTagger{}, artifact.Workspace, imageName)
				if err != nil {
					return nil, fmt.Errorf("generating checksum as fall-back tag for %q: %w", imageName, err)
				}

				t.tag = fallbackTag
				showWarning = true
			}

			tag, err := r.ApplyDefaultRepo(t.tag)
			if err != nil {
				return nil, err
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

func checkWorkspaces(artifacts []*latest.Artifact) error {
	for _, a := range artifacts {
		if a.Workspace != "" {
			if info, err := os.Stat(a.Workspace); err != nil {
				// err could be permission-related
				if os.IsNotExist(err) {
					return fmt.Errorf("image %q context %q does not exist", a.ImageName, a.Workspace)
				}
				return fmt.Errorf("image %q context %q: %w", a.ImageName, a.Workspace, err)
			} else if !info.IsDir() {
				return fmt.Errorf("image %q context %q is not a directory", a.ImageName, a.Workspace)
			}
		}
	}
	return nil
}
