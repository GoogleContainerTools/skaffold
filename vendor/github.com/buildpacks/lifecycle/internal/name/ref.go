package name

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// ParseMaybe attempts to parse the provided reference as a GGCR `name.Reference`, returning a modified `reference.Name()` if parsing is successful.
// Unlike GGCR's `reference.Name()`, `ParseMaybe` will strip the digest portion of the reference,
// retaining the provided tag or adding a `latest` tag if no tag is provided.
// This is to aid in comparing two references when we really care about image names and not about image digests,
// such as when populating `files.RunImageForRebase` information on an exported image.
func ParseMaybe(provided string) string {
	toParse := provided
	if hasDigest(provided) {
		toParse = trimDigest(provided)
	}
	if ref, err := name.ParseReference(toParse); err == nil {
		return ref.Name()
	}
	return provided
}

func hasDigest(ref string) bool {
	return strings.Contains(ref, "@sha256:")
}

func trimDigest(ref string) string {
	parts := strings.Split(ref, "@")
	return parts[0]
}
