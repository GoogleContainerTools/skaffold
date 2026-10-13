package layers

import (
	"archive/tar"
	"io"
	"path/filepath"

	"github.com/buildpacks/lifecycle/archive"
)

// Extract extracts an OCI layer from r. Layer entries carry absolute paths, so files
// land at their recorded location. confineTo restricts extraction to descendants of
// confineTo: any entry outside it is rejected. An empty confineTo confines to "/"
// (no effective confinement).
func Extract(r io.Reader, confineTo string) error {
	root := confineTo
	if root == "" {
		// Intentional documented fallback: no confinement. Every lifecycle caller
		// passes a non-empty LayersDir, so this branch is not reached in practice.
		root = `/`
	}
	root = filepath.Clean(root)
	tr := tarReader(r)
	return archive.Extract(tr, root)
}

func tarReader(r io.Reader) archive.TarReader {
	tr := archive.NewNormalizingTarReader(tar.NewReader(r))
	tr.PrependDir(`/`) // no-op for the absolute paths OCI layers carry
	return tr
}
