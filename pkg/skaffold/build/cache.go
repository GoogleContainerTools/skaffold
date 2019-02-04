/*
Copyright 2018 The Skaffold Authors

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
	"crypto/sha256"
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

// ArtifactCache is a map of hash to image digest
type ArtifactCache map[string]string

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
		logrus.Warnf("Error resolving cache file, not using cache: %v", err)
		return &Cache{}
	}
	cache, err := retrieveArtifactCache(cf)
	if err != nil {
		logrus.Warnf("Error retrieving artifact cache, not using cache: %v", err)
		return &Cache{}
	}
	client, err := docker.NewAPIClient()
	if err != nil {
		logrus.Warnf("Error retrieving local daemon client, not using cache: %v", err)
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

// RetrieveCachedArtifacts checks to see if artifacts are cached, and returns tags for cached images otherwise a list of images to be built
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
			fmt.Printf("error retrieving cached artifact for %s: %v\n", a.ImageName, err)
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
	imageID, ok := c.artifactCache[hash]
	if !ok {
		return nil, nil
	}
	newTag := fmt.Sprintf("%s:%s", a.ImageName, imageID[7:])
	// Check if image exists remotely
	existsRemotely := imageExistsRemotely(newTag, imageID)
	if existsRemotely {
		return &Artifact{
			ImageName: a.ImageName,
			Tag:       newTag,
		}, nil
	}

	// First, check if this image has already been built and pushed remotely
	if c.client.ImageExists(ctx, newTag) {
		color.Yellow.Fprintf(out, "Found %s locally as %s, checking if image is available remotely...\n", a.ImageName, newTag)
		// Push if the image doesn't exist remotely
		if !existsRemotely {
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
	// Check for local image with the same id as cached
	id, err := c.client.ImageFromID(ctx, imageID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting image from id %s", imageID)
	}
	if id == "" {
		return nil, errors.Wrapf(err, "empty id")
	}
	// Retag the image
	if err := c.client.Tag(ctx, id, newTag); err != nil {
		return nil, errors.Wrap(err, "retagging image")
	}
	// Push the retagged image
	color.Yellow.Fprintf(out, "Found %s locally, retagging and pushing...\n", a.ImageName)
	if !existsRemotely {
		if _, err := c.client.Push(ctx, out, newTag); err != nil {
			return nil, errors.Wrap(err, "pushing image")
		}
	}

	color.Yellow.Fprintf(out, "Retagged %s, skipping rebuild.\n", id)
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
	return d.Hex == digest
}

// CacheArtifacts determines the hash for each artifact, stores it in the artifact cache, and saves the cache at the end
func (c *Cache) CacheArtifacts(ctx context.Context, artifacts []*latest.Artifact, buildArtifacts []Artifact) error {
	if !c.useCache {
		return nil
	}
	fmt.Println("Caching artifacts", buildArtifacts)
	tags := map[string]string{}
	for _, t := range buildArtifacts {
		tags[t.ImageName] = t.Tag
	}
	for _, a := range artifacts {
		hash, err := hashForArtifact(ctx, a)
		if err != nil {
			continue
		}
		id, err := c.retrieveImageID(ctx, tags[a.ImageName])
		if err != nil {
			logrus.Warn("error getting id for %s: %v", a.ImageName, err)
			continue
		}
		if id == "" {
			logrus.Debugf("not caching %s because image id is empty", a.ImageName)
			continue
		}
		c.artifactCache[hash] = id
	}
	return c.save()
}

// First, check the local daemon. If that doesn't exist, check a remote registry.
func (c *Cache) retrieveImageID(ctx context.Context, img string) (string, error) {
	id, err := c.client.ImageID(ctx, img)
	if err == nil && id != "" {
		return id, nil
	}
	return docker.RemoteDigest(img)
}

// Save saves the artifactCache to the cacheFile
func (c *Cache) save() error {
	data, err := yaml.Marshal(c.artifactCache)
	if err != nil {
		return errors.Wrap(err, "marshalling hashes")
	}
	fmt.Println("writing", c.artifactCache, "to", c.cacheFile)
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
	return SHA256(c)
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

// SHA256 returns the shasum of the contents of r
func SHA256(r io.Reader) (string, error) {
	hasher := sha256.New()
	_, err := io.Copy(hasher, r)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size()))), nil
}
