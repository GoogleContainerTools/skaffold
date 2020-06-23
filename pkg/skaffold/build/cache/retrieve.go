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
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func (c *cache) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact, buildAndTest BuildAndTestFn) ([]build.Artifact, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	start := time.Now()

	color.Default.Fprintln(out, "Checking cache...")

	lookup := make(chan []cacheDetails)
	go func() { lookup <- c.lookupArtifacts(ctx, tags, artifacts) }()

	var results []cacheDetails
	select {
	case <-ctx.Done():
		return nil, context.Canceled
	case results = <-lookup:
	}

	hashByName := make(map[string]string)
	var needToBuild []*latest.Artifact
	var alreadyBuilt []build.Artifact
	for i, artifact := range artifacts {
		color.Default.Fprintf(out, " - %s: ", artifact.ImageName)

		result := results[i]
		switch result := result.(type) {
		case failed:
			color.Red.Fprintln(out, "Error checking cache.")
			return nil, result.err

		case needsBuilding:
			color.Yellow.Fprintln(out, "Not found. Building")
			hashByName[artifact.ImageName] = result.Hash()
			needToBuild = append(needToBuild, artifact)
			continue

		case needsTagging:
			color.Green.Fprintln(out, "Found. Tagging")
			if err := result.Tag(ctx, c); err != nil {
				return nil, fmt.Errorf("tagging image: %w", err)
			}

		case needsPushing:
			color.Green.Fprintln(out, "Found. Pushing")
			if err := result.Push(ctx, out, c); err != nil {
				return nil, fmt.Errorf("%s: %w", sErrors.PushImageErrPrefix, err)
			}

		default:
			if c.imagesAreLocal {
				color.Green.Fprintln(out, "Found Locally")
			} else {
				color.Green.Fprintln(out, "Found Remotely")
			}
		}

		// Image is already built
		entry := c.artifactCache[result.Hash()]
		tag := tags[artifact.ImageName]

		var uniqueTag string
		if c.imagesAreLocal {
			var err error
			uniqueTag, err = build.TagWithImageID(ctx, tag, entry.ID, c.client)
			if err != nil {
				return nil, err
			}
		} else {
			uniqueTag = build.TagWithDigest(tag, entry.Digest)
		}

		alreadyBuilt = append(alreadyBuilt, build.Artifact{
			ImageName: artifact.ImageName,
			Tag:       uniqueTag,
		})
	}

	logrus.Infoln("Cache check complete in", time.Since(start))

	bRes, err := buildAndTest(ctx, out, tags, needToBuild)
	if err != nil {
		return nil, err
	}

	if err := c.addArtifacts(ctx, bRes, hashByName); err != nil {
		logrus.Warnf("error adding artifacts to cache; caching may not work as expected: %v", err)
		return append(bRes, alreadyBuilt...), nil
	}

	if err := saveArtifactCache(c.cacheFile, c.artifactCache); err != nil {
		logrus.Warnf("error saving cache file; caching may not work as expected: %v", err)
		return append(bRes, alreadyBuilt...), nil
	}

	return maintainArtifactOrder(append(bRes, alreadyBuilt...), artifacts), err
}

func maintainArtifactOrder(built []build.Artifact, artifacts []*latest.Artifact) []build.Artifact {
	byName := make(map[string]build.Artifact)
	for _, build := range built {
		byName[build.ImageName] = build
	}

	var ordered []build.Artifact

	for _, artifact := range artifacts {
		ordered = append(ordered, byName[artifact.ImageName])
	}

	return ordered
}

func (c *cache) addArtifacts(ctx context.Context, bRes []build.Artifact, hashByName map[string]string) error {
	for _, a := range bRes {
		entry := ImageDetails{}

		if !c.imagesAreLocal {
			ref, err := docker.ParseReference(a.Tag)
			if err != nil {
				return fmt.Errorf("parsing reference %q: %w", a.Tag, err)
			}

			entry.Digest = ref.Digest
		}

		imageID, err := c.client.ImageID(ctx, a.Tag)
		if err != nil {
			return err
		}

		if imageID != "" {
			entry.ID = imageID
		}

		c.artifactCache[hashByName[a.ImageName]] = entry
	}

	return nil
}
