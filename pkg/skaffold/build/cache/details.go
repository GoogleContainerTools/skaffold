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
	"io"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
)

type cacheDetails interface {
	Hash() string
}

// Failed: couldn't lookup cache
type failed struct {
	err error
}

func (d failed) Hash() string {
	return ""
}

// Not found, needs building
type needsBuilding struct {
	hash string
}

func (d needsBuilding) Hash() string {
	return d.hash
}

// Found in cache
type found struct {
	hash string
}

func (d found) Hash() string {
	return d.hash
}

type needsTagging interface {
	Tag(context.Context, *cache, platform.Resolver) error
}

// Found locally with wrong tag. Needs retagging
type needsLocalTagging struct {
	hash    string
	tag     string
	imageID string
}

func (d needsLocalTagging) Hash() string {
	return d.hash
}

func (d needsLocalTagging) Tag(ctx context.Context, c *cache, platforms platform.Resolver) error {
	return c.client.Tag(ctx, d.imageID, d.tag)
}

// Found remotely with wrong tag. Needs retagging
type needsRemoteTagging struct {
	hash   string
	tag    string
	digest string
}

func (d needsRemoteTagging) Hash() string {
	return d.hash
}

func (d needsRemoteTagging) Tag(ctx context.Context, c *cache, platforms platform.Resolver) error {
	fqn := d.tag + "@" + d.digest // Tag is not important. We just need the registry and the digest to locate the image.
	matched := platforms.GetPlatforms(strings.Split(filepath.Base(d.tag), ":")[0])
	return docker.AddRemoteTag(fqn, d.tag, c.cfg, matched.Platforms)
}

// Found locally. Needs pushing
type needsPushing struct {
	hash    string
	tag     string
	imageID string
}

func (d needsPushing) Hash() string {
	return d.hash
}

func (d needsPushing) Push(ctx context.Context, out io.Writer, c *cache) error {
	if err := c.client.Tag(ctx, d.imageID, d.tag); err != nil {
		return err
	}

	digest, err := c.client.Push(ctx, out, d.tag)
	if err != nil {
		return err
	}

	// Update cache
	c.cacheMutex.Lock()
	e := c.artifactCache[d.hash]
	e.Digest = digest
	c.artifactCache[d.hash] = e
	c.cacheMutex.Unlock()
	return nil
}
