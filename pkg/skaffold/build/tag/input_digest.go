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
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/dep"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type inputDigestTagger struct {
	cfg docker.Config
	ag  dep.ArtifactGraph
}

func NewInputDigestTagger(cfg docker.Config, ag dep.ArtifactGraph) (Tagger, error) {
	return &inputDigestTagger{
		cfg: cfg,
		ag:  ag,
	}, nil
}

func (t *inputDigestTagger) GenerateTag(_ string, image *latest.Artifact) (string, error) {
	var inputs []string
	ctx := context.Background()
	srcFies, err := dep.DependenciesForArtifact(ctx, image, t.cfg, nil)

	if err != nil {
		return "", err
	}

	for _, dkrDep := range t.ag.Dependencies(image) {
		srcOfDep, err := dep.DependenciesForArtifact(ctx, dkrDep, t.cfg, nil)
		if err != nil {
			return "", err
		}

		srcFies = append(srcFies, srcOfDep...)
	}

	sort.Strings(srcFies)
	for _, d := range srcFies {
		h, err := fileHasher(d)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Tracef("skipping dependency for artifact cache calculation, file not found %s: %s", d, err)
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
func fileHasher(p string) (string, error) {
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
