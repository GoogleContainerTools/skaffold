/*
Copyright 2022 The Skaffold Authors

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

	"github.com/containerd/containerd/platforms"
	spec "github.com/opencontainers/image-spec/specs-go/v1"
	v1 "k8s.io/api/core/v1"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

const (
	tolerationField = "tolerations"
)

var gkeARMToleration = v1.Toleration{
	Key:      "kubernetes.io/arch",
	Operator: v1.TolerationOpEqual,
	Value:    "arm64",
	Effect:   v1.TaintEffectNoSchedule,
}

// SetGKEARMToleration adds a toleration for GKE ARM node taint `kubernetes.io/arch="arm64":NoSchedule`.
func (l *ManifestList) SetGKEARMToleration(ctx context.Context, rs ResourceSelector, platforms PodPlatforms) (ManifestList, error) {
	r := &gkeTolerationSetter{ctx: ctx, platforms: platforms}
	return l.Visit(r, rs)
}

type gkeTolerationSetter struct {
	ctx       context.Context
	platforms PodPlatforms
}

func (s *gkeTolerationSetter) Visit(gk apimachinery.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	if _, ok := rs.allowByNavpath(gk, navpath, k); !ok {
		return true
	}
	if len(s.platforms) == 0 {
		return false
	}
	pls, ok := s.platforms[navpath]
	if !ok || len(pls) == 0 {
		return true
	}
	matcher := platforms.Any(pls...)
	if !matcher.Match(spec.Platform{OS: "linux", Architecture: "arm64"}) {
		return false
	}

	spec, ok := v.(map[string]interface{})
	if !ok {
		return true
	}

	tolerations, err := addGKEARMToleration(spec[tolerationField])
	if err != nil {
		log.Entry(s.ctx).Debugf("failed to update spec tolerations: %s", err.Error())
		return true
	}
	spec[tolerationField] = tolerations
	return false
}

func addGKEARMToleration(data interface{}) ([]interface{}, error) {
	var tolerations []v1.Toleration
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &tolerations); err != nil {
			return nil, err
		}
	}
	exists := false
	for _, t := range tolerations {
		if t == gkeARMToleration {
			exists = true
			break
		}
	}
	if !exists {
		tolerations = append(tolerations, gkeARMToleration)
	}
	bytes, err := json.Marshal(tolerations)
	if err != nil {
		return nil, err
	}

	var sl []interface{}
	if err := yaml.Unmarshal(bytes, &sl); err != nil {
		return nil, err
	}
	return sl, err
}
