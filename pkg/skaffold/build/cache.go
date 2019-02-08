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

package build

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// ArtifactCache is a map of [workspace hash : image digest]
type ArtifactCache map[string]string

// Cache holds any data necessary for accessing the cache
type Cache struct {
	artifactCache ArtifactCache
	client        docker.LocalDaemon
	cacheFile     string
	useCache      bool
}

var (
	// For testing
	hashForArtifact = getHashForArtifact
)

// NewCache returns the current state of the cache
func NewCache(useCache bool, cacheFile string) *Cache {
	if !useCache {
		return &Cache{}
	}
	cf, err := resolveCacheFile(cacheFile)
	if err != nil {
		logrus.Warnf("Error resolving cache file, not using skaffold cache: %v", err)
		return &Cache{}
	}
	cache, err := retrieveArtifactCache(cf)
	if err != nil {
		logrus.Warnf("Error retrieving artifact cache, not using skaffold cache: %v", err)
		return &Cache{}
	}
	client, err := docker.NewAPIClient()
	if err != nil {
		logrus.Warnf("Error retrieving local daemon client, not using skaffold cache: %v", err)
		return &Cache{}
	}
	return &Cache{
		artifactCache: cache,
		cacheFile:     cf,
		useCache:      useCache,
		client:        client,
	}
}

// resolveCacheFile makes sure that either a passed in cache file or the default cache file exists
func resolveCacheFile(cacheFile string) (string, error) {
	if cacheFile != "" {
		return cacheFile, util.VerifyOrCreateFile(cacheFile)
	}
	home, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "retrieving home directory")
	}
	defaultFile := filepath.Join(home, constants.DefaultSkaffoldDir, constants.DefaultCacheFile)
	return defaultFile, util.VerifyOrCreateFile(defaultFile)
}

func retrieveArtifactCache(cacheFile string) (ArtifactCache, error) {
	cache := ArtifactCache{}
	contents, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(contents, &cache); err != nil {
		return nil, err
	}
	return cache, nil
}

// RetrieveCachedArtifacts checks to see if artifacts are cached, and returns tags for cached images, otherwise a list of images to be built
func (c *Cache) RetrieveCachedArtifacts(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]*latest.Artifact, []Artifact) {
	if !c.useCache {
		return artifacts, nil
	}
	color.Default.Fprintln(out, "Checking cache...")
	var needToBuild []*latest.Artifact
	var built []Artifact
	for _, a := range artifacts {
		artifact, err := c.retrieveCachedArtifact(ctx, out, a)
		if err != nil {
			logrus.Debugf("error retrieving cached artifact for %s: %v\n", a.ImageName, err)
			needToBuild = append(needToBuild, a)
			continue
		}
		if artifact == nil {
			needToBuild = append(needToBuild, a)
			continue
		}
		built = append(built, *artifact)
	}
	return needToBuild, built
}

