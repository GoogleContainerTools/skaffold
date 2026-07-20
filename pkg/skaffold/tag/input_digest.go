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

package tag

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type inputDigestTagger struct {
	cfg   docker.Config
	cache graph.SourceDependenciesCache
}

func NewInputDigestTagger(cfg docker.Config, ag graph.ArtifactGraph) (Tagger, error) {
	return NewInputDigestTaggerWithSourceCache(cfg, graph.NewSourceDependenciesCache(cfg, docker.NewSimpleStubArtifactResolver(), ag))
}

func NewInputDigestTaggerWithSourceCache(cfg docker.Config, cache graph.SourceDependenciesCache) (Tagger, error) {
	return &inputDigestTagger{
		cfg:   cfg,
		cache: cache,
	}, nil
}

func (t *inputDigestTagger) GenerateTag(ctx context.Context, image latest.Artifact) (string, error) {
	var inputs []string
	srcFiles, err := t.cache.TransitiveArtifactDependencies(ctx, &image)
	if err != nil {
		return "", err
	}

	if image.DockerArtifact != nil {
		srcFiles = append(srcFiles, image.DockerArtifact.DockerfilePath)
	}

	if image.KanikoArtifact != nil {
		srcFiles = append(srcFiles, image.KanikoArtifact.DockerfilePath)
	}

	if image.CustomArtifact != nil && image.CustomArtifact.Dependencies != nil && image.CustomArtifact.Dependencies.Dockerfile != nil {
		srcFiles = append(srcFiles, image.CustomArtifact.Dependencies.Dockerfile.Path)
	}

	// must sort as hashing is sensitive to the order in which files are processed
	sort.Strings(srcFiles)
	for _, d := range srcFiles {
		// if the dependency is not an absolute path, we consider it relative to the workspace
		// (or fileHasher will fail to find it)
		if !filepath.IsAbs(d) {
			d = filepath.Join(image.Workspace, d)
		}
		h, err := fileHasher(d, image.Workspace)
		if err != nil {
			if os.IsNotExist(err) {
				log.Entry(ctx).Tracef("skipping dependency %q for artifact cache calculation: %v", d, err)
				continue // Ignore files that don't exist
			}

			return "", fmt.Errorf("getting hash for %q: %w", d, err)
		} else {
			log.Entry(ctx).Tracef("dependency %q hash: %v", d, h)
		}
		inputs = append(inputs, h)
	}

	return encode(inputs)
}

func encode(inputs []string) (string, error) {
	// get a key for the hashes
	hasher := sha256.New()
	enc := json.NewEncoder(hasher)
	if err := enc.Encode(inputs); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// fileHasher hashes the contents and name of a file
func fileHasher(path string, workspacePath string) (string, error) {
	h := md5.New()
	fi, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	// Always try to use the file path relative to workspace when calculating hash.
	// This will ensure we will always get the same hash independent of workspace location and hierarchy.
	pathToHash, err := filepath.Rel(workspacePath, path)
	if err != nil {
		pathToHash = path
	}
	log.Entry(context.TODO()).Tracef("Hashing file %q %s %s", pathToHash, workspacePath, path)

	if fi.Mode().IsRegular() {
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		log.Entry(context.TODO()).Tracef("MD5 content hash for %s: %x", pathToHash, h.Sum(nil))
	}

	// include file path in the hash to catch renames
	// (after content has been hashed, so it is comparable with other tools like md5sum)
	h.Write([]byte(pathToHash))

	return hex.EncodeToString(h.Sum(nil)), nil
}
