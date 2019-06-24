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

package cache

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

func (c *cache) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildAndTest BuildAndTestFn) ([]build.Artifact, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	start := time.Now()

	color.Default.Fprintln(out, "Checking cache...")

	results := c.lookupArtifacts(ctx, tags, artifacts)

	hashByName := make(map[string]string)
	var (
		needToBuild []*latest.Artifact
		built       []build.Artifact
	)

	for i, artifact := range artifacts {
		color.Default.Fprintf(out, " - %s: ", artifact.ImageName)

		select {
		case <-ctx.Done():
			return nil, context.Canceled

		case d := <-results[i]:
			switch d := d.(type) {
			case failed:
				return nil, errors.Wrap(d.err, "checking cache")

			case needsBuilding:
				color.Red.Fprintln(out, "Not found. Building")
				hashByName[artifact.ImageName] = d.hash
				needToBuild = append(needToBuild, artifact)
				continue

			case needsTagging:
				color.Green.Fprintln(out, "Found. Tagging")
				if err := d.Tag(ctx, c); err != nil {
					return nil, errors.Wrap(err, "tagging image")
				}

			case needsPushing:
				color.Green.Fprintln(out, "Found. Pushing")
				if err := d.Push(ctx, out, c); err != nil {
					return nil, errors.Wrap(err, "pushing image")
				}

			default:
				color.Green.Fprintln(out, "Found")
			}

			dd := c.artifactCache[d.Hash()]
			var uniqueTag string
			if c.imagesAreLocal {
				// k8s doesn't recognize the imageID or any combination of the image name
				// suffixed with the imageID, as a valid image name.
				// So, the solution we chose is to create a tag, just for Skaffold, from
				// the imageID, and use that in the manifests.
				uniqueTag = artifact.ImageName + ":" + strings.TrimPrefix(dd.ID, "sha256:")
				if err := c.client.Tag(ctx, dd.ID, uniqueTag); err != nil {
					return nil, err
				}
			} else {
				uniqueTag = tags[artifact.ImageName] + "@" + dd.Digest
			}

			built = append(built, build.Artifact{
				ImageName: artifact.ImageName,
				Tag:       uniqueTag,
			})
		}
	}

	color.Default.Fprintln(out, "Cache check complete in", time.Since(start))

	bRes, err := buildAndTest(ctx, out, tags, needToBuild)
	if err != nil {
		return nil, errors.Wrap(err, "build failed")
	}

	if err := c.addArtifacts(ctx, bRes, hashByName); err != nil {
		return nil, errors.Wrap(err, "adding artifacts to cache")
	}

	if err := saveArtifactCache(c.cacheFile, c.artifactCache); err != nil {
		return nil, errors.Wrap(err, "saving cache")
	}

	return append(bRes, built...), err
}

func (c *cache) addArtifacts(ctx context.Context, bRes []build.Artifact, hashByName map[string]string) error {
	for _, a := range bRes {
		if c.imagesAreLocal {
			imageID, err := c.client.ImageID(ctx, a.Tag)
			if err != nil {
				return err
			}

			c.artifactCache[hashByName[a.ImageName]] = ImageDetails{
				ID: imageID,
			}
		} else {
			ref, err := docker.ParseReference(a.Tag)
			if err != nil {
				return errors.Wrapf(err, "parsing reference %s", a.Tag)
			}

			c.artifactCache[hashByName[a.ImageName]] = ImageDetails{
				Digest: ref.Digest,
			}
		}
	}

	return nil
}
