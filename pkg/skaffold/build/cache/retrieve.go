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
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// For testing
	hashForArtifact   = getHashForArtifact
	imgExistsRemotely = imageExistsRemotely
)

// ImageDetails holds the Digest and ID of an image
type ImageDetails struct {
	Digest string `yaml:"digest,omitempty"`
	ID     string `yaml:"id,omitempty"`
}

// RetrieveCachedArtifacts checks to see if artifacts are cached, and returns tags for cached images, otherwise a list of images to be built
func (c *Cache) RetrieveCachedArtifacts(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) ([]*latest.Artifact, []build.Artifact, error) {
	if !c.useCache {
		return artifacts, nil, nil
	}

	color.Default.Fprintln(out, "Checking cache...")
	start := time.Now()

	wg := sync.WaitGroup{}
	builtImages := make([]bool, len(artifacts))
	needToBuildImages := make([]bool, len(artifacts))

	for i := range artifacts {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			ok, err := c.retrieveCachedArtifact(ctx, out, artifacts[i])
			if ok {
				builtImages[i] = true
				return
			}
			if err != nil {
				logrus.Debugf("error finding cached artifact for %s: %v", artifacts[i].ImageName, err)
			}
			color.Default.Fprintf(out, " - %s: ", artifacts[i].ImageName)
			color.Red.Fprintf(out, "Not Found. Rebuilding.\n")
			needToBuildImages[i] = true
		}()
	}

	wg.Wait()

	var (
		needToBuild []*latest.Artifact
		built       []build.Artifact
	)

	for i, imageBuilt := range builtImages {
		if imageBuilt {
			built = append(built, build.Artifact{
				ImageName: artifacts[i].ImageName,
				Tag:       HashTag(artifacts[i]),
			})
		}
	}

	for i, imageBuilt := range needToBuildImages {
		if imageBuilt {
			needToBuild = append(needToBuild, artifacts[i])
		}
	}

	color.Default.Fprintln(out, "Cache check complete in", time.Since(start))
	return needToBuild, built, nil
}

// retrieveCachedArtifact tries to retrieve a cached artifact.
// If it cannot, it returns false.
// If it can, it returns true.
func (c *Cache) retrieveCachedArtifact(ctx context.Context, out io.Writer, a *latest.Artifact) (bool, error) {
	hash, err := hashForArtifact(ctx, c.builder, a)
	if err != nil {
		return false, errors.Wrapf(err, "getting hash for artifact %s", a.ImageName)
	}
	a.WorkspaceHash = hash
	imageDetails, cacheHit := c.artifactCache[hash]
	if !cacheHit {
		return false, nil
	}
	if c.localCluster && !c.pushImages {
		return c.artifactExistsLocally(ctx, out, a, imageDetails)
	}
	return c.artifactExistsRemotely(ctx, out, a, imageDetails)
}

// artifactExistsLocally assumes the artifact must exist locally.
// This is called when using a local cluster and push:false.
// 1. Check if image exists locally
// 2. If not, check if same digest exists locally. If it does, retag.
// 3. If not, rebuild
// It returns true if the artifact exists locally, and false if not.
func (c *Cache) artifactExistsLocally(ctx context.Context, out io.Writer, a *latest.Artifact, imageDetails ImageDetails) (bool, error) {
	hashTag := HashTag(a)
	if c.client.ImageExists(ctx, hashTag) {
		color.Default.Fprintf(out, " - %s: ", a.ImageName)
		color.Green.Fprintf(out, "Found\n")
		return true, nil
	}
	// Check for a local image with the same digest as the image we want to build
	prebuiltImage, err := c.retrievePrebuiltImage(imageDetails)
	if err != nil {
		return false, errors.Wrapf(err, "getting prebuilt image")
	}
	// If prebuilt image exists, tag it. Otherwise, return false, as artifact doesn't exist locally.
	if prebuiltImage != "" {
		color.Default.Fprintf(out, " - %s: ", a.ImageName)
		color.Green.Fprintf(out, "Found. Retagging.\n")
		if err := c.client.Tag(ctx, prebuiltImage, hashTag); err != nil {
			return false, errors.Wrap(err, "retagging image")
		}
		return true, nil
	}
	return false, nil
}

// artifactExistsRemotely assumes the artifact must exist locally.
// this is used when running a remote cluster, or when push:true
// 1. Check if image exists remotely.
// 2. If not, check if same digest exists locally. If it does, retag and repush.
// 3. If not, rebuild.
// It returns true if the artifact exists remotely, and false if not.
func (c *Cache) artifactExistsRemotely(ctx context.Context, out io.Writer, a *latest.Artifact, imageDetails ImageDetails) (bool, error) {
	hashTag := HashTag(a)
	if imgExistsRemotely(hashTag, imageDetails.Digest, c.insecureRegistries) {
		color.Default.Fprintf(out, " - %s: ", a.ImageName)
		color.Green.Fprintf(out, "Found\n")
		return true, nil
	}

	// Check if image exists locally.
	if c.client.ImageExists(ctx, hashTag) {
		color.Default.Fprintf(out, " - %s: ", a.ImageName)
		color.Green.Fprintf(out, "Found Locally. Pushing.\n")
		if _, err := c.client.Push(ctx, out, hashTag); err != nil {
			return false, errors.Wrap(err, "retagging image")
		}
		return true, nil
	}

	// Check for a local image with the same digest as the image we want to build
	prebuiltImage, err := c.retrievePrebuiltImage(imageDetails)
	if err != nil {
		return false, errors.Wrapf(err, "getting prebuilt image")
	}
	// If prebuilt image exists, tag it and push it. Otherwise, return false, as artifact doesn't exist locally.
	if prebuiltImage != "" {
		color.Default.Fprintf(out, " - %s: ", a.ImageName)
		color.Green.Fprintf(out, "Found Locally. Retagging and Pushing.\n")
		if err := c.client.Tag(ctx, prebuiltImage, hashTag); err != nil {
			return false, errors.Wrap(err, "retagging image")
		}
		if _, err := c.client.Push(ctx, out, hashTag); err != nil {
			return false, errors.Wrap(err, "retagging image")
		}
		return true, nil
	}
	return false, nil
}

func (c *Cache) retrievePrebuiltImage(details ImageDetails) (string, error) {
	if c.client == nil {
		return "", nil
	}
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

func imageExistsRemotely(image, digest string, insecureRegistries map[string]bool) bool {
	if digest == "" {
		logrus.Debugf("Checking if %s exists remotely, but digest is empty", image)
		return false
	}
	d, err := remoteDigest(image, insecureRegistries)
	if err != nil {
		logrus.Debugf("Checking if %s exists remotely, can't get digest: %v", image, err)
		return false
	}
	return d == digest
}

func HashTag(a *latest.Artifact) string {
	return fmt.Sprintf("%s:%s", a.ImageName, a.WorkspaceHash)
}
