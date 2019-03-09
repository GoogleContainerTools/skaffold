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
	"sort"
	"time"

	skafconfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/docker/docker/api/types"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/google/go-containerregistry/pkg/name"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var (
	// For testing
	hashFunction = cacheHasher
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
	builder       Builder
	imageList     []types.ImageSummary
	cacheFile     string
	useCache      bool
	needsPush     bool
}

var (
	// For testing
	hashForArtifact = getHashForArtifact
	localCluster    = config.GetLocalCluster
	remoteDigest    = docker.RemoteDigest
	newDockerCilent = docker.NewAPIClient
	noCache         = &Cache{}
)

// NewCache returns the current state of the cache
func NewCache(ctx context.Context, builder Builder, opts *skafconfig.SkaffoldOptions, needsPush bool) *Cache {
	if !opts.CacheArtifacts {
		return noCache
	}
	cf, err := resolveCacheFile(opts.CacheFile)
	if err != nil {
		logrus.Warnf("Error resolving cache file, not using skaffold cache: %v", err)
		return noCache
	}
	cache, err := retrieveArtifactCache(cf)
	if err != nil {
		logrus.Warnf("Error retrieving artifact cache, not using skaffold cache: %v", err)
		return noCache
	}
	client, err := newDockerCilent()
	if err != nil {
		logrus.Warnf("Error retrieving local daemon client, not using skaffold cache: %v", err)
		return noCache
	}
	imageList, err := client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		logrus.Warn("Unable to get list of images from local docker daemon, won't be checked for cache.")
	}
	return &Cache{
		artifactCache: cache,
		cacheFile:     cf,
		useCache:      opts.CacheArtifacts,
		client:        client,
		builder:       builder,
		needsPush:     needsPush,
		imageList:     imageList,
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

	start := time.Now()
	color.Default.Fprintln(out, "Checking cache...")

	var needToBuild []*latest.Artifact
	var built []Artifact
	for _, a := range artifacts {
		artifact, err := c.resolveCachedArtifact(ctx, out, a)
		if err != nil {
			logrus.Debugf("error retrieving cached artifact for %s: %v\n", a.ImageName, err)
			color.Red.Fprintf(out, "Unable to retrieve %s from cache; this image will be rebuilt.\n", a.ImageName)
			needToBuild = append(needToBuild, a)
			continue
		}
		if artifact == nil {
			needToBuild = append(needToBuild, a)
			continue
		}
		built = append(built, *artifact)
	}

	color.Default.Fprintln(out, "Cache check complete in", time.Since(start))
	return needToBuild, built
}

func (c *Cache) resolveCachedArtifact(ctx context.Context, out io.Writer, a *latest.Artifact) (*Artifact, error) {
	details, err := c.retrieveCachedArtifactDetails(ctx, a)
	if err != nil {
		return nil, errors.Wrap(err, "getting cached artifact details")
	}

	color.Default.Fprintf(out, " - %s: ", a.ImageName)

	if details.needsRebuild {
		color.Red.Fprintln(out, "Not found. Rebuilding.")
		return nil, nil
	}

	color.Green.Fprint(out, "Found")
	if details.needsRetag {
		color.Green.Fprint(out, ". Retagging")
	}
	if details.needsPush {
		color.Green.Fprint(out, ". Pushing.")
	}
	color.Default.Fprintln(out)

	if details.needsRetag {
		if err := c.client.Tag(ctx, details.prebuiltImage, details.hashTag); err != nil {
			return nil, errors.Wrap(err, "retagging image")
		}
	}
	if details.needsPush {
		if _, err := c.client.Push(ctx, out, details.hashTag); err != nil {
			return nil, errors.Wrap(err, "pushing image")
		}
	}

	return &Artifact{
		ImageName: a.ImageName,
		Tag:       details.hashTag,
	}, nil
}

type cachedArtifactDetails struct {
	needsRebuild  bool
	needsRetag    bool
	needsPush     bool
	prebuiltImage string
	hashTag       string
}

