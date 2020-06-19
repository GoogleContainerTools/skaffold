package buildpack

import (
	"fmt"
	"strings"
)

// ParseIDLocator parses a buildpack locator of the form <id>@<version> into its ID and version.
// If version is omitted, the version returned will be empty. Any "from=builder:" or "urn:cnb" prefix will be ignored.
func ParseIDLocator(locator string) (id string, version string) {
	nakedLocator := strings.TrimPrefix(strings.TrimPrefix(locator, fromBuilderPrefix+":"), fromRegistryPrefix+":")

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
