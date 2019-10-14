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
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
)

// For testing
var (
	hashFunction           = cacheHasher
	artifactConfigFunction = artifactConfig
)

func getHashForArtifact(ctx context.Context, depLister DependencyLister, a *latest.Artifact) (string, error) {
	var inputs []string

	// Append the artifact's configuration
	config, err := artifactConfigFunction(a)
	if err != nil {
		return "", errors.Wrapf(err, "getting artifact's configuration for %s", a.ImageName)
	}
	inputs = append(inputs, config)

	// Append the digest of each input file
	deps, err := depLister.DependenciesForArtifact(ctx, a)
	if err != nil {
		return "", errors.Wrapf(err, "getting dependencies for %s", a.ImageName)
	}
	sort.Strings(deps)

	for _, d := range deps {
		h, err := hashFunction(d)
		if err != nil {
			return "", errors.Wrapf(err, "getting hash for %s", d)
		}
		inputs = append(inputs, h)
	}

	// add build args for the artifact if specified
	if buildArgs := retrieveBuildArgs(a); buildArgs != nil {
		buildArgs, err := docker.EvaluateBuildArgs(buildArgs)
		if err != nil {
			return "", errors.Wrap(err, "evaluating build args")
		}
		args := convertBuildArgsToStringArray(buildArgs)
		inputs = append(inputs, args...)
	}

	// get a key for the hashes
	hasher := sha256.New()
	enc := json.NewEncoder(hasher)
	if err := enc.Encode(inputs); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func artifactConfig(a *latest.Artifact) (string, error) {
	buf, err := json.Marshal(a.ArtifactType)
	if err != nil {
		return "", errors.Wrapf(err, "marshalling the artifact's configuration for %s", a.ImageName)
	}

	return string(buf), nil
}

func retrieveBuildArgs(artifact *latest.Artifact) map[string]*string {
	switch {
	case artifact.DockerArtifact != nil:
		return artifact.DockerArtifact.BuildArgs

	case artifact.KanikoArtifact != nil:
		return artifact.KanikoArtifact.BuildArgs

	case artifact.CustomArtifact != nil && artifact.CustomArtifact.Dependencies.Dockerfile != nil:
		return artifact.CustomArtifact.Dependencies.Dockerfile.BuildArgs

	default:
		return nil
	}
}

func convertBuildArgsToStringArray(buildArgs map[string]*string) []string {
	var args []string
	for k, v := range buildArgs {
		if v == nil {
			args = append(args, k)
			continue
		}
		args = append(args, fmt.Sprintf("%s=%s", k, *v))
	}
	sort.Strings(args)
	return args
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
