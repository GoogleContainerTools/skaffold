package layout

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
)

const ImageRefNameKey = "org.opencontainers.image.ref.name"

// ParseRefToPath parse the given image reference to local path directory following the rules:
// An image reference refers to either a tag reference or digest reference.
//   - A tag reference refers to an identifier of form <registry>/<repo>/<image>:<tag>
//   - A digest reference refers to a content addressable identifier of form <registry>/<repo>/<image>@<algorithm>:<digest>
//
// WHEN the image reference points to a tag reference returns <registry>/<repo>/<image>/<tag>
// WHEN the image reference points to a digest reference returns <registry>/<repo>/<image>/<algorithm>/<digest>
func ParseRefToPath(imageRef string) (string, error) {
	reference, err := name.ParseReference(imageRef, name.WeakValidation)
	if err != nil {
		return "", err
	}
	path := filepath.Join(reference.Context().RegistryStr(), reference.Context().RepositoryStr())
	if strings.Contains(reference.Identifier(), ":") {
		splitDigest := strings.Split(reference.Identifier(), ":")
		path = filepath.Join(path, splitDigest[0], splitDigest[1])
	} else {
		path = filepath.Join(path, reference.Identifier())
	}

	return path, nil
}

// ParseIdentifier parse the given identifier string into a layout.Identifier.
// WHEN image.Identifier() method is called, it returns the layout identifier with the format [path]@[digest] this
// method reconstruct a string reference in that format into a layout.Identifier
func ParseIdentifier(identifier string) (Identifier, error) {
	if strings.Contains(identifier, identifierDelim) {
		referenceSplit := strings.Split(identifier, identifierDelim)
		if len(referenceSplit) == 2 {
			hash, err := v1.NewHash(referenceSplit[1])
			if err != nil {
				return Identifier{}, err
			}
			path := path.Clean(referenceSplit[0])
			return newLayoutIdentifier(path, hash)
		}
	}
	return Identifier{}, errors.Errorf("identifier %s does not have the format '[path]%s[digest]'", identifier, identifierDelim)
}

// ImageRefAnnotation creates a map containing the key 'org.opencontainers.image.ref.name' with the provided value.
func ImageRefAnnotation(imageRefName string) map[string]string {
	if imageRefName == "" {
		return nil
	}
	annotations := make(map[string]string, 1)
	annotations[ImageRefNameKey] = imageRefName
	return annotations
}
