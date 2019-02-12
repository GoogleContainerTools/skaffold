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

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
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

// ImageDetails holds the Digest and ID of an image
type ImageDetails struct {
	Digest string `yaml:"digest,omitempty"`
	ID     string `yaml:"id,omitempty"`
}

// ArtifactCache is a map of [artifact dependencies hash : ImageDetails]
type ArtifactCache map[string]ImageDetails

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
	localCluster    = config.GetLocalCluster
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
	a.WorkspaceHash = hash
	imageDetails, cacheHit := c.artifactCache[hash]
	if !cacheHit {
		return nil, nil
	}
	newTag := fmt.Sprintf("%s:%s", a.ImageName, hash)
	// Check if tagged image exists remotely with the same digest
	existsRemotely := imageExistsRemotely(newTag, imageDetails.Digest)
	// Check if we are using a local cluster
	local, _ := localCluster()

	// See if this image exists in the local daemon
	if c.client.ImageExists(ctx, newTag) {
		color.Yellow.Fprintf(out, "Found %s locally...\n", a.ImageName)
		// Push if the image doesn't exist remotely
		if !existsRemotely && !local {
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
	prebuiltImage, err := c.retrievePrebuiltImage(ctx, imageDetails)
	if err != nil {
		return nil, errors.Wrapf(err, "getting prebuilt image")
	}
	if prebuiltImage == "" {
		return nil, errors.New("no prebuilt image")
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

func (c *Cache) retrievePrebuiltImage(ctx context.Context, details ImageDetails) (string, error) {
	// first, search for an image with the same image ID
	img, err := c.client.ImageFromID(ctx, details.ID)
	if err != nil {
		logrus.Debugf("error getting tagged image with id %s, checking digest: %v", details.ID, err)
	}
	if err == nil && img != "" {
		return img, nil
	}
	// else, search for an image with the same digest
	img, err = c.client.TaggedImageFromDigest(ctx, details.Digest)
	if err != nil {
		return "", errors.Wrapf(err, "getting image from digest %s", details.Digest)
	}
	if img == "" {
		return "", errors.New("no prebuilt image")
	}
	return img, nil
}

func imageExistsRemotely(image, digest string) bool {
	if digest == "" {
		return false
	}
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
			logrus.Debugf("error getting id for %s: %v, will try to get image id (expected with a local cluster)", tags[a.ImageName], err)
		}
		if digest == "" {
			logrus.Debugf("couldn't get image digest for %s, will try to cache just image id (expected with a local cluster)", tags[a.ImageName])
		}
		id, err := c.client.ImageID(ctx, tags[a.ImageName])
		if err != nil {
			logrus.Debugf("couldn't get image id for %s", tags[a.ImageName])
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

// Retag retags newly built images in the format [imageName:workspaceHash] and pushes them if using a remote cluster
func (c *Cache) Retag(ctx context.Context, out io.Writer, artifactsToBuild []*latest.Artifact, buildArtifacts []Artifact) {
	if !c.useCache {
		return
	}
	tags := map[string]string{}
	for _, t := range buildArtifacts {
		tags[t.ImageName] = t.Tag
	}
	local, _ := localCluster()
	color.Default.Fprintln(out, "Retagging cached images...")
	for _, artifact := range artifactsToBuild {
		newTag := fmt.Sprintf("%s:%s", artifact.ImageName, artifact.WorkspaceHash)
		// Retag the image
		if err := c.client.Tag(ctx, tags[artifact.ImageName], newTag); err != nil {
			logrus.Warnf("error retagging %s as %s, caching for this image may not work: %v", tags[artifact.ImageName], newTag, err)
			continue
		}
		if local {
			continue
		}
		// Push the retagged image
		if _, err := c.client.Push(ctx, out, newTag); err != nil {
			logrus.Warnf("error pushing %s, caching for this image may not work: %v", newTag, err)
		}
	}
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
