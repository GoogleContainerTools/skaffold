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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/misc"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// For testing
var (
	hashFunction           = cacheHasher
	artifactConfigFunction = artifactConfig
)

func getHashForArtifact(ctx context.Context, depLister DependencyLister, a *latest.Artifact, devMode bool) (string, error) {
	var inputs []string

	// Append the artifact's configuration
	config, err := artifactConfigFunction(a, devMode)
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
	if buildArgs := retrieveBuildArgs(a); buildArgs != nil {
		buildArgs, err := docker.EvaluateBuildArgs(buildArgs)
		if err != nil {
			return "", fmt.Errorf("evaluating build args: %w", err)
		}
		args := convertBuildArgsToStringArray(buildArgs)
		inputs = append(inputs, args...)
	}

	// add env variables for the artifact if specified
	if env := retrieveEnv(a); len(env) > 0 {
		evaluatedEnv, err := misc.EvaluateEnv(env)
		if err != nil {
			return "", fmt.Errorf("evaluating build args: %w", err)
		}
		inputs = append(inputs, evaluatedEnv...)
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
func artifactConfig(a *latest.Artifact, devMode bool) (string, error) {
	buf, err := json.Marshal(a.ArtifactType)
	if err != nil {
		return "", fmt.Errorf("marshalling the artifact's configuration for %q: %w", a.ImageName, err)
	}

	if devMode && a.BuildpackArtifact != nil && a.Sync != nil && a.Sync.Auto != nil {
		return string(buf) + ".DEV", nil
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

func retrieveEnv(artifact *latest.Artifact) []string {
	if artifact.BuildpackArtifact != nil {
		return artifact.BuildpackArtifact.Env
	}
	return nil
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
