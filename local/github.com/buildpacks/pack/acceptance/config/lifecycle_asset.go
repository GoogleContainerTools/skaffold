//go:build acceptance

package config

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/buildpacks/lifecycle/api"

	"github.com/buildpacks/pack/internal/builder"
)

type LifecycleAsset struct {
	path       string
	descriptor builder.LifecycleDescriptor
	image      string
}

func (a AssetManager) NewLifecycleAsset(kind ComboValue) LifecycleAsset {
	return LifecycleAsset{
		path:       a.LifecyclePath(kind),
		descriptor: a.LifecycleDescriptor(kind),
		image:      a.LifecycleImage(kind),
	}
}

func (l *LifecycleAsset) Version() string {
	return l.SemVer().String()
}

func (l *LifecycleAsset) SemVer() *builder.Version {
	return l.descriptor.Info.Version
}

func (l *LifecycleAsset) Identifier() string {
	if l.HasLocation() {
		return l.path
	} else {
		return l.Version()
	}
}

func (l *LifecycleAsset) HasLocation() bool {
	return l.path != ""
}

func (l *LifecycleAsset) EscapedPath() string {
	return strings.ReplaceAll(l.path, `\`, `\\`)
}

func (l *LifecycleAsset) Image() string {
	return l.image
}

func earliestVersion(versions []*api.Version) *api.Version {
	var earliest *api.Version
	for _, version := range versions {
		switch {
		case version == nil:
			continue
		case earliest == nil:
			earliest = version
		case earliest.Compare(version) > 0:
			earliest = version
		}
	}
	return earliest
}

func latestVersion(versions []*api.Version) *api.Version {
	var latest *api.Version
	for _, version := range versions {
		switch {
		case version == nil:
			continue
		case latest == nil:
			latest = version
		case latest.Compare(version) < 0:
			latest = version
		}
	}
	return latest
}
func (l *LifecycleAsset) EarliestBuildpackAPIVersion() string {
	return earliestVersion(l.descriptor.APIs.Buildpack.Supported).String()
}

func (l *LifecycleAsset) EarliestPlatformAPIVersion() string {
	return earliestVersion(l.descriptor.APIs.Platform.Supported).String()
}

func (l *LifecycleAsset) LatestPlatformAPIVersion() string {
	return latestVersion(l.descriptor.APIs.Platform.Supported).String()
}

func (l *LifecycleAsset) OutputForAPIs() (deprecatedBuildpackAPIs, supportedBuildpackAPIs, deprecatedPlatformAPIs, supportedPlatformAPIs string) {
	stringify := func(apiSet builder.APISet) string {
		versions := apiSet.AsStrings()
		if len(versions) == 0 {
			return "(none)"
		}
		return strings.Join(versions, ", ")
	}

	return stringify(l.descriptor.APIs.Buildpack.Deprecated),
		stringify(l.descriptor.APIs.Buildpack.Supported),
		stringify(l.descriptor.APIs.Platform.Deprecated),
		stringify(l.descriptor.APIs.Platform.Supported)
}

func (l *LifecycleAsset) TOMLOutputForAPIs() (deprecatedBuildpacksAPIs,
	supportedBuildpacksAPIs,
	deprectatedPlatformAPIs,
	supportedPlatformAPIS string,
) {
	stringify := func(apiSet builder.APISet) string {
		if len(apiSet) < 1 || apiSet == nil {
			return "[]"
		}

		var quotedAPIs []string
		for _, a := range apiSet {
			quotedAPIs = append(quotedAPIs, fmt.Sprintf("%q", a))
		}

		return fmt.Sprintf("[%s]", strings.Join(quotedAPIs, ", "))
	}

	return stringify(l.descriptor.APIs.Buildpack.Deprecated),
		stringify(l.descriptor.APIs.Buildpack.Supported),
		stringify(l.descriptor.APIs.Platform.Deprecated),
		stringify(l.descriptor.APIs.Platform.Supported)
}

func (l *LifecycleAsset) YAMLOutputForAPIs(baseIndentationWidth int) (deprecatedBuildpacksAPIs,
	supportedBuildpacksAPIs,
	deprectatedPlatformAPIs,
	supportedPlatformAPIS string,
) {
	stringify := func(apiSet builder.APISet, baseIndentationWidth int) string {
		if len(apiSet) < 1 || apiSet == nil {
			return "[]"
		}

		apiIndentation := strings.Repeat(" ", baseIndentationWidth+2)

		var quotedAPIs []string
		for _, a := range apiSet {
			quotedAPIs = append(quotedAPIs, fmt.Sprintf(`%s- %q`, apiIndentation, a))
		}

		return fmt.Sprintf(`
%s`, strings.Join(quotedAPIs, "\n"))
	}

	return stringify(l.descriptor.APIs.Buildpack.Deprecated, baseIndentationWidth),
		stringify(l.descriptor.APIs.Buildpack.Supported, baseIndentationWidth),
		stringify(l.descriptor.APIs.Platform.Deprecated, baseIndentationWidth),
		stringify(l.descriptor.APIs.Platform.Supported, baseIndentationWidth)
}

func (l *LifecycleAsset) JSONOutputForAPIs(baseIndentationWidth int) (
	deprecatedBuildpacksAPIs,
	supportedBuildpacksAPIs,
	deprectatedPlatformAPIs,
	supportedPlatformAPIS string,
) {
	stringify := func(apiSet builder.APISet, baseIndentationWidth int) string {
		if len(apiSet) < 1 {
			if apiSet == nil {
				return "null"
			}
			return "[]"
		}

		apiIndentation := strings.Repeat(" ", baseIndentationWidth+2)

		var quotedAPIs []string
		for _, a := range apiSet {
			quotedAPIs = append(quotedAPIs, fmt.Sprintf(`%s%q`, apiIndentation, a))
		}

		lineEndSeparator := `,
`

		return fmt.Sprintf(`[
%s
%s]`, strings.Join(quotedAPIs, lineEndSeparator), strings.Repeat(" ", baseIndentationWidth))
	}

	return stringify(l.descriptor.APIs.Buildpack.Deprecated, baseIndentationWidth),
		stringify(l.descriptor.APIs.Buildpack.Supported, baseIndentationWidth),
		stringify(l.descriptor.APIs.Platform.Deprecated, baseIndentationWidth),
		stringify(l.descriptor.APIs.Platform.Supported, baseIndentationWidth)
}

type LifecycleFeature int

const (
	CreationTime = iota
	BuildImageExtensions
	RunImageExtensions
)

type LifecycleAssetSupported func(l *LifecycleAsset) bool

func supportsPlatformAPI(version string) LifecycleAssetSupported {
	return func(i *LifecycleAsset) bool {
		for _, platformAPI := range i.descriptor.APIs.Platform.Supported {
			if platformAPI.AtLeast(version) {
				return true
			}
		}
		for _, platformAPI := range i.descriptor.APIs.Platform.Deprecated {
			if platformAPI.AtLeast(version) {
				return true
			}
		}
		return false
	}
}

var lifecycleFeatureTests = map[LifecycleFeature]LifecycleAssetSupported{
	CreationTime:         supportsPlatformAPI("0.9"),
	BuildImageExtensions: supportsPlatformAPI("0.10"),
	RunImageExtensions:   supportsPlatformAPI("0.12"),
}

func (l *LifecycleAsset) SupportsFeature(f LifecycleFeature) bool {
	return lifecycleFeatureTests[f](l)
}

func (l *LifecycleAsset) atLeast074() bool {
	return !l.SemVer().LessThan(semver.MustParse("0.7.4"))
}
