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
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	"github.com/containers/storage"
)

func (c *cache) lookupArtifacts(ctx context.Context, tags tag.ImageTags, artifacts []*latestV1.Artifact) []cacheDetails {
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
			details[i] = c.lookup(ctx, artifacts[i], tags[artifacts[i].ImageName], h)
			wg.Done()
		}()
	}
	wg.Wait()

	return details
}

func (c *cache) lookup(ctx context.Context, a *latestV1.Artifact, tag string, h artifactHasher) cacheDetails {
	ctx, endTrace := instrumentation.StartTrace(ctx, "lookup_CacheLookupOneArtifact", map[string]string{
		"ImageName": instrumentation.PII(a.ImageName),
	})
	defer endTrace()

	hash, err := h.hash(ctx, a)
	if err != nil {
		return failed{err: fmt.Errorf("getting hash for artifact %q: %s", a.ImageName, err)}
	}

	c.cacheMutex.RLock()
	entry, cacheHit := c.artifactCache[hash]
	c.cacheMutex.RUnlock()
	if !cacheHit {
		if entry, err = c.tryImport(ctx, a, tag, hash); err != nil {
			log.Entry(ctx).Debugf("Could not import artifact from Docker, building instead (%s)", err)
			return needsBuilding{hash: hash}
		}
	}

	if isLocal, err := c.isLocalImage(a.ImageName); err != nil {
		return failed{err}
	} else if isLocal {
		log.Entry(ctx).Debugf("using local lookup for %v", a.ImageName)
		return c.lookupLocal(ctx, hash, tag, entry, a)
	}

	log.Entry(ctx).Debugf("using remote lookup for %v", a.ImageName)
	return c.lookupRemote(ctx, hash, tag, entry, a)
}

func (c *cache) lookupLocal(ctx context.Context, hash, tag string, entry ImageDetails, a *latestV1.Artifact) cacheDetails {
	if entry.ID == "" {
		return needsBuilding{hash: hash}
	}
	if a.BuildahArtifact != nil {
		log.Entry(ctx).Debugf("using libimage cache localLookup for %v", a.ImageName)
		return c.lookupLocalLibImage(ctx, hash, tag, a.ImageName, entry)
	}
	return c.lookupLocalDocker(ctx, hash, tag, entry)
}

func (c *cache) lookupLocalDocker(ctx context.Context, hash, tag string, entry ImageDetails) cacheDetails {
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

func (c *cache) lookupLocalLibImage(ctx context.Context, hash, tag, imageName string, entry ImageDetails) cacheDetails {
	image, _, err := c.libimageRuntime.LookupImage(fmt.Sprintf("%v:%v", imageName, tag), &libimage.LookupImageOptions{})
	if err == storage.ErrImageUnknown {
		// image was not found in local storage
		return needsBuilding{hash: hash}
	} else if err != nil {
		return failed{err: err}
	}

	// Image exists locally with the same tag
	if image.ID() == entry.ID {
		return found{hash: hash}
	}
	return needsLocalTagging{hash: hash, tag: tag, imageID: entry.ID}
}

func (c *cache) lookupRemote(ctx context.Context, hash, tag string, entry ImageDetails, a *latestV1.Artifact) cacheDetails {
	if remoteDigest, err := docker.RemoteDigest(tag, c.cfg); err == nil {
		// Image exists remotely with the same tag and digest
		if remoteDigest == entry.Digest {
			return found{hash: hash}
		}
	}

	// Image exists remotely with a different tag
	fqn := tag + "@" + entry.Digest // Actual tag will be ignored but we need the registry and the digest part of it.
	if remoteDigest, err := docker.RemoteDigest(fqn, c.cfg); err == nil {
		if remoteDigest == entry.Digest {
			return needsRemoteTagging{hash: hash, tag: tag, digest: entry.Digest}
		}
	}

	// Image exists locally
	var exists bool
	if a.BuildahArtifact != nil {
		_, _, err := c.libimageRuntime.LookupImage(a.ImageName, &libimage.LookupImageOptions{})
		if err != storage.ErrImageUnknown {
			exists = true
		}
	} else {
		exists = c.client.ImageExists(ctx, entry.ID)
	}

	if entry.ID != "" && c.client != nil && exists {
		return needsPushing{hash: hash, tag: tag, imageID: entry.ID}
	}

	return needsBuilding{hash: hash}
}

func (c *cache) tryImport(ctx context.Context, a *latestV1.Artifact, tag string, hash string) (ImageDetails, error) {
	if importMissing, err := c.importMissingImage(a.ImageName); err != nil {
		return ImageDetails{}, err
	} else if !importMissing {
		return ImageDetails{}, fmt.Errorf("import of missing images disabled")
	}

	var err error
	var entry ImageDetails

	if a.BuildahArtifact == nil {
		entry, err = c.tryImportDocker(ctx, tag)
		if err != nil {
			return ImageDetails{}, err
		}
	} else {
		entry, err = c.tryImportLibImage(ctx, a.ImageName, tag)
		if err != nil {
			return ImageDetails{}, err
		}
	}

	if digest, err := docker.RemoteDigest(tag, c.cfg); err == nil {
		log.Entry(ctx).Debugf("Added digest for %s to cache entry", tag)
		entry.Digest = digest
	}

	c.cacheMutex.Lock()
	c.artifactCache[hash] = entry
	c.cacheMutex.Unlock()
	return entry, nil
}

func (c *cache) tryImportDocker(ctx context.Context, tag string) (ImageDetails, error) {
	if !c.client.ImageExists(ctx, tag) {
		log.Entry(ctx).Debugf("Importing artifact %s from docker registry", tag)
		err := c.client.Pull(ctx, ioutil.Discard, tag)
		if err != nil {
			return ImageDetails{}, err
		}
	} else {
		log.Entry(ctx).Debugf("Importing artifact %s from local docker", tag)
	}

	imageID, err := c.client.ImageID(ctx, tag)
	if err != nil {
		return ImageDetails{}, err
	}

	entry := ImageDetails{}
	if imageID != "" {
		entry.ID = imageID
	}
	return entry, nil
}

func (c *cache) tryImportLibImage(ctx context.Context, imageName, tag string) (ImageDetails, error) {
	images, err := c.libimageRuntime.Pull(ctx, fmt.Sprintf("%v:%v", imageName, tag), config.PullPolicyMissing, &libimage.PullOptions{})
	if err != nil {
		return ImageDetails{}, fmt.Errorf("pulling image: %w", err)
	}
	if len(images) > 1 {
		return ImageDetails{}, errors.New("pulled multiple images")
	}
	var details ImageDetails
	for _, image := range images {
		details.Digest = image.StorageImage().Digest.String()
		details.ID = image.StorageImage().ID
	}
	return details, nil
}
