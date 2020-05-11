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
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func (c *cache) lookupArtifacts(ctx context.Context, tags tag.ImageTags, artifacts []*latest.Artifact) []cacheDetails {
	details := make([]cacheDetails, len(artifacts))

	var wg sync.WaitGroup
	for i := range artifacts {
		wg.Add(1)

		i := i
		go func() {
			details[i] = c.lookup(ctx, artifacts[i], tags[artifacts[i].ImageName])
			wg.Done()
		}()
	}
	wg.Wait()

	return details
}

func (c *cache) lookup(ctx context.Context, a *latest.Artifact, tag string) cacheDetails {
	hash, err := c.hashForArtifact(ctx, a)
	if err != nil {
		return failed{err: fmt.Errorf("getting hash for artifact %q: %s", a.ImageName, err)}
	}

	entry, cacheHit := c.artifactCache[hash]
	if !cacheHit {
		return needsBuilding{hash: hash}
	}

	if c.imagesAreLocal {
		return c.lookupLocal(ctx, hash, tag, entry)
	}
	return c.lookupRemote(ctx, hash, tag, entry)
}

func (c *cache) lookupLocal(ctx context.Context, hash, tag string, entry ImageDetails) cacheDetails {
	if entry.ID == "" {
		return needsBuilding{hash: hash}
	}

	// Check the imageID for the tag
	idForTag, err := c.client.ImageID(ctx, tag)
	if err != nil {
		return failed{err: fmt.Errorf("getting imageID for %s: %v", tag, err)}
	}

	// Image exists locally with the same tag
	if idForTag == entry.ID {
		return found{hash: hash}
	}

	// Image exists locally with a different tag
	if c.client.ImageExists(ctx, entry.ID) {
		return needsLocalTagging{hash: hash, tag: tag, imageID: entry.ID}
	}

	return needsBuilding{hash: hash}
}

func (c *cache) lookupRemote(ctx context.Context, hash, tag string, entry ImageDetails) cacheDetails {
	if remoteDigest, err := docker.RemoteDigest(tag, c.insecureRegistries); err == nil {
		// Image exists remotely with the same tag and digest
		if remoteDigest == entry.Digest {
			return found{hash: hash}
		}
	}

	// Image exists remotely with a different tag
	fqn := tag + "@" + entry.Digest // Actual tag will be ignored but we need the registry and the digest part of it.
	if remoteDigest, err := docker.RemoteDigest(fqn, c.insecureRegistries); err == nil {
		if remoteDigest == entry.Digest {
			return needsRemoteTagging{hash: hash, tag: tag, digest: entry.Digest}
		}
	}

	// Image exists locally
	if entry.ID != "" && c.client != nil && c.client.ImageExists(ctx, entry.ID) {
		return needsPushing{hash: hash, tag: tag, imageID: entry.ID}
	}

	return needsBuilding{hash: hash}
}
