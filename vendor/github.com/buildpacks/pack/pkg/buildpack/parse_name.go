package buildpack

import (
	"fmt"
	"strings"
)

// ParseIDLocator parses a buildpack locator in the following formats into its ID and version.
//
//   - <id>[@<version>]
//   - urn:cnb:builder:<id>[@<version>]
//   - urn:cnb:registry:<id>[@<version>]
//   - from=builder:<id>[@<version>] (deprecated)
//
// If version is omitted, the version returned will be empty. Any "from=builder:" or "urn:cnb" prefix will be ignored.
func ParseIDLocator(locator string) (id string, version string) {
	nakedLocator := parseRegistryLocator(parseBuilderLocator(locator))

	parts := strings.Split(nakedLocator, "@")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// ParsePackageLocator parses a locator (in format `[docker://][<host>/]<path>[:<tag>⏐@<digest>]`) to image name (`[<host>/]<path>[:<tag>⏐@<digest>]`)
func ParsePackageLocator(locator string) (imageName string) {
	return strings.TrimPrefix(
		strings.TrimPrefix(
			strings.TrimPrefix(locator, fromDockerPrefix+"//"),
			fromDockerPrefix+"/"),
		fromDockerPrefix)
}

// ParseRegistryID parses a registry id (ie. `<namespace>/<name>@<version>`) into namespace, name and version components.
//
// Supported formats:
//   - <ns>/<name>[@<version>]
//   - urn:cnb:registry:<ns>/<name>[@<version>]
//
func ParseRegistryID(registryID string) (namespace string, name string, version string, err error) {
	id, version := ParseIDLocator(registryID)

	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid registry ID: %s", registryID)
	}

	return parts[0], parts[1], version, nil
}

func parseRegistryLocator(locator string) (path string) {
	return strings.TrimPrefix(locator, fromRegistryPrefix+":")
}

func parseBuilderLocator(locator string) (path string) {
	return strings.TrimPrefix(
		strings.TrimPrefix(locator, deprecatedFromBuilderPrefix+":"),
		fromBuilderPrefix+":")
}
