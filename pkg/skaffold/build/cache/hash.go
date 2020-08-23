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

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// For testing
var (
	hashFunction           = cacheHasher
	artifactConfigFunction = artifactConfig
)

func getHashForArtifact(ctx context.Context, depLister DependencyLister, a *latest.Artifact, mode config.RunMode) (string, error) {
	var inputs []string

	// Append the artifact's configuration
	config, err := artifactConfigFunction(a)
	if err != nil {
		return "", fmt.Errorf("getting artifact's configuration for %q: %w", a.ImageName, err)
	}
	inputs = append(inputs, config)

	// Append the digest of each input file
	deps, err := depLister(ctx, a)
	if err != nil {
		return "", fmt.Errorf("getting dependencies for %q: %w", a.ImageName, err)
	}
	sort.Strings(deps)

	for _, d := range deps {
		h, err := hashFunction(d)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Tracef("skipping dependency for artifact cache calculation, file not found %s: %s", d, err)
				continue // Ignore files that don't exist
			}

			return "", fmt.Errorf("getting hash for %q: %w", d, err)
		}
		inputs = append(inputs, h)
	}

	// add build args for the artifact if specified
	args, err := hashBuildArgs(a, mode)
	if err != nil {
		return "", fmt.Errorf("hashing build args: %w", err)
	}
	if args != nil {
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

// TODO(dgageot): when the buildpacks builder image digest changes, we need to change the hash
func artifactConfig(a *latest.Artifact) (string, error) {
	buf, err := json.Marshal(a.ArtifactType)
	if err != nil {
		return "", fmt.Errorf("marshalling the artifact's configuration for %q: %w", a.ImageName, err)
	}
	return string(buf), nil
}

func hashBuildArgs(artifact *latest.Artifact, mode config.RunMode) ([]string, error) {
	// only one of args or env is ever populated
	var args map[string]*string
	var env map[string]string
	var err error
	switch {
	case artifact.DockerArtifact != nil:
		args, err = docker.EvalBuildArgs(mode, artifact.Workspace, artifact.DockerArtifact)
	case artifact.KanikoArtifact != nil:
		args, err = util.EvaluateEnvTemplateMap(artifact.KanikoArtifact.BuildArgs)
	case artifact.BuildpackArtifact != nil:
		env, err = buildpacks.GetEnv(artifact, mode)
	case artifact.CustomArtifact != nil && artifact.CustomArtifact.Dependencies.Dockerfile != nil:
		args, err = util.EvaluateEnvTemplateMap(artifact.CustomArtifact.Dependencies.Dockerfile.BuildArgs)
	default:
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sl []string
	if args != nil {
		sl = util.EnvPtrMapToSlice(args, "=")
	}
	if env != nil {
		sl = util.EnvMapToSlice(env, "=")
	}
	return sl, nil
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