func (c *Cache) retrieveCachedArtifact(ctx context.Context, out io.Writer, a *latest.Artifact) (*Artifact, error) {
	hash, err := hashForArtifact(ctx, a)
	if err != nil {
		return nil, errors.Wrapf(err, "getting hash for artifact %s", a.ImageName)
	}
	imageDigest, cacheHit := c.artifactCache[hash]
	if !cacheHit {
		return nil, nil
	}
	newTag := fmt.Sprintf("%s:%s", a.ImageName, imageDigest[7:])
	// Check if image exists remotely
	existsRemotely := imageExistsRemotely(newTag, imageDigest)

	// See if this image exists in the local daemon
	if c.client.ImageExists(ctx, newTag) {
		color.Yellow.Fprintf(out, "Found %s locally...\n", a.ImageName)
		// Push if the image doesn't exist remotely
		if !existsRemotely {
			color.Yellow.Fprintf(out, "Pushing %s since it doesn't exist remotely...\n", a.ImageName)
			if _, err := c.client.Push(ctx, out, newTag); err != nil {
				return nil, errors.Wrapf(err, "pushing %s", newTag)
			}
		}
		color.Yellow.Fprintf(out, "%s ready, skipping rebuild\n", newTag)
		return &Artifact{
			ImageName: a.ImageName,
			Tag:       newTag,
		}, nil
	}

	// Check for a local image with the same digest as the image we want to build
	prebuiltImage, err := c.client.TaggedImageFromDigest(ctx, imageDigest)
	if err != nil {
		return nil, errors.Wrapf(err, "getting image from digest %s", imageDigest)
	}
	if prebuiltImage == "" {
		return nil, errors.Wrapf(err, "no prebuilt image")
	}
	color.Yellow.Fprintf(out, "Found %s locally, retagging and pushing...\n", a.ImageName)
	// Retag the image
	if err := c.client.Tag(ctx, prebuiltImage, newTag); err != nil {
		return nil, errors.Wrap(err, "retagging image")
	}
	// Push the retagged image
	if _, err := c.client.Push(ctx, out, newTag); err != nil {
		return nil, errors.Wrap(err, "pushing image")
	}

	color.Yellow.Fprintf(out, "Retagged %s, skipping rebuild.\n", prebuiltImage)
	return &Artifact{
		ImageName: a.ImageName,
		Tag:       newTag,
	}, nil
}

func imageExistsRemotely(image, digest string) bool {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return false
	}
	img, err := remote.Image(ref)
	if err != nil {
		return false
	}
	d, err := img.Digest()
	if err != nil {
		return false
	}
	return d.Hex == digest[7:]
}

// CacheArtifacts determines the hash for each artifact, stores it in the artifact cache, and saves the cache at the end
func (c *Cache) CacheArtifacts(ctx context.Context, artifacts []*latest.Artifact, buildArtifacts []Artifact) error {
	if !c.useCache {
		return nil
	}
	tags := map[string]string{}
	for _, t := range buildArtifacts {
		tags[t.ImageName] = t.Tag
	}
	for _, a := range artifacts {
		hash, err := hashForArtifact(ctx, a)
		if err != nil {
			continue
		}
		digest, err := c.retrieveImageDigest(ctx, tags[a.ImageName])
		if err != nil {
			logrus.Debugf("error getting id for %s: %v, skipping caching", tags[a.ImageName], err)
			continue
		}
		if digest == "" {
			logrus.Debugf("skipping caching %s because image id is empty", tags[a.ImageName])
			continue
		}
		c.artifactCache[hash] = digest
	}
	return c.save()
}

// Check local daemon for img digest
func (c *Cache) retrieveImageDigest(ctx context.Context, img string) (string, error) {
	repoDigest, err := c.client.RepoDigest(ctx, img)
	if err != nil {
		return docker.RemoteDigest(img)
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

func getHashForArtifact(ctx context.Context, a *latest.Artifact) (string, error) {
	deps, err := DependenciesForArtifact(ctx, a)
	if err != nil {
		return "", errors.Wrapf(err, "getting dependencies for %s", a.ImageName)
	}
	hasher := cacheHasher()
	var hashes []string
	for _, d := range deps {
		h, err := hasher(d)
		if err != nil {
			return "", errors.Wrapf(err, "getting hash for %s", d)
		}
		hashes = append(hashes, h)
	}
	// get a key for the hashes
	c := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(c)
	enc.Encode(hashes)
	return util.SHA256(c)
}

// cacheHasher takes hashes the contents and name of a file
func cacheHasher() func(string) (string, error) {
	hasher := func(p string) (string, error) {
		h := md5.New()
		fi, err := os.Lstat(p)
		if err != nil {
			return "", err
		}
		h.Write([]byte(fi.Mode().String()))
		if fi.Mode().IsRegular() {
			f, err := os.Open(p)
			if err != nil {
				return "", err
			}
			defer f.Close()
			if _, err := io.Copy(h, f); err != nil {
				return "", err
			}
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	}
	return hasher
}
