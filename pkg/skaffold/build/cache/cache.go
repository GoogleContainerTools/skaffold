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
	"io/ioutil"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

// ImageDetails holds the Digest and ID of an image
type ImageDetails struct {
	Digest string `yaml:"digest,omitempty"`
	ID     string `yaml:"id,omitempty"`
}

// ArtifactCache is a map of [artifact dependencies hash : ImageDetails]
type ArtifactCache map[string]ImageDetails

// cache holds any data necessary for accessing the cache
type cache struct {
	artifactCache      ArtifactCache
	client             docker.LocalDaemon
	insecureRegistries map[string]bool
	cacheFile          string
	imagesAreLocal     bool
	hashForArtifact    func(ctx context.Context, a *latest.Artifact) (string, error)
}

// DependencyLister fetches a list of dependencies for an artifact
type DependencyLister func(ctx context.Context, artifact *latest.Artifact) ([]string, error)

type Config interface {
	docker.Config

	CacheArtifacts() bool
	CacheFile() string
	Mode() config.RunMode
}

// NewCache returns the current state of the cache
func NewCache(cfg Config, imagesAreLocal bool, dependencies DependencyLister) (Cache, error) {
	if !cfg.CacheArtifacts() {
		return &noCache{}, nil
	}

	cacheFile, err := resolveCacheFile(cfg.CacheFile())
	if err != nil {
		logrus.Warnf("Error resolving cache file, not using skaffold cache: %v", err)
		return &noCache{}, nil
	}

	artifactCache, err := retrieveArtifactCache(cacheFile)
	if err != nil {
		logrus.Warnf("Error retrieving artifact cache, not using skaffold cache: %v", err)
		return &noCache{}, nil
	}

	client, err := docker.NewAPIClient(cfg)
	if imagesAreLocal && err != nil {
		return nil, fmt.Errorf("getting local Docker client: %w", err)
	}

	return &cache{
		artifactCache:      artifactCache,
		client:             client,
		insecureRegistries: cfg.GetInsecureRegistries(),
		cacheFile:          cacheFile,
		imagesAreLocal:     imagesAreLocal,
		hashForArtifact: func(ctx context.Context, a *latest.Artifact) (string, error) {
			return getHashForArtifact(ctx, dependencies, a, cfg.Mode())
		},
	}, nil
}

// resolveCacheFile makes sure that either a passed in cache file or the default cache file exists
func resolveCacheFile(cacheFile string) (string, error) {
	if cacheFile != "" {
		return cacheFile, util.VerifyOrCreateFile(cacheFile)
	}
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("retrieving home directory: %w", err)
	}
	defaultFile := filepath.Join(home, constants.DefaultSkaffoldDir, constants.DefaultCacheFile)
	return defaultFile, util.VerifyOrCreateFile(defaultFile)
}

func retrieveArtifactCache(cacheFile string) (ArtifactCache, error) {
	cache := ArtifactCache{}
	contents, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(contents, &cache); err != nil {
		return nil, err
	}
	return cache, nil
}

func saveArtifactCache(cacheFile string, contents ArtifactCache) error {
	data, err := yaml.Marshal(contents)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cacheFile, data, 0755)
}
