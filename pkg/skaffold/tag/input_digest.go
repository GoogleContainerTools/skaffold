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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

type inputDigestTagger struct {
	cfg   docker.Config
	ag    graph.ArtifactGraph
	cache graph.SourceDependenciesCache
}

func NewInputDigestTagger(cfg docker.Config, ag graph.ArtifactGraph) (Tagger, error) {
	return &inputDigestTagger{
		cfg:   cfg,
		ag:    ag,
		cache: graph.NewSourceDependenciesCache(cfg, nil, ag),
	}, nil
}

func (t *inputDigestTagger) GenerateTag(image latest_v1.Artifact) (string, error) {
	var inputs []string
	// TODO(nkubala): plumb through context into Tagger interface
	ctx := context.TODO()
	// srcFiles, err := getDependenciesForArtifact(ctx, &image, t.cfg, nil)
	srcFiles, err := t.cache.TransitiveArtifactDependencies(ctx, &image)
	if err != nil {
		return "", err
	}

	for _, artifactDep := range t.ag.Dependencies(&image) {
		// srcOfDep, err := getDependenciesForArtifact(ctx, artifactDep, t.cfg, nil)
		srcOfDep, err := t.cache.TransitiveArtifactDependencies(ctx, artifactDep)
		if err != nil {
			return "", err
		}
		srcFiles = append(srcFiles, srcOfDep...)
	}

	// must sort as hashing is sensitive to the order in which files are processed
	sort.Strings(srcFiles)
	for _, d := range srcFiles {
		h, err := fileHasher(d)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Tracef("skipping dependency %q for artifact cache calculation: %v", d, err)
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
func fileHasher(path string) (string, error) {
	h := md5.New()
	fi, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	h.Write([]byte(filepath.Clean(path)))
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
