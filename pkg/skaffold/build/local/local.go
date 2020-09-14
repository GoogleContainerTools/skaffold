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
	"io"
	"sort"

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

// Build runs a docker build on the host and tags the resulting image with
// its checksum. It streams build progress to the writer argument.
func (b *Builder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	if b.localCluster {
		color.Default.Fprintf(out, "Found [%s] context, using local docker daemon.\n", b.kubeContext)
	}
	defer b.localDocker.Close()

	const pruneLimit = 2

	b.cleanupPrevImages(ctx, pruneLimit, out, artifacts)

	builder := build.WithLogFile(b.buildArtifact, b.muted)
	return build.InParallel(ctx, out, tags, artifacts, builder, *b.cfg.Concurrency)
}

func (b *Builder) listUniqImages(ctx context.Context, name string, limit int) ([]types.ImageSummary, error) {
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
	for i := 1; i < len(imgs) && uqIdx < limit; i++ {
		if imgs[i].ID != imgs[uqIdx].ID {
			uqIdx += 1
			imgs[uqIdx] = imgs[i]
		}
	}
	logrus.Debugf("%d of %d ids for %s are uniq", uqIdx+1, len(imgs), name)
	return imgs[:uqIdx+1], nil
}

func (b *Builder) cleanupPrevImages(ctx context.Context, limit int, out io.Writer, artifacts []*latest.Artifact) {

	imgNameCount := make(map[string]int)
	for _, a := range artifacts {
		imgNameCount[a.ImageName]++
	}

	for _, a := range artifacts {
		name := a.ImageName
		imgsToKeep := limit * imgNameCount[name]
		imgs, err := b.listUniqImages(ctx, name, imgsToKeep)
		if err != nil {
			logrus.Warnf("failed to list images: %v", err)
			continue
		}
		imgsToPrune := len(imgs) - imgsToKeep
		if imgsToPrune > 0 {
			logrus.Debugf("need to prune %v %qs", imgsToPrune, name)
			go func() {
				idsToPrune := make([]string, imgsToPrune)
				for i, img := range imgs[imgsToKeep:] {
					idsToPrune[i] = img.ID
				}
				logrus.Debugf("going to prune: %v", idsToPrune)
				err := b.localDocker.Prune(ctx, out, idsToPrune, b.pruneChildren)
				if err != nil {
					logrus.Warnf("failed to prune: %v", err)
				}
			}()
		} else {
			//TODO: remove log
			logrus.Debugf("no need to prune %v", name)
		}
	}
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