func (c *Cache) retrieveCachedArtifactDetails(ctx context.Context, a *latest.Artifact) (*cachedArtifactDetails, error) {
	hash, err := hashForArtifact(ctx, c.builder, a)
	if err != nil {
		return nil, errors.Wrapf(err, "getting hash for artifact %s", a.ImageName)
	}
	a.WorkspaceHash = hash
	localCluster, _ := localCluster()
	imageDetails, cacheHit := c.artifactCache[hash]
	if !cacheHit {
		return &cachedArtifactDetails{
			needsRebuild: true,
		}, nil
	}
	hashTag := fmt.Sprintf("%s:%s", a.ImageName, hash)

	// Check if we are using a local cluster
	var existsRemotely bool
	if !localCluster {
		// Check if tagged image exists remotely with the same digest
		existsRemotely = imageExistsRemotely(hashTag, imageDetails.Digest)
	}

	// See if this image exists in the local daemon
	if c.client.ImageExists(ctx, hashTag) {
		return &cachedArtifactDetails{
			needsPush: (!existsRemotely && !localCluster) || (localCluster && c.needsPush),
			hashTag:   hashTag,
		}, nil
	}
	// Check for a local image with the same digest as the image we want to build
	prebuiltImage, err := c.retrievePrebuiltImage(imageDetails)
	if err != nil {
		return nil, errors.Wrapf(err, "getting prebuilt image")
	}
	if prebuiltImage == "" {
		return nil, errors.New("no tagged prebuilt image")
	}

	return &cachedArtifactDetails{
		needsRetag:    true,
		needsPush:     !localCluster || c.needsPush,
		prebuiltImage: prebuiltImage,
		hashTag:       hashTag,
	}, nil
}

func (c *Cache) retrievePrebuiltImage(details ImageDetails) (string, error) {
	for _, r := range c.imageList {
		if r.ID == details.ID && details.ID != "" {
			if len(r.RepoTags) == 0 {
				return "", nil
			}
			return r.RepoTags[0], nil
		}
		if details.Digest == "" {
			continue
		}
		for _, d := range r.RepoDigests {
			if getDigest(d) == details.Digest {
				// Return a tagged version of this image, since we can't retag an image in the image@sha256: format
				if len(r.RepoTags) > 0 {
					return r.RepoTags[0], nil
				}
			}
		}
	}
	return "", errors.New("no prebuilt image")
}

func getDigest(img string) string {
	ref, _ := name.NewDigest(img, name.WeakValidation)
	return ref.DigestStr()
}

func imageExistsRemotely(image, digest string) bool {
	if digest == "" {
		logrus.Debugf("Checking if %s exists remotely, but digest is empty", image)
		return false
	}
	d, err := remoteDigest(image)
	if err != nil {
		logrus.Debugf("Checking if %s exists remotely, can't get digest: %v", image, err)
		return false
	}
	return d == digest
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
	if !c.useCache || len(artifactsToBuild) == 0 {
		return
	}
	tags := map[string]string{}
	for _, t := range buildArtifacts {
		tags[t.ImageName] = t.Tag
	}
	local, _ := localCluster()
	color.Default.Fprintln(out, "Retagging cached images...")
	for _, artifact := range artifactsToBuild {
		hashTag := fmt.Sprintf("%s:%s", artifact.ImageName, artifact.WorkspaceHash)
		// Retag the image
		if err := c.client.Tag(ctx, tags[artifact.ImageName], hashTag); err != nil {
			logrus.Warnf("error retagging %s as %s, caching for this image may not work: %v", tags[artifact.ImageName], hashTag, err)
			continue
		}
		if local {
			continue
		}
		// Push the retagged image
		if _, err := c.client.Push(ctx, out, hashTag); err != nil {
			logrus.Warnf("error pushing %s, caching for this image may not work: %v", hashTag, err)
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

func getHashForArtifact(ctx context.Context, builder Builder, a *latest.Artifact) (string, error) {
	deps, err := builder.DependenciesForArtifact(ctx, a)
	if err != nil {
		return "", errors.Wrapf(err, "getting dependencies for %s", a.ImageName)
	}
	sort.Strings(deps)
	var hashes []string
	for _, d := range deps {
		h, err := hashFunction(d)
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
func cacheHasher(p string) (string, error) {
	h := md5.New()
	fi, err := os.Lstat(p)
	if err != nil {
		return "", err
	}
	h.Write([]byte(fi.Mode().String()))
	h.Write([]byte(fi.Name()))
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
