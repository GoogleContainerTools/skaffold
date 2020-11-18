package buildpack

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
)

type LocatorType int

const (
	InvalidLocator = iota
	FromBuilderLocator
	URILocator
	IDLocator
	PackageLocator
	RegistryLocator
)

const fromBuilderPrefix = "urn:cnb:builder"
const deprecatedFromBuilderPrefix = "from=builder"
const fromRegistryPrefix = "urn:cnb:registry"
const fromDockerPrefix = "docker:/"

func (l LocatorType) String() string {
	return []string{
		"InvalidLocator",
		"FromBuilderLocator",
		"URILocator",
		"IDLocator",
		"PackageLocator",
	}[l]
}

// GetLocatorType determines which type of locator is designated by the given input.
// If a type cannot be determined, `INVALID_LOCATOR` will be returned. If an error
// is encountered, it will be returned.
func GetLocatorType(locator string, buildpacksFromBuilder []dist.BuildpackInfo) (LocatorType, error) {
	if locator == deprecatedFromBuilderPrefix {
		return FromBuilderLocator, nil
	}

	if strings.HasPrefix(locator, fromBuilderPrefix+":") || strings.HasPrefix(locator, deprecatedFromBuilderPrefix+":") {
		if !builderMatchFound(locator, buildpacksFromBuilder) {
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

	return parseNakedLocator(locator, buildpacksFromBuilder), nil
}

func HasDockerLocator(locator string) bool {
	return strings.HasPrefix(locator, fromDockerPrefix)
}

func builderMatchFound(locator string, candidates []dist.BuildpackInfo) bool {
	id, version := ParseIDLocator(locator)
	for _, c := range candidates {
		if id == c.ID && (version == "" || version == c.Version) {
			return true
		}
	}
	return false
}

func hasHostPortPrefix(locator string) bool {
	if strings.Contains(locator, "/") {
		prefix := strings.Split(locator, "/")[0]
		if strings.Contains(prefix, ":") {
			return true
		}
	}
	return false
}

func parseNakedLocator(locator string, buildpacksFromBuilder []dist.BuildpackInfo) LocatorType {
	// from here on, we're dealing with a naked locator, and we try to figure out what it is. To do this we check
	// the following characteristics in order:
	//   1. Does it match a path on the file system
	//   2. Does it match a buildpack ID in the builder
	//   3. Does it look like a Docker ref
	//   4. Does it look like a Buildpack Registry ID

	if _, err := os.Stat(locator); err == nil {
		return URILocator
	}

	if builderMatchFound(locator, buildpacksFromBuilder) {
		return IDLocator
	}

	if hasHostPortPrefix(locator) || strings.Contains(locator, "@sha") || strings.Count(locator, "/") > 1 {
		if _, err := name.ParseReference(locator); err == nil {
			return PackageLocator
		}
	}

	if strings.Count(locator, "/") == 1 {
		return RegistryLocator
	}

	return InvalidLocator
}
