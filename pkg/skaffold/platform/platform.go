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

package platform

import (
	"context"
	"strings"

	"github.com/containerd/containerd/platforms"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

var (
	All  = Matcher{All: true}
	Host = Matcher{Platforms: []specs.Platform{platforms.DefaultSpec()}}
)

// Matcher describes a collection of target image build platforms
type Matcher struct {
	All       bool
	Platforms []specs.Platform
}

func (m Matcher) IsEmpty() bool {
	return !m.All && len(m.Platforms) == 0
}

func (m Matcher) IsNotEmpty() bool {
	return !m.IsEmpty()
}

func (m Matcher) IsMultiPlatform() bool {
	return m.All || len(m.Platforms) > 1
}

func (m Matcher) IsCrossPlatform() bool {
	return m.IsMultiPlatform() || (len(m.Platforms) == 1 && !platforms.Only(m.Platforms[0]).Match(platforms.DefaultSpec()))
}

func (m Matcher) Array() []string {
	var pl []string
	if m.All {
		return append(pl, "all")
	}
	for _, p := range m.Platforms {
		pl = append(pl, Format(p))
	}
	return pl
}

func (m Matcher) String() string {
	return strings.Join(m.Array(), ",")
}

// Intersect returns the intersection set of Matchers.
func (m Matcher) Intersect(other Matcher) Matcher {
	if m.All {
		return other
	}
	if other.All {
		return m
	}
	var pl []specs.Platform
	for i := 0; i < len(m.Platforms); i++ {
		matcher := platforms.OnlyStrict(m.Platforms[i])
		for j := 0; j < len(other.Platforms); j++ {
			if matcher.Match(other.Platforms[j]) {
				pl = append(pl, m.Platforms[i])
			}
		}
	}
	res := Matcher{Platforms: pl}
	log.Entry(context.TODO()).Debugf("intersect matchers %q and %q, result %q", m, other, res)
	return res
}

func Parse(ps []string) (Matcher, error) {
	var sl []specs.Platform
	for _, p := range ps {
		if strings.ToLower(p) == "all" {
			return All, nil
		}
		platform, err := platforms.Parse(p)
		if err != nil {
			return Matcher{}, UnknownPlatformCLIFlag(p, err)
		}
		sl = append(sl, platform)
	}
	return Matcher{Platforms: sl}, nil
}

func Format(pl specs.Platform) string { return platforms.Format(pl) }
