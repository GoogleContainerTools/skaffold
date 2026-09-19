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
	"os"
	"path/filepath"
	"sync"

	"github.com/mitchellh/go-homedir"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
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
	hashByName         map[string]string
	artifactGraph      graph.ArtifactGraph
	artifactStore      build.ArtifactStore
	cacheMutex         sync.RWMutex
	client             docker.LocalDaemon
	cfg                Config
	cacheFile          string
	isLocalImage       func(imageName string) (bool, error)
	importMissingImage func(imageName string) (bool, error)
	lister             DependencyLister
	buildx             bool
}

// DependencyLister fetches a list of dependencies for an artifact
type DependencyLister func(ctx context.Context, artifact *latest.Artifact, tag string) ([]string, error)

type Config interface {
	docker.Config
	PipelineForImage(imageName string) (latest.Pipeline, bool)
	GetPipelines() []latest.Pipeline
	DefaultPipeline() latest.Pipeline
	GetCluster() config.Cluster
	DetectBuildX() bool
	CacheArtifacts() bool
	CacheFile() string
	Mode() config.RunMode
}

// NewCache returns the current state of the cache
func NewCache(ctx context.Context, cfg Config, isLocalImage func(imageName string) (bool, error), dependencies DependencyLister, graph graph.ArtifactGraph, store build.ArtifactStore) (Cache, error) {
	if !cfg.CacheArtifacts() {
		return &noCache{}, nil
	}

	cacheFile, err := resolveCacheFile(cfg.CacheFile())
	if err != nil {
		log.Entry(context.TODO()).Warnf("Error resolving cache file, not using skaffold cache: %v", err)
		return &noCache{}, nil
	}

	artifactCache, err := retrieveArtifactCache(cacheFile)
	if err != nil {
		log.Entry(context.TODO()).Warnf("Error retrieving artifact cache, not using skaffold cache: %v", err)
		return &noCache{}, nil
	}

	hashByName := make(map[string]string)

	client, err := docker.NewAPIClient(ctx, cfg)
	if err != nil {
		// error only if any pipeline is local.
		for _, p := range cfg.GetPipelines() {
			for _, a := range p.Build.Artifacts {
				if local, _ := isLocalImage(a.ImageName); local {
					return nil, fmt.Errorf("getting local Docker client: %w", err)
				}
			}
		}
	}

	importMissingImage := func(imageName string) (bool, error) {
		pipeline, found := cfg.PipelineForImage(imageName)
		if !found {
			pipeline = cfg.DefaultPipeline()
		}

		if pipeline.Build.GoogleCloudBuild != nil || pipeline.Build.Cluster != nil {
			return false, nil
		}
		return pipeline.Build.LocalBuild.TryImportMissing, nil
	}

	// for backward compatibility, extended build capabilities with BuildKit are disabled by default
	buildx := cfg.DetectBuildX() && docker.IsBuildXDetected()

	return &cache{
		artifactCache:      artifactCache,
		hashByName:         hashByName,
		artifactGraph:      graph,
		artifactStore:      store,
		client:             client,
		cfg:                cfg,
		cacheFile:          cacheFile,
		isLocalImage:       isLocalImage,
		importMissingImage: importMissingImage,
		lister:             dependencies,
		buildx:             buildx,
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
	contents, err := os.ReadFile(cacheFile)
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

	return os.WriteFile(cacheFile, data, 0755)
}
