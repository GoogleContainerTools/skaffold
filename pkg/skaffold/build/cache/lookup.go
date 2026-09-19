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
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

func (c *cache) lookupArtifacts(ctx context.Context, out io.Writer, tags tag.ImageTags, platforms platform.Resolver, artifacts []*latest.Artifact) []cacheDetails {
	details := make([]cacheDetails, len(artifacts))
	// Create a new `artifactHasher` on every new dev loop.
	// This way every artifact hash is calculated at most once in a single dev loop, and recalculated on every dev loop.

	ctx, endTrace := instrumentation.StartTrace(ctx, "lookupArtifacts_CacheLookupArtifacts")
	defer endTrace()
	h := newArtifactHasherFunc(c.artifactGraph, c.lister, c.cfg.Mode())
	var wg sync.WaitGroup
	for i := range artifacts {
		wg.Add(1)

		i := i
		go func() {
			details[i] = c.lookup(ctx, out, artifacts[i], tags, platforms, h)
			wg.Done()
		}()
	}
	wg.Wait()

	return details
}

func (c *cache) lookup(ctx context.Context, out io.Writer, a *latest.Artifact, tags map[string]string, platforms platform.Resolver, h artifactHasher) cacheDetails {
	tag := tags[a.ImageName]
	ctx, endTrace := instrumentation.StartTrace(ctx, "lookup_CacheLookupOneArtifact", map[string]string{
		"ImageName": instrumentation.PII(a.ImageName),
	})
	defer endTrace()

	hash, err := h.hash(ctx, out, a, platforms, tag)
	if err != nil {
		return failed{err: fmt.Errorf("getting hash for artifact %q: %s", a.ImageName, err)}
	}

	c.cacheMutex.RLock()
	entry, cacheHit := c.artifactCache[hash]
	c.cacheMutex.RUnlock()

	pls := platforms.GetPlatforms(a.ImageName)
	// TODO (gaghosh): allow `tryImport` when the Docker daemon starts supporting multiarch images
	// See https://github.com/docker/buildx/issues/1220#issuecomment-1189996403
	if !cacheHit && !pls.IsMultiPlatform() {
		var pl v1.Platform
		if len(pls.Platforms) == 1 {
			pl = util.ConvertToV1Platform(pls.Platforms[0])
		}
		if entry, err = c.tryImport(ctx, a, tag, hash, pl); err != nil {
			log.Entry(ctx).Debugf("Could not import artifact from Docker, building instead (%s)", err)
			return needsBuilding{hash: hash}
		}
	}

	if isLocal, err := c.isLocalImage(a.ImageName); err != nil {
		log.Entry(ctx).Debugf("isLocalImage failed %v", err)
		return failed{err}
	} else if isLocal {
		return c.lookupLocal(ctx, hash, tag, entry)
	}
	return c.lookupRemote(ctx, hash, tag, pls.Platforms, entry)
}

func (c *cache) lookupLocal(ctx context.Context, hash, tag string, entry ImageDetails) cacheDetails {
	if entry.ID == "" {
		return needsBuilding{hash: hash}
	}

	// Check the imageID for the tag
	idForTag, err := c.client.ImageID(ctx, tag)
	if err != nil {
		// Rely on actionable errors thrown from pkg/skaffold/docker.LocalDaemon api.
		return failed{err: err}
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

func (c *cache) lookupRemote(ctx context.Context, hash, tag string, platforms []specs.Platform, entry ImageDetails) cacheDetails {
	if remoteDigest, err := docker.RemoteDigest(tag, c.cfg, nil); err == nil {
		// Image exists remotely with the same tag and digest
		log.Entry(ctx).Debugf("RemoteDigest: %s entry.Digest %s", remoteDigest, entry.Digest)
		if remoteDigest == entry.Digest {
			return found{hash: hash}
		}
	} else {
		log.Entry(ctx).Debugf("RemoteDigest error %v", err)
	}

	// Image exists remotely with a different tag
	fqn := tag + "@" + entry.Digest // Actual tag will be ignored but we need the registry and the digest part of it.
	if remoteDigest, err := docker.RemoteDigest(fqn, c.cfg, nil); err == nil {
		if remoteDigest == entry.Digest {
			return needsRemoteTagging{hash: hash, tag: tag, digest: entry.Digest, platforms: platforms}
		}
	}

	// Image exists locally
	if entry.ID != "" && c.client != nil && c.client.ImageExists(ctx, entry.ID) {
		return needsPushing{hash: hash, tag: tag, imageID: entry.ID}
	}

	return needsBuilding{hash: hash}
}

func (c *cache) tryImport(ctx context.Context, a *latest.Artifact, tag string, hash string, pl v1.Platform) (ImageDetails, error) {
	entry := ImageDetails{}

	if importMissing, err := c.importMissingImage(a.ImageName); err != nil {
		return entry, err
	} else if !importMissing {
		return ImageDetails{}, fmt.Errorf("import of missing images disabled")
	}

	// under buildx, docker daemon is not really needed and could be disabled
	load := true
	if c.buildx {
		_, err := c.client.ServerVersion(ctx)
		load = err == nil
		if !load {
			log.Entry(ctx).Debugf("Docker client error, disabling image load as using buildx: %v", err)
		}
	}
	if load {
		if !c.client.ImageExists(ctx, tag) {
			log.Entry(ctx).Debugf("Importing artifact %s from docker registry", tag)
			err := c.client.Pull(ctx, io.Discard, tag, pl)
			if err != nil {
				return entry, err
			}
		} else {
			log.Entry(ctx).Debugf("Importing artifact %s from local docker", tag)
		}
		imageID, err := c.client.ImageID(ctx, tag)
		if err != nil {
			return entry, err
		}
		if imageID != "" {
			entry.ID = imageID
		}
	}

	if digest, err := docker.RemoteDigest(tag, c.cfg, nil); err == nil {
		log.Entry(ctx).Debugf("Added digest for %s to cache entry", tag)
		entry.Digest = digest
	}

	c.cacheMutex.Lock()
	c.artifactCache[hash] = entry
	c.cacheMutex.Unlock()
	return entry, nil
}
