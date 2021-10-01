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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
)

type inputDigestTagger struct {
	cfg   docker.Config
	cache graph.SourceDependenciesCache
}

func NewInputDigestTagger(cfg docker.Config, ag graph.ArtifactGraph) (Tagger, error) {
	return &inputDigestTagger{
		cfg:   cfg,
		cache: graph.NewSourceDependenciesCache(cfg, nil, ag),
	}, nil
}

func (t *inputDigestTagger) GenerateTag(ctx context.Context, image latestV2.Artifact) (string, error) {
	var inputs []string
	srcFiles, err := t.cache.TransitiveArtifactDependencies(ctx, &image)
	if err != nil {
		return "", err
	}

	// must sort as hashing is sensitive to the order in which files are processed
	sort.Strings(srcFiles)
	for _, d := range srcFiles {
		h, err := fileHasher(d, image.Workspace)
		if err != nil {
			if os.IsNotExist(err) {
				log.Entry(ctx).Tracef("skipping dependency %q for artifact cache calculation: %v", d, err)
				continue // Ignore files that don't exist
			}

			return "", fmt.Errorf("getting hash for %q: %w", d, err)
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
	h.Write([]byte(pathToHash))

	if fi.Mode().IsRegular() {
		f, err := os.Open(path)
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
