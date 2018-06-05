/*
Copyright 2018 The Skaffold Authors

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
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/bazel"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// DependencyMap is a bijection between artifacts and the files they depend on.
type DependencyMap struct {
	artifacts       []*v1alpha2.Artifact
	pathToArtifacts map[string][]*v1alpha2.Artifact
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

func (d *DependencyMap) ArtifactsForPaths(paths []string) []*v1alpha2.Artifact {
	m := map[*v1alpha2.Artifact]struct{}{}
	for _, p := range paths {
		artifacts := d.pathToArtifacts[p]
		for _, a := range artifacts {
			m[a] = struct{}{}
		}
	}
	artifacts := []*v1alpha2.Artifact{}
	for a := range m {
		artifacts = append(artifacts, a)
	}
	return artifacts
}

func NewDependencyMap(artifacts []*v1alpha2.Artifact) (*DependencyMap, error) {
	m, err := pathToArtifactMap(artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "generating path to artifact map")
	}

	return NewExplicitDependencyMap(artifacts, m), nil
}

func NewExplicitDependencyMap(artifacts []*v1alpha2.Artifact, pathToArtifacts map[string][]*v1alpha2.Artifact) *DependencyMap {
	return &DependencyMap{
		artifacts:       artifacts,
		pathToArtifacts: pathToArtifacts,
	}
}

func pathToArtifactMap(artifacts []*v1alpha2.Artifact) (map[string][]*v1alpha2.Artifact, error) {
	m := make(map[string][]*v1alpha2.Artifact)

	for _, a := range artifacts {
		deps, err := DependenciesForArtifact(a)
		if err != nil {
			return nil, errors.Wrapf(err, "getting dependencies for artifact %s", a.ImageName)
		}
		logrus.Infof("Source code dependencies %s: %s", a.ImageName, deps)

		for _, dep := range deps {
			//TODO(r2d4): what does the ignore workspace look like for bazel?
			ignored, err := isIgnored(dep)
			if err != nil {
				return nil, errors.Wrapf(err, "calculating ignored files for artifact %s", a.ImageName)
			}

			if ignored {
				logrus.Debugf("Ignoring %s for artifact dependencies", dep)
				continue
			}

			path := filepath.Join(a.Workspace, dep)
			m[path] = append(m[path], a)
		}
	}

	return m, nil
}

// DependenciesForArtifact is used in tests.
var DependenciesForArtifact = dependenciesForArtifact

func dependenciesForArtifact(a *v1alpha2.Artifact) ([]string, error) {
	if a.DockerArtifact != nil {
		return docker.GetDependencies(a.DockerArtifact.DockerfilePath, a.Workspace)
	}
	if a.BazelArtifact != nil {
		return bazel.GetDependencies(a)
	}

	return nil, fmt.Errorf("undefined artifact type: %+v", a.ArtifactType)
}

func isIgnored(path string) (bool, error) {
	for _, ignoredPrefix := range ignoredPrefixes {
		if strings.HasPrefix(path, ignoredPrefix) {
			return true, nil
		}
	}

	return false, nil
}
