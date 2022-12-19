/*
Copyright 2021 The Skaffold Authors

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

package manifest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	spec "github.com/opencontainers/image-spec/specs-go/v1"
	v1 "k8s.io/api/core/v1"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

// for testing
var getPlatforms = docker.GetPlatforms

const (
	specField     = "spec"
	affinityField = "affinity"

	// kubernetes.io/arch and kubernetes.io/os are known node labels. See https://kubernetes.io/docs/reference/labels-annotations-taints/
	nodeOperatingSystemLabel = "kubernetes.io/os"
	nodeArchitectureLabel    = "kubernetes.io/arch"
)

type ResourceSelectorPodSpec struct {
	allowlist map[apimachinery.GroupKind]latest.ResourceFilter
	denylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func NewResourceSelectorPodSpec(allowlist map[apimachinery.GroupKind]latest.ResourceFilter, denylist map[apimachinery.GroupKind]latest.ResourceFilter) *ResourceSelectorPodSpec {
	return &ResourceSelectorPodSpec{
		allowlist: allowlist,
		denylist:  denylist,
	}
}

func (rs *ResourceSelectorPodSpec) allowByGroupKind(gk apimachinery.GroupKind) bool {
	if _, allowed := rs.allowlist[gk]; allowed {
		if rf, disallowed := rs.denylist[gk]; disallowed {
			for _, s := range rf.Labels {
				if s == ".*" {
					return false
				}
			}
			for _, s := range rf.Image {
				if s == ".*" {
					return false
				}
			}
			for _, s := range rf.PodSpec {
				if s == ".*" {
					return false
				}
			}
		}
		return true
	}
	return false
}

//nolint:unparam
func (rs *ResourceSelectorPodSpec) allowByNavpath(gk apimachinery.GroupKind, navpath string, k string) (string, bool) {
	for _, w := range ConfigConnectorResourceSelector {
		if w.Matches(gk.Group, gk.Kind) {
			return "", false
		}
	}

	if rf, ok := rs.denylist[gk]; ok {
		for _, denypath := range rf.PodSpec {
			if denypath == ".*" {
				return "", false
			}
			if navpath == denypath {
				return "", false
			}
		}
	}

	if rf, ok := rs.allowlist[gk]; ok {
		for _, allowpath := range rf.PodSpec {
			if allowpath == ".*" {
				if k != specField {
					return "", false
				}
				return "", true
			}
			if navpath == allowpath {
				return "", true
			}
		}
	}
	return "", false
}

// SetPlatformNodeAffinity adds `nodeAffinity` definitions specific to the target platform (os/architecture) for each built image.
func (l *ManifestList) SetPlatformNodeAffinity(ctx context.Context, rs ResourceSelector, platforms PodPlatforms) (ManifestList, error) {
	r := &nodeAffinitySetter{ctx: ctx, platforms: platforms}
	return l.Visit(r, rs)
}

func (l *ManifestList) GetImagePlatforms(ctx context.Context, rs ResourceSelector) (PodPlatforms, error) {
	s := &imagePlatformSaver{ctx: ctx, platforms: make(PodPlatforms)}
	_, err := l.Visit(s, rs)
	return s.platforms, parseImagesInManifestErr(err)
}

// PodPlatforms maps the pod spec json path to the list of common platforms for all containers in that pod.
type PodPlatforms map[string][]spec.Platform

type imagePlatformSaver struct {
	ctx       context.Context
	platforms PodPlatforms
}

func (r *imagePlatformSaver) Visit(gk apimachinery.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	if k != imageField {
		return true
	}
	image, ok := v.(string)
	if !ok {
		return true
	}
	platforms, err := getPlatforms(image)
	if err != nil {
		log.Entry(r.ctx).Debugf("Couldn't get target platforms for image %q: %s", image, err.Error())
		return false
	}
	specPath := strings.TrimSuffix(navpath, ".containers.image")
	if pl, ok := r.platforms[specPath]; ok && len(pl) > 0 {
		// only keep common platforms for all containers in a pod spec.
		// for example, if a pod spec has two containers where the first image is a manifest list supporting `linux/arm64` and `linux/amd64`,
		// and the second image only supports `linux/amd64`, then we only save `linux/amd64` platform for that pod spec.
		r.platforms[specPath] = platform.Matcher{Platforms: pl}.Intersect(platform.Matcher{Platforms: platforms}).Platforms
	} else {
		r.platforms[specPath] = platforms
	}
	return false
}

type nodeAffinitySetter struct {
	ctx       context.Context
	platforms PodPlatforms
}

func (s *nodeAffinitySetter) Visit(gk apimachinery.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	if _, ok := rs.allowByNavpath(gk, navpath, k); !ok {
		return true
	}

	if len(s.platforms) == 0 {
		return false
	}
	platforms, ok := s.platforms[navpath]
	if !ok || len(platforms) == 0 {
		return true
	}

	spec, ok := v.(map[string]interface{})
	if !ok {
		return true
	}
	affinity, err := updateAffinity(spec[affinityField], platforms)
	if err != nil {
		log.Entry(s.ctx).Debugf("failed to update affinity definition: %s", err.Error())
		return true
	}
	spec[affinityField] = affinity
	return false
}

func updateAffinity(data interface{}, platforms []spec.Platform) (map[string]interface{}, error) {
	var affinity v1.Affinity
	if data == nil {
		affinity = v1.Affinity{}
	} else {
		data, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &affinity); err != nil {
			return nil, err
		}
	}

	if affinity.NodeAffinity == nil {
		affinity.NodeAffinity = &v1.NodeAffinity{}
	}
	if affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{}
	}

	for _, term := range affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
		for _, exp := range term.MatchExpressions {
			if exp.Key == nodeOperatingSystemLabel || exp.Key == nodeArchitectureLabel {
				return nil, fmt.Errorf("found existing node affinity for os/arch: %s", exp)
			}
		}
	}

	if len(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) == 0 {
		affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, v1.NodeSelectorTerm{})
	}

	var terms []v1.NodeSelectorTerm
	for _, pl := range platforms {
		for _, term := range affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
			t := term.DeepCopy()
			if pl.OS != "" {
				t.MatchExpressions = append(t.MatchExpressions, v1.NodeSelectorRequirement{
					Key:      nodeOperatingSystemLabel,
					Operator: v1.NodeSelectorOpIn,
					Values:   []string{pl.OS},
				})
			}
			if pl.Architecture != "" {
				t.MatchExpressions = append(t.MatchExpressions, v1.NodeSelectorRequirement{
					Key:      nodeArchitectureLabel,
					Operator: v1.NodeSelectorOpIn,
					Values:   []string{pl.Architecture},
				})
			}
			terms = append(terms, *t)
		}
	}
	affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = terms
	bytes, err := json.Marshal(affinity)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	if err := yaml.Unmarshal(bytes, &m); err != nil {
		return nil, err
	}
	return m, nil
}
