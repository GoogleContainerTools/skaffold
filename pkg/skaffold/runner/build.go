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
	"runtime"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	eventV2 "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tag"
	timeutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/time"
)

func NewBuilder(builder build.Builder, tagger tag.Tagger, platforms platform.Resolver, cache cache.Cache, runCtx *runcontext.RunContext) *Builder {
	return &Builder{
		Builder:   builder,
		tagger:    tagger,
		platforms: platforms,
		cache:     cache,
		runCtx:    runCtx,
	}
}

type Builder struct {
	Builder   build.Builder
	tagger    tag.Tagger
	platforms platform.Resolver
	cache     cache.Cache
	Builds    []graph.Artifact

	hasBuilt bool
	runCtx   *runcontext.RunContext
}

// GetBuilds returns the builds value.
func (r *Builder) GetBuilds() []graph.Artifact {
	return r.Builds
}

// Build builds a list of artifacts.
func (r *Builder) Build(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]graph.Artifact, error) {
	eventV2.TaskInProgress(constants.Build, "Build containers")
	out, ctx = output.WithEventContext(ctx, out, constants.Build, constants.SubtaskIDNone)

	// Use tags directly from the Kubernetes manifests.
	if r.runCtx.DigestSource() == constants.NoneDigestSource {
		return []graph.Artifact{}, nil
	}

	if err := CheckWorkspaces(artifacts); err != nil {
		eventV2.TaskFailed(constants.Build, err)
		return nil, err
	}

	tags, err := r.imageTags(ctx, out, artifacts)
	if err != nil {
		eventV2.TaskFailed(constants.Build, err)
		return nil, err
	}

	// In dry-run mode or with --digest-source set to 'remote' or 'tag' in render, we don't build anything, just return the tag for each artifact.
	switch {
	case r.runCtx.DryRun():
		output.Yellow.Fprintln(out, "Skipping build phase since --dry-run=true")
		return artifactsWithTags(tags, artifacts), nil
	case r.runCtx.RenderOnly() && r.runCtx.DigestSource() == constants.RemoteDigestSource:
		output.Yellow.Fprintln(out, "Skipping build phase since --digest-source=remote")
		return artifactsWithTags(tags, artifacts), nil
	case r.runCtx.RenderOnly() && r.runCtx.DigestSource() == constants.TagDigestSource:
		output.Yellow.Fprintln(out, "Skipping build phase since --digest-source=tag")
		return artifactsWithTags(tags, artifacts), nil
	default:
	}

	bRes, err := r.cache.Build(ctx, out, tags, artifacts, r.platforms, func(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, platforms platform.Resolver) ([]graph.Artifact, error) {
		if len(artifacts) == 0 {
			return nil, nil
		}

		r.hasBuilt = true
		if err != nil {
			return nil, err
		}
		bRes, err := r.Builder.Build(ctx, out, tags, platforms, artifacts)
		if err != nil {
			return nil, err
		}

		return bRes, nil
	})
	if err != nil {
		eventV2.TaskFailed(constants.Build, err)
		return nil, err
	}

	// Make sure all artifacts are redeployed. Not only those that were just built.
	r.Builds = build.MergeWithPreviousBuilds(bRes, r.Builds)

	eventV2.TaskSucceeded(constants.Build)
	return bRes, nil
}

// HasBuilt returns true if this runner has built something.
func (r *Builder) HasBuilt() bool {
	return r.hasBuilt
}

func artifactsWithTags(tags tag.ImageTags, artifacts []*latest.Artifact) []graph.Artifact {
	var bRes []graph.Artifact
	for _, artifact := range artifacts {
		bRes = append(bRes, graph.Artifact{
			ImageName:   artifact.ImageName,
			Tag:         tags[artifact.ImageName],
			RuntimeType: artifact.RuntimeType,
		})
	}

	return bRes
}

type tagErr struct {
	tag string
	err error
}

// ApplyDefaultRepo applies the default repo to a given image tag.
func (r *Builder) ApplyDefaultRepo(tag string) (string, error) {
	return deployutil.ApplyDefaultRepo(r.runCtx.GlobalConfig(), r.runCtx.DefaultRepo(), tag)
}

// imageTags generates tags for a list of artifacts
func (r *Builder) imageTags(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) (tag.ImageTags, error) {
	start := time.Now()
	maxWorkers := runtime.GOMAXPROCS(0)

	if len(artifacts) > 0 {
		output.Default.Fprintln(out, "Generating tags...")
	} else {
		output.Default.Fprintln(out, "No tags generated")
	}

	tagErrs := make([]chan tagErr, len(artifacts))

	// Use a weighted semaphore as a counting semaphore to limit the number of simultaneous taggers.
	sem := semaphore.NewWeighted(int64(maxWorkers))
	for i := range artifacts {
		tagErrs[i] = make(chan tagErr, 1)

		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, err
		}

		i := i
		go func() {
			defer sem.Release(1)
			_tag, err := tag.GenerateFullyQualifiedImageName(ctx, r.tagger, *artifacts[i])
			tagErrs[i] <- tagErr{tag: _tag, err: err}
		}()
	}

	imageTags := make(tag.ImageTags, len(artifacts))
	showWarning := false

	for i, artifact := range artifacts {
		imageName := artifact.ImageName
		out, ctx := output.WithEventContext(ctx, out, constants.Build, imageName)
		output.Default.Fprintf(out, " - %s -> ", imageName)

		select {
		case <-ctx.Done():
			return nil, context.Canceled

		case t := <-tagErrs[i]:
			if t.err != nil {
				log.Entry(ctx).Debug(t.err)
				log.Entry(ctx).Debug("Using a fall-back tagger")

				fallbackTag, err := tag.GenerateFullyQualifiedImageName(ctx, &tag.ChecksumTagger{}, *artifact)
				if err != nil {
					return nil, fmt.Errorf("generating checksum as fall-back tag for %q: %w", imageName, err)
				}

				t.tag = fallbackTag
				showWarning = true
			}

			_tag, err := r.ApplyDefaultRepo(t.tag)
			if err != nil {
				return nil, err
			}

			fmt.Fprintln(out, _tag)
			imageTags[imageName] = _tag
		}
	}

	if showWarning {
		output.Yellow.Fprintln(out, "Some taggers failed. Rerun with -vdebug for errors.")
	}

	log.Entry(ctx).Infoln("Tags generated in", timeutil.Humanize(time.Since(start)))
	return imageTags, nil
}

func CheckWorkspaces(artifacts []*latest.Artifact) error {
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
