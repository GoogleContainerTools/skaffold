package layout

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
)

var _ imgutil.Image = (*Image)(nil)
var _ imgutil.ImageIndex = (*ImageIndex)(nil)

type Image struct {
	*imgutil.CNBImageCore
	repoPath          string
	saveWithoutLayers bool
	preserveDigest    bool
}

func (i *Image) Kind() string {
	return "layout"
}

func (i *Image) Name() string {
	return i.repoPath
}

func (i *Image) Rename(name string) {
	i.repoPath = name
}

// Found reports if image exists in the image store with `Name()`.
func (i *Image) Found() bool {
	return imageExists(i.repoPath)
}

func imageExists(path string) bool {
	if !pathExists(path) {
		return false
	}
	index := filepath.Join(path, "index.json")
	if _, err := os.Stat(index); os.IsNotExist(err) {
		return false
	}
	return true
}

func pathExists(path string) bool {
	if path != "" {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return true
		}
	}
	return false
}

// Identifier
// Each image's ID is given by the SHA256 hash of its configuration JSON. It is represented as a hexadecimal encoding of 256 bits,
// e.g., sha256:a9561eb1b190625c9adb5a9513e72c4dedafc1cb2d4c5236c9a6957ec7dfd5a9.
func (i *Image) Identifier() (imgutil.Identifier, error) {
	hash, err := i.Image.Digest()
	if err != nil {
		return nil, errors.Wrapf(err, "getting identifier for image at path %q", i.repoPath)
	}
	return newLayoutIdentifier(i.repoPath, hash)
}

func (i *Image) Valid() bool {
	// layout images may be invalid if they are missing layer data
	return true
}

func (i *Image) Delete() error {
	return os.RemoveAll(i.repoPath)
}

type ImageIndex struct {
	*imgutil.CNBIndex
}
