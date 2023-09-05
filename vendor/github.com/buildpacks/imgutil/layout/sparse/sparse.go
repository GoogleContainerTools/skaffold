package sparse

import (
	"github.com/buildpacks/imgutil/layout"

	"github.com/buildpacks/imgutil"
)

var _ imgutil.Image = (*Image)(nil)

// Image is a struct created to override the Save() method of a layout image,
// so that when the image is saved to disk, it does not include any layers in the `blobs` directory.
type Image struct {
	layout.Image
}
