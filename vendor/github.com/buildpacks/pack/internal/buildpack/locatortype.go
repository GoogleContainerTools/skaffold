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

const fromBuilderPrefix = "from=builder"
const fromRegistryPrefix = "urn:cnb:registry"

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
	if locator == fromBuilderPrefix {
		return FromBuilderLocator, nil
	}

	if strings.HasPrefix(locator, fromBuilderPrefix+":") {
		if !builderMatchFound(locator, buildpacksFromBuilder) {
			return InvalidLocator, fmt.Errorf("%s is not a valid identifier", style.Symbol(locator))
		}
		return IDLocator, nil
	}

	if strings.HasPrefix(locator, fromRegistryPrefix+":") {
		return RegistryLocator, nil
	}

	if paths.IsURI(locator) {
		return URILocator, nil
	}

	if _, err := os.Stat(locator); err == nil {
		return URILocator, nil
	}

	if builderMatchFound(locator, buildpacksFromBuilder) {
		return IDLocator, nil
	}

	if _, err := name.ParseReference(locator); err == nil {
		return PackageLocator, nil
	}

	return InvalidLocator, nil
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
