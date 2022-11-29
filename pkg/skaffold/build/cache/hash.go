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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/kaniko"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// For testing
var (
	newArtifactHasherFunc = newArtifactHasher
	fileHasherFunc        = fileHasher
	artifactConfigFunc    = artifactConfig
)

type artifactHasher interface {
	hash(context.Context, *latest.Artifact, platform.Resolver) (string, error)
}

type artifactHasherImpl struct {
	artifacts graph.ArtifactGraph
	lister    DependencyLister
	mode      config.RunMode
	syncStore *util.SyncStore[string]
}

// newArtifactHasher returns a new instance of an artifactHasher. Use newArtifactHasherFunc instead of calling this function directly.
func newArtifactHasher(artifacts graph.ArtifactGraph, lister DependencyLister, mode config.RunMode) artifactHasher {
	return &artifactHasherImpl{
		artifacts: artifacts,
		lister:    lister,
		mode:      mode,
		syncStore: util.NewSyncStore[string](),
	}
}

func (h *artifactHasherImpl) hash(ctx context.Context, a *latest.Artifact, platforms platform.Resolver) (string, error) {
	ctx, endTrace := instrumentation.StartTrace(ctx, "hash_GenerateHashOneArtifact", map[string]string{
		"ImageName": instrumentation.PII(a.ImageName),
	})
	defer endTrace()

	hash, err := h.safeHash(ctx, a, platforms.GetPlatforms(a.ImageName))
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return "", err
	}
	hashes := []string{hash}
	for _, dep := range sortedDependencies(a, h.artifacts) {
		depHash, err := h.hash(ctx, dep, platforms)
		if err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return "", err
		}
		hashes = append(hashes, depHash)
	}

	if len(hashes) == 1 {
		return hashes[0], nil
	}
	return encode(hashes)
}

func (h *artifactHasherImpl) safeHash(ctx context.Context, a *latest.Artifact, platforms platform.Matcher) (string, error) {
	return h.syncStore.Exec(a.ImageName,
		func() (string, error) {
			return singleArtifactHash(ctx, h.lister, a, h.mode, platforms)
		})
}

// singleArtifactHash calculates the hash for a single artifact, and ignores its required artifacts.
func singleArtifactHash(ctx context.Context, depLister DependencyLister, a *latest.Artifact, mode config.RunMode, m platform.Matcher) (string, error) {
	var inputs []string

	// Append the artifact's configuration
	config, err := artifactConfigFunc(a)
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
		h, err := fileHasherFunc(d)
		if err != nil {
			if os.IsNotExist(err) {
				log.Entry(ctx).Tracef("skipping dependency for artifact cache calculation, file not found %s: %s", d, err)
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

	// add build platforms
	var ps []string
	for _, p := range m.Platforms {
		ps = append(ps, platform.Format(p))
	}
	sort.Strings(ps)
	inputs = append(inputs, ps...)

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
		args, err = docker.EvalBuildArgs(mode, artifact.Workspace, artifact.DockerArtifact.DockerfilePath, artifact.DockerArtifact.BuildArgs, nil)
	case artifact.KanikoArtifact != nil:
		args, err = docker.EvalBuildArgs(mode, kaniko.GetContext(artifact.KanikoArtifact, artifact.Workspace), artifact.KanikoArtifact.DockerfilePath, artifact.KanikoArtifact.BuildArgs, nil)
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

// sortedDependencies returns the dependencies' corresponding Artifacts as sorted by their image name.
func sortedDependencies(a *latest.Artifact, artifacts graph.ArtifactGraph) []*latest.Artifact {
	sl := artifacts.Dependencies(a)
	sort.Slice(sl, func(i, j int) bool {
		ia, ja := sl[i], sl[j]
		return ia.ImageName < ja.ImageName
	})
	return sl
}
