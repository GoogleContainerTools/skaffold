/*
Copyright 2018 Google LLC

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

package build

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type DependencyMap struct {
	artifacts       []*config.Artifact
	pathToArtifacts map[string][]*config.Artifact
}

type DependencyResolver interface {
	GetDependencies(a *config.Artifact) ([]string, error)
}

//TODO(@r2d4): Figure out best UX to support configuring this blacklist
var ignoredPrefixes = []string{"vendor", ".git"}

func (d *DependencyMap) Paths() []string {
	allPaths := []string{}
	for path := range d.pathToArtifacts {
		allPaths = append(allPaths, path)
	}
	sort.Strings(allPaths)
	return allPaths
}

func (d *DependencyMap) ArtifactsForPaths(paths []string) []*config.Artifact {
	m := map[*config.Artifact]struct{}{}
	for _, p := range paths {
		artifacts := d.pathToArtifacts[p]
		for _, a := range artifacts {
			m[a] = struct{}{}
		}
	}
	artifacts := []*config.Artifact{}
	for a := range m {
		artifacts = append(artifacts, a)
	}
	return artifacts
}

func NewDependencyMap(artifacts []*config.Artifact) (*DependencyMap, error) {
	m, err := pathToArtifactMap(artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "generating path to artifact map")
	}
	return &DependencyMap{
		artifacts:       artifacts,
		pathToArtifacts: m,
	}, nil
}

// path must be an absolute path
func isIgnored(workspace, path string) (bool, error) {
	for _, ig := range ignoredPrefixes {
		ignoredPrefix, err := filepath.Abs(filepath.Join(workspace, ig))
		if err != nil {
			return false, errors.Wrapf(err, "calculating absolute path of ignored dep %s", ig)
		}

		if strings.HasPrefix(path, ignoredPrefix) {
			logrus.Debugf("Ignoring %s for artifact dependencies", path)
			return true, nil
		}
	}

	return false, nil
}

func pathToArtifactMap(artifacts []*config.Artifact) (map[string][]*config.Artifact, error) {
	m := map[string][]*config.Artifact{}
	for _, a := range artifacts {
		paths, err := pathsForArtifact(a)
		if err != nil {
			return nil, errors.Wrapf(err, "getting paths for artifact %s", a.ImageName)
		}

		for _, p := range paths {
			m[p] = append(m[p], a)
		}
	}

	return m, nil
}

func pathsForArtifact(a *config.Artifact) ([]string, error) {
	deps, err := GetDependenciesForArtifact(a)
	if err != nil {
		return nil, errors.Wrap(err, "getting dockerfile dependencies")
	}
	filteredDeps := []string{}
	for _, dep := range deps {
		//TODO(r2d4): what does the ignore workspace look like for bazel?
		ignored, err := isIgnored(a.DockerArtifact.Workspace, dep)
		if err != nil {
			return nil, errors.Wrapf(err, "calculating ignored files for artifact %s", a.ImageName)
		}
		if ignored {
			continue
		}
		filteredDeps = append(filteredDeps, dep)
	}
	return filteredDeps, nil
}

func GetDependenciesForArtifact(a *config.Artifact) ([]string, error) {
	if a.DockerArtifact != nil {
		return docker.DefaultDockerfileDepResolver.GetDependencies(a)
	}
	if a.BazelArtifact != nil {
		b := &BazelDependencyResolver{}
		return b.GetDependencies(a)
	}
	return nil, errors.New("unknown artifact type")
}
