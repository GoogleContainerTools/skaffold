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
	"fmt"
	"strings"

	"github.com/containerd/containerd/platforms"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
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

// Contains returns if the Matcher contains the other platform
func (m Matcher) Contains(other specs.Platform) bool {
	if m.All {
		return true
	}
	matcher := platforms.Any(m.Platforms...)
	return matcher.Match(other)
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
		if err := isKnownPlatform(platform); err != nil {
			return Matcher{}, UnknownPlatformCLIFlag(p, err)
		}
		sl = append(sl, platform)
	}
	return Matcher{Platforms: sl}, nil
}

func isKnownPlatform(p specs.Platform) error {
	if err := isKnownOS(p.OS); err != nil {
		return err
	}
	if err := isKnownArch(p.Architecture); err != nil {
		return err
	}
	return nil
}

// isKnownOS returns an error if we don't know about the operating system.
// Unexported function copied from "github.com/containerd/containerd/platforms"
func isKnownOS(os string) error {
	switch os {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "windows", "zos":
		return nil
	}
	return fmt.Errorf("unknown operating system %q", os)
}

// isKnownArch returns an error if we don't know about the architecture.
// Unexported function copied from "github.com/containerd/containerd/platforms"
func isKnownArch(arch string) error {
	switch arch {
	case "386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "ppc64", "ppc64le", "mips", "mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le", "ppc", "riscv", "riscv64", "s390", "s390x", "sparc", "sparc64", "wasm":
		return nil
	}
	return fmt.Errorf("unknown architecture %q", arch)
}

func Format(pl specs.Platform) string { return platforms.Format(pl) }
