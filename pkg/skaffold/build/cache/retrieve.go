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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	timeutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/time"
)

func (c *cache) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latestV1.Artifact, buildAndTest BuildAndTestFn) ([]graph.Artifact, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	start := time.Now()

	output.Default.Fprintln(out, "Checking cache...")
	ctx, endTrace := instrumentation.StartTrace(ctx, "Build_CheckBuildCache")
	defer endTrace()

	lookup := make(chan []cacheDetails)
	go func() { lookup <- c.lookupArtifacts(ctx, tags, artifacts) }()

	var results []cacheDetails
	select {
	case <-ctx.Done():
		return nil, context.Canceled
	case results = <-lookup:
	}

	hashByName := make(map[string]string)
	var needToBuild []*latestV1.Artifact
	var alreadyBuilt []graph.Artifact
	for i, artifact := range artifacts {
		eventV2.CacheCheckInProgress(artifact.ImageName)
		out, ctx := output.WithEventContext(ctx, out, constants.Build, artifact.ImageName)
		output.Default.Fprintf(out, " - %s: ", artifact.ImageName)

		result := results[i]
		switch result := result.(type) {
		case failed:
			output.Red.Fprintln(out, "Error checking cache.")
			endTrace(instrumentation.TraceEndError(result.err))
			return nil, result.err

		case needsBuilding:
			eventV2.CacheCheckMiss(artifact.ImageName)
			output.Yellow.Fprintln(out, "Not found. Building")
			hashByName[artifact.ImageName] = result.Hash()
			needToBuild = append(needToBuild, artifact)
			continue

		case needsTagging:
			eventV2.CacheCheckHit(artifact.ImageName)
			output.Green.Fprintln(out, "Found. Tagging")
			if err := result.Tag(ctx, c); err != nil {
				endTrace(instrumentation.TraceEndError(err))
				return nil, fmt.Errorf("tagging image: %w", err)
			}

		case needsPushing:
			eventV2.CacheCheckHit(artifact.ImageName)
			output.Green.Fprintln(out, "Found. Pushing")
			if err := result.Push(ctx, out, c); err != nil {
				endTrace(instrumentation.TraceEndError(err))

				return nil, fmt.Errorf("%s: %w", sErrors.PushImageErr, err)
			}

		default:
			eventV2.CacheCheckHit(artifact.ImageName)
			isLocal, err := c.isLocalImage(artifact.ImageName)
			if err != nil {
				endTrace(instrumentation.TraceEndError(err))
				return nil, err
			}
			if isLocal {
				output.Green.Fprintln(out, "Found Locally")
			} else {
				output.Green.Fprintln(out, "Found Remotely")
			}
		}

		// Image is already built
		c.cacheMutex.RLock()
		entry := c.artifactCache[result.Hash()]
		c.cacheMutex.RUnlock()
		tag := tags[artifact.ImageName]

		var uniqueTag string
		isLocal, err := c.isLocalImage(artifact.ImageName)
		if err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}
		if isLocal && artifact.BuildahArtifact == nil {
			var err error
			uniqueTag, err = build.TagWithImageID(ctx, tag, entry.ID, c.client)
			if err != nil {
				endTrace(instrumentation.TraceEndError(err))
				return nil, err
			}
		} else if isLocal && artifact.BuildahArtifact != nil {

		} else {
			uniqueTag = build.TagWithDigest(tag, entry.Digest)
		}
		c.artifactStore.Record(artifact, uniqueTag)
		alreadyBuilt = append(alreadyBuilt, graph.Artifact{
			ImageName: artifact.ImageName,
			Tag:       uniqueTag,
		})
	}

	log.Entry(ctx).Infoln("Cache check completed in", timeutil.Humanize(time.Since(start)))

	bRes, err := buildAndTest(ctx, out, tags, needToBuild)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, err
	}

	if err := c.addArtifacts(ctx, bRes, hashByName); err != nil {
		log.Entry(ctx).Warnf("error adding artifacts to cache; caching may not work as expected: %v", err)
		return append(bRes, alreadyBuilt...), nil
	}

	if err := saveArtifactCache(c.cacheFile, c.artifactCache); err != nil {
		log.Entry(ctx).Warnf("error saving cache file; caching may not work as expected: %v", err)
		return append(bRes, alreadyBuilt...), nil
	}

	return maintainArtifactOrder(append(bRes, alreadyBuilt...), artifacts), err
}

func maintainArtifactOrder(built []graph.Artifact, artifacts []*latestV1.Artifact) []graph.Artifact {
	byName := make(map[string]graph.Artifact)
	for _, build := range built {
		byName[build.ImageName] = build
	}

	var ordered []graph.Artifact

	for _, artifact := range artifacts {
		ordered = append(ordered, byName[artifact.ImageName])
	}

	return ordered
}

func (c *cache) addArtifacts(ctx context.Context, bRes []graph.Artifact, hashByName map[string]string) error {
	for _, a := range bRes {
		entry := ImageDetails{}
		isLocal, err := c.isLocalImage(a.ImageName)
		if err != nil {
			return err
		}
		if isLocal {
			imageID, err := c.client.ImageID(ctx, a.Tag)
			if err != nil {
				return err
			}

			if imageID != "" {
				entry.ID = imageID
			}
		} else {
			ref, err := docker.ParseReference(a.Tag)
			if err != nil {
				return fmt.Errorf("parsing reference %q: %w", a.Tag, err)
			}
			entry.Digest = ref.Digest
		}
		c.cacheMutex.Lock()
		c.artifactCache[hashByName[a.ImageName]] = entry
		c.cacheMutex.Unlock()
	}
	return nil
}
