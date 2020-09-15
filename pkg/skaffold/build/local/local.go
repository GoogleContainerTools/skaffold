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
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/dustin/go-humanize"
	"io"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/custom"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

const (
	usageRetries       = 5
	usageRetryInterval = 500 * time.Millisecond
	//TODO
	pruneLimit = 1
)

// Build runs a docker build on the host and tags the resulting image with
// its checksum. It streams build progress to the writer argument.
func (b *Builder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if b.localCluster {
		color.Default.Fprintf(out, "Found [%s] context, using local docker daemon.\n", b.kubeContext)
	}
	defer b.localDocker.Close()

	b.cleanupOldImages(ctx, pruneLimit, out, artifacts)

	builder := build.WithLogFile(b.buildArtifact, b.muted)
	return build.InParallel(ctx, out, tags, artifacts, builder, *b.cfg.Concurrency)
}

func (b *Builder) listUniqImages(ctx context.Context, name string) ([]types.ImageSummary, error) {
	imgs, err := b.localDocker.ImageList(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(imgs) < 2 {
		return imgs, nil
	}

	sort.Slice(imgs, func(i, j int) bool {
		return imgs[i].Created < imgs[j].Created
	})

	// keep only uniq images (an image can have more than one tag)
	uqIdx := 0
	for i, img := range imgs {
		if imgs[i].ID != imgs[uqIdx].ID {
			uqIdx += 1
			imgs[uqIdx] = img
		}
	}
	return imgs[:uqIdx+1], nil
}

func (b *Builder) cleanupOldImages(ctx context.Context, limit int, out io.Writer, artifacts []*latest.Artifact) {

	toPrune := b.collectImagesToPrune(ctx, limit, artifacts)

	if len(toPrune) > 0 {
		go func() {
			logrus.Debugf("Going to prune: %v", toPrune)

			beforeDu, err := b.diskUsage(ctx)
			if err != nil {
				logrus.Warnf("Failed to get docker usage info: %v", err)
			}

			err = b.localDocker.Prune(ctx, out, toPrune, b.pruneChildren)
			if err != nil {
				logrus.Warnf("Failed to prune: %v", err)
				return
			}
			// do not print usage report, if initial 'du' failed
			if beforeDu > 0 {
				afterDu, err := b.diskUsage(ctx)
				if err != nil {
					logrus.Warnf("Failed to get docker usage info: %v", err)
					return
				}
				logrus.Infof("%d image(s) pruned. Gained disk space: %s %d %d",
					len(toPrune), humanize.Bytes(uint64(afterDu-beforeDu)), beforeDu, afterDu)
			}
		}()
	}
}

func (b *Builder) collectImagesToPrune(ctx context.Context, limit int, artifacts []*latest.Artifact) []string {
	imgNameCount := make(map[string]int)
	for _, a := range artifacts {
		imgNameCount[a.ImageName]++
	}
	rt := make([]string, 0)
	for _, a := range artifacts {
		imgs, err := b.listUniqImages(ctx, a.ImageName)
		if err != nil {
			logrus.Warnf("failed to list images: %v", err)
			continue
		}
		limForImage := limit * imgNameCount[a.ImageName]
		for i := limForImage; i < len(imgs); i++ {
			rt = append(rt, imgs[i].ID)
		}
	}
	return rt
}

func (b *Builder) diskUsage(ctx context.Context) (int64, error) {
	for retry := 0; retry < usageRetries-1; retry++ {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		usage, err := b.localDocker.RawClient().DiskUsage(ctx)
		if err == nil {
			return usage.LayersSize, nil
		}
		logrus.Debugf("[%d of %d] failed to get disk usage: %v. Will retry in %v",
			retry, usageRetries, err, usageRetryInterval)

		time.Sleep(usageRetryInterval)
	}

	usage, err := b.localDocker.RawClient().DiskUsage(ctx)
	if err == nil {
		return usage.LayersSize, nil
	}
	logrus.Debugf("Failed to get usage after %d retries : %v. giving up", usageRetries, err)
	return 0, err
}

func (b *Builder) buildArtifact(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	digestOrImageID, err := b.runBuildForArtifact(ctx, out, a, tag)
	if err != nil {
		return "", err
	}

	if b.pushImages {
		// only track images for pruning when building with docker
		// if we're pushing a bazel image, it was built directly to the registry
		if a.DockerArtifact != nil {
			imageID, err := b.getImageIDForTag(ctx, tag)
			if err != nil {
				logrus.Warnf("unable to inspect image: built images may not be cleaned up correctly by skaffold")
			}
			if imageID != "" {
				b.builtImages = append(b.builtImages, imageID)
			}
		}

		digest := digestOrImageID
		return build.TagWithDigest(tag, digest), nil
	}

	imageID := digestOrImageID
	b.builtImages = append(b.builtImages, imageID)
	return build.TagWithImageID(ctx, tag, imageID, b.localDocker)
}

func (b *Builder) runBuildForArtifact(ctx context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	if !b.pushImages {
		// All of the builders will rely on a local Docker:
		// + Either to build the image,
		// + Or to docker load it.
		// Let's fail fast if Docker is not available
		if _, err := b.localDocker.ServerVersion(ctx); err != nil {
			return "", err
		}
	}

	switch {
	case a.DockerArtifact != nil:
		return b.buildDocker(ctx, out, a, tag, b.mode)

	case a.BazelArtifact != nil:
		return bazel.NewArtifactBuilder(b.localDocker, b.insecureRegistries, b.pushImages).Build(ctx, out, a, tag)

	case a.JibArtifact != nil:
		return jib.NewArtifactBuilder(b.localDocker, b.insecureRegistries, b.pushImages, b.skipTests).Build(ctx, out, a, tag)

	case a.CustomArtifact != nil:
		return custom.NewArtifactBuilder(b.localDocker, b.insecureRegistries, b.pushImages, b.retrieveExtraEnv()).Build(ctx, out, a, tag)

	case a.BuildpackArtifact != nil:
		return buildpacks.NewArtifactBuilder(b.localDocker, b.pushImages, b.mode).Build(ctx, out, a, tag)

	default:
		return "", fmt.Errorf("unexpected type %q for local artifact:\n%s", misc.ArtifactType(a), misc.FormatArtifact(a))
	}
}

func (b *Builder) getImageIDForTag(ctx context.Context, tag string) (string, error) {
	insp, _, err := b.localDocker.ImageInspectWithRaw(ctx, tag)
	if err != nil {
		return "", err
	}
	return insp.ID, nil
}
