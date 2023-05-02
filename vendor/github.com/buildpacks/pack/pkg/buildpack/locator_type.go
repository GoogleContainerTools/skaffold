package buildpack

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/dist"
)

type LocatorType int

const (
	InvalidLocator LocatorType = iota
	FromBuilderLocator
	URILocator
	IDLocator
	PackageLocator
	RegistryLocator
	// added entries here should also be added to `String()`
)

const (
	fromBuilderPrefix           = "urn:cnb:builder"
	deprecatedFromBuilderPrefix = "from=builder"
	fromRegistryPrefix          = "urn:cnb:registry"
	fromDockerPrefix            = "docker:/"
)

var (
	// https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
	semverPattern   = `(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`
	registryPattern = regexp.MustCompile(`^[a-z0-9\-\.]+\/[a-z0-9\-\.]+(?:@` + semverPattern + `)?$`)
)

func (l LocatorType) String() string {
	return []string{
		"InvalidLocator",
		"FromBuilderLocator",
		"URILocator",
		"IDLocator",
		"PackageLocator",
		"RegistryLocator",
	}[l]
}

// GetLocatorType determines which type of locator is designated by the given input.
// If a type cannot be determined, `INVALID_LOCATOR` will be returned. If an error
// is encountered, it will be returned.
func GetLocatorType(locator string, relativeBaseDir string, buildpacksFromBuilder []dist.BuildpackInfo) (LocatorType, error) {
	if locator == deprecatedFromBuilderPrefix {
		return FromBuilderLocator, nil
	}

	if strings.HasPrefix(locator, fromBuilderPrefix+":") || strings.HasPrefix(locator, deprecatedFromBuilderPrefix+":") {
		if !isFoundInBuilder(locator, buildpacksFromBuilder) {
			return InvalidLocator, fmt.Errorf("%s is not a valid identifier", style.Symbol(locator))
		}
		return IDLocator, nil
	}

	if strings.HasPrefix(locator, fromRegistryPrefix+":") {
		return RegistryLocator, nil
	}

	if paths.IsURI(locator) {
		if HasDockerLocator(locator) {
			if _, err := name.ParseReference(locator); err == nil {
				return PackageLocator, nil
			}
		}
		return URILocator, nil
	}

	return parseNakedLocator(locator, relativeBaseDir, buildpacksFromBuilder), nil
}

func HasDockerLocator(locator string) bool {
	return strings.HasPrefix(locator, fromDockerPrefix)
}

func parseNakedLocator(locator, relativeBaseDir string, buildpacksFromBuilder []dist.BuildpackInfo) LocatorType {
	// from here on, we're dealing with a naked locator, and we try to figure out what it is. To do this we check
	// the following characteristics in order:
	//   1. Does it match a path on the file system
	//   2. Does it match a buildpack ID in the builder
	//   3. Does it look like a Buildpack Registry ID
	//   4. Does it look like a Docker ref

	if isLocalFile(locator, relativeBaseDir) {
		return URILocator
	}

	if isFoundInBuilder(locator, buildpacksFromBuilder) {
		return IDLocator
	}

	if canBeRegistryRef(locator) {
		return RegistryLocator
	}

	if canBePackageRef(locator) {
		return PackageLocator
	}

	return InvalidLocator
}

func canBePackageRef(locator string) bool {
	if _, err := name.ParseReference(locator); err == nil {
		return true
	}

	return false
}

func canBeRegistryRef(locator string) bool {
	return registryPattern.MatchString(locator)
}

func isFoundInBuilder(locator string, candidates []dist.BuildpackInfo) bool {
	id, version := ParseIDLocator(locator)
	for _, c := range candidates {
		if id == c.ID && (version == "" || version == c.Version) {
			return true
		}
	}
	return false
}

func isLocalFile(locator, relativeBaseDir string) bool {
	if !filepath.IsAbs(locator) {
		locator = filepath.Join(relativeBaseDir, locator)
	}

	if _, err := os.Stat(locator); err == nil {
		return true
	}

	return false
}
