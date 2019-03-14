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

	start := time.Now()
	color.Default.Fprintln(out, "Checking cache...")

	var needToBuild []*latest.Artifact
	var built []build.Artifact

	var wg sync.WaitGroup
	wg.Add(len(artifacts))

	var canceled bool

	for _, a := range artifacts {
		a := a
		go func() {
			select {
			case <-ctx.Done():
				canceled = true
			default:
				defer wg.Done()
				artifact, err := c.resolveCachedArtifact(ctx, out, a)
				if err != nil {
					logrus.Debugf("error retrieving cached artifact for %s: %v\n", a.ImageName, err)
					color.Red.Fprintf(out, "Unable to retrieve %s from cache; this image will be rebuilt.\n", a.ImageName)
					needToBuild = append(needToBuild, a)
					return
				}
				if artifact == nil {
					needToBuild = append(needToBuild, a)
					return
				}
				built = append(built, *artifact)
			}
		}()
	}

	if canceled {
		return nil, nil, context.Canceled
	}

	wg.Wait()

	color.Default.Fprintln(out, "Cache check complete in", time.Since(start))
	return needToBuild, built, nil
}

func (c *Cache) resolveCachedArtifact(ctx context.Context, out io.Writer, a *latest.Artifact) (*build.Artifact, error) {
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

	return &build.Artifact{
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
	imageDetails, cacheHit := c.artifactCache[hash]
	if !cacheHit {
		return &cachedArtifactDetails{
			needsRebuild: true,
		}, nil
	}
	hashTag := HashTag(a)
	il, err := c.imageLocation(ctx, imageDetails, hashTag)
	if err != nil {
		return nil, errors.Wrapf(err, "getting artifact details for %s", a.ImageName)
	}
	return &cachedArtifactDetails{
		needsRebuild:  needsRebuild(il, c.localCluster),
		needsRetag:    needsRetag(il),
		needsPush:     needsPush(il, c.localCluster, c.needsPush),
		prebuiltImage: il.prebuiltImage,
		hashTag:       hashTag,
	}, nil
}

// imageLocation holds information about where the image currently is
type imageLocation struct {
	existsRemotely bool
	existsLocally  bool
	prebuiltImage  string
}

func (c *Cache) imageLocation(ctx context.Context, imageDetails ImageDetails, tag string) (*imageLocation, error) {
	// Check if tagged image exists remotely with the same digest
	existsRemotely := imgExistsRemotely(tag, imageDetails.Digest)
	existsLocally := false
	if c.client != nil {
		// See if this image exists in the local daemon
		if c.client.ImageExists(ctx, tag) {
			existsLocally = true
		}
	}
	if existsLocally {
		return &imageLocation{
			existsLocally:  existsLocally,
			existsRemotely: existsRemotely,
			prebuiltImage:  tag,
		}, nil
	}
	// Check for a local image with the same digest as the image we want to build
	prebuiltImage, err := c.retrievePrebuiltImage(imageDetails)
	if err != nil {
		return nil, errors.Wrapf(err, "getting prebuilt image")
	}
	return &imageLocation{
		existsRemotely: existsRemotely,
		existsLocally:  existsLocally,
		prebuiltImage:  prebuiltImage,
	}, nil
}

func needsRebuild(d *imageLocation, localCluster bool) bool {
	// If using local cluster, rebuild if all of the following are true:
	//   1. does not exist locally
	//   2. can't retag a prebuilt image
	if localCluster {
		return !d.existsLocally && d.prebuiltImage == ""
	}
	// If using remote cluster, only rebuild image if all of the following are true:
	//  1. does not exist locally
	//  2. does not exist remotely
	//  3. can't retag a prebuilt image
	return !d.existsLocally && !d.existsRemotely && d.prebuiltImage == ""
}

func needsPush(d *imageLocation, localCluster, push bool) bool {
	// If using local cluster...
	if localCluster {
		// ...  only push if specified and image does not exist remotely
		return push && !d.existsRemotely
	}
	// If using remote cluster, push if image does not exist remotely
	return !d.existsRemotely
}

func needsRetag(d *imageLocation) bool {
	// Don't need a retag if image already exists locally
	if d.existsLocally {
		return false
	}
	// If a prebuilt image is found locally, retag the image
	return d.prebuiltImage != ""
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

func HashTag(a *latest.Artifact) string {
	return fmt.Sprintf("%s:%s", a.ImageName, a.WorkspaceHash)
}
