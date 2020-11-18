package builder

import (
	"github.com/BurntSushi/toml"
	"github.com/buildpacks/lifecycle/api"
	"github.com/pkg/errors"
)

// LifecycleDescriptor contains information described in the lifecycle.toml
type LifecycleDescriptor struct {
	Info LifecycleInfo `toml:"lifecycle"`
	// Deprecated: Use `LifecycleAPIs` instead
	API  LifecycleAPI  `toml:"api"`
	APIs LifecycleAPIs `toml:"apis"`
}

// LifecycleInfo contains information about the lifecycle
type LifecycleInfo struct {
	Version *Version `toml:"version" json:"version" yaml:"version"`
}

// LifecycleAPI describes which API versions the lifecycle satisfies
type LifecycleAPI struct {
	BuildpackVersion *api.Version `toml:"buildpack" json:"buildpack"`
	PlatformVersion  *api.Version `toml:"platform" json:"platform"`
}

// LifecycleAPIs describes the supported API versions per specification
type LifecycleAPIs struct {
	Buildpack APIVersions `toml:"buildpack" json:"buildpack"`
	Platform  APIVersions `toml:"platform" json:"platform"`
}

type APISet []*api.Version

func (a APISet) search(comp func(prevMatch, value *api.Version) bool) *api.Version {
	var match *api.Version
	for _, version := range a {
		switch {
		case version == nil:
			continue
		case match == nil:
			match = version
		case comp(match, version):
			match = version
		}
	}

	return match
}

func (a APISet) Earliest() *api.Version {
	return a.search(func(prevMatch, value *api.Version) bool { return value.Compare(prevMatch) < 0 })
}

func (a APISet) Latest() *api.Version {
	return a.search(func(prevMatch, value *api.Version) bool { return value.Compare(prevMatch) > 0 })
}

func (a APISet) AsStrings() []string {
	verStrings := make([]string, len(a))
	for i, version := range a {
		verStrings[i] = version.String()
	}

	return verStrings
}

// APIVersions describes the supported API versions
type APIVersions struct {
	Deprecated APISet `toml:"deprecated" json:"deprecated" yaml:"deprecated"`
	Supported  APISet `toml:"supported" json:"supported" yaml:"supported"`
}

// ParseDescriptor parses LifecycleDescriptor from toml formatted string.
func ParseDescriptor(contents string) (LifecycleDescriptor, error) {
	descriptor := LifecycleDescriptor{}
	_, err := toml.Decode(contents, &descriptor)
	if err != nil {
		return descriptor, errors.Wrap(err, "decoding descriptor")
	}

	return descriptor, nil
}

// CompatDescriptor provides compatibility by mapping new fields to old and vice-versa
func CompatDescriptor(descriptor LifecycleDescriptor) LifecycleDescriptor {
	if len(descriptor.APIs.Buildpack.Supported) != 0 || len(descriptor.APIs.Platform.Supported) != 0 {
		// select earliest value for deprecated parameters
		if len(descriptor.APIs.Buildpack.Supported) != 0 {
			descriptor.API.BuildpackVersion =
				append(descriptor.APIs.Buildpack.Deprecated, descriptor.APIs.Buildpack.Supported...).Earliest()
		}
		if len(descriptor.APIs.Platform.Supported) != 0 {
			descriptor.API.PlatformVersion =
				append(descriptor.APIs.Platform.Deprecated, descriptor.APIs.Platform.Supported...).Earliest()
		}
	} else if descriptor.API.BuildpackVersion != nil && descriptor.API.PlatformVersion != nil {
		// fill supported with deprecated field
		descriptor.APIs = LifecycleAPIs{
			Buildpack: APIVersions{
				Supported: APISet{descriptor.API.BuildpackVersion},
			},
			Platform: APIVersions{
				Supported: APISet{descriptor.API.PlatformVersion},
			},
		}
	}

	return descriptor
}
