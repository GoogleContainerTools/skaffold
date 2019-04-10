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
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// Retag retags newly built images in the format [imageName:workspaceHash] and pushes them if using a remote cluster
func (c *Cache) RetagLocalImages(ctx context.Context, out io.Writer, artifactsToBuild []*latest.Artifact, buildArtifacts []build.Artifact) {
	if !c.useCache || len(artifactsToBuild) == 0 {
		return
	}
	// Built images remotely, nothing to retag
	if !c.isLocalBuilder {
		return
	}
	tags := map[string]string{}
	for _, t := range buildArtifacts {
		tags[t.ImageName] = t.Tag
	}
	color.Default.Fprintln(out, "Retagging cached images...")
	for _, artifact := range artifactsToBuild {
		hashTag := fmt.Sprintf("%s:%s", artifact.ImageName, artifact.WorkspaceHash)
		// Retag the image
		if err := c.client.Tag(ctx, tags[artifact.ImageName], hashTag); err != nil {
			logrus.Warnf("error retagging %s as %s, caching for this image may not work: %v", tags[artifact.ImageName], hashTag, err)
			continue
		}
		if c.localCluster {
			continue
		}
		// Push the retagged image
		if _, err := c.client.Push(ctx, out, hashTag); err != nil {
			logrus.Warnf("error pushing %s, caching for this image may not work: %v", hashTag, err)
		}
	}
}

// CacheArtifacts determines the hash for each artifact, stores it in the artifact cache, and saves the cache at the end
func (c *Cache) CacheArtifacts(ctx context.Context, artifacts []*latest.Artifact, buildArtifacts []build.Artifact) error {
	if !c.useCache {
		return nil
	}
	tags := map[string]string{}
	for _, t := range buildArtifacts {
		tags[t.ImageName] = t.Tag
	}
	for _, a := range artifacts {
		hash, err := hashForArtifact(ctx, c.builder, a)
		if err != nil {
			continue
		}
		digest, err := c.retrieveImageDigest(ctx, tags[a.ImageName])
		if err != nil {
			logrus.Debugf("error getting id for %s: %v, will try to get image id (expected with a local cluster)", tags[a.ImageName], err)
		}
		if digest == "" {
			logrus.Debugf("couldn't get image digest for %s, will try to cache just image id (expected with a local cluster)", tags[a.ImageName])
		}
		var id string
		if c.client != nil {
			id, err = c.client.ImageID(ctx, tags[a.ImageName])
			if err != nil {
				logrus.Debugf("couldn't get image id for %s", tags[a.ImageName])
			}
		}
		if id == "" && digest == "" {
			logrus.Debugf("both image id and digest are empty for %s, skipping caching", tags[a.ImageName])
			continue
		}
		c.artifactCache[hash] = ImageDetails{
			Digest: digest,
			ID:     id,
		}
	}
	return c.save()
}

// Check local daemon for img digest
func (c *Cache) retrieveImageDigest(ctx context.Context, img string) (string, error) {
	if c.client == nil {
		// Check for remote digest
		return docker.RemoteDigest(img, c.insecureRegistries)
	}
	repoDigest, err := c.client.RepoDigest(ctx, img)
	if err != nil {
		return docker.RemoteDigest(img, c.insecureRegistries)
	}
	ref, err := name.NewDigest(repoDigest, name.WeakValidation)
	return ref.DigestStr(), err
}

// Save saves the artifactCache to the cacheFile
func (c *Cache) save() error {
	data, err := yaml.Marshal(c.artifactCache)
	if err != nil {
		return errors.Wrap(err, "marshalling hashes")
	}
	return ioutil.WriteFile(c.cacheFile, data, 0755)
}
