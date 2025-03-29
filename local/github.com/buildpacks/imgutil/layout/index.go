package layout

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/buildpacks/imgutil"
)

// NewIndex will return an OCI ImageIndex saved on disk using OCI media Types. It can be modified and saved to a registry
func NewIndex(repoName string, ops ...imgutil.IndexOption) (*imgutil.CNBIndex, error) {
	options := &imgutil.IndexOptions{}
	for _, op := range ops {
		if err := op(options); err != nil {
			return nil, err
		}
	}

	var err error

	if options.BaseIndex == nil && options.BaseIndexRepoName != "" { // options.BaseIndex supersedes options.BaseIndexRepoName
		options.BaseIndex, err = newV1Index(
			options.BaseIndexRepoName,
		)
		if err != nil {
			return nil, err
		}
	}

	return imgutil.NewCNBIndex(repoName, *options)
}

// newV1Index creates a layout image index from the given path.
func newV1Index(path string) (v1.ImageIndex, error) {
	if !imageExists(path) {
		return nil, nil
	}
	layoutPath, err := FromPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load layout from path: %w", err)
	}
	return layoutPath.ImageIndex()
}
