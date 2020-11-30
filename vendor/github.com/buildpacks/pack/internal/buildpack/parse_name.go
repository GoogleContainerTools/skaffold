package buildpack

import (
	"fmt"
	"strings"
)

func ParseLocator(locator string) (id string) {
	return ParseRegistryLocator(ParseBuilderLocator(ParsePackageLocator(locator)))
}

// ParseIDLocator parses a buildpack locator of the form <id>@<version> into its ID and version.
// If version is omitted, the version returned will be empty. Any "from=builder:" or "urn:cnb" prefix will be ignored.
func ParseIDLocator(locator string) (id string, version string) {
	nakedLocator := ParseLocator(locator)

	parts := strings.Split(nakedLocator, "@")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

func ParseRegistryID(registryID string) (namespace string, name string, version string, err error) {
	id, version := ParseIDLocator(registryID)

	parts := strings.Split(id, "/")
	if len(parts) == 2 {
		return parts[0], parts[1], version, nil
	}
	return parts[0], "", version, fmt.Errorf("invalid registry ID: %s", registryID)
}

func ParseRegistryLocator(locator string) (path string) {
	return strings.TrimPrefix(locator, fromRegistryPrefix+":")
}

func ParseBuilderLocator(locator string) (path string) {
	return strings.TrimPrefix(
		strings.TrimPrefix(locator, deprecatedFromBuilderPrefix+":"),
		fromBuilderPrefix+":")
}

func ParsePackageLocator(locator string) (path string) {
	return strings.TrimPrefix(
		strings.TrimPrefix(
			strings.TrimPrefix(locator, fromDockerPrefix+"//"),
			fromDockerPrefix+"/"),
		fromDockerPrefix)
}
