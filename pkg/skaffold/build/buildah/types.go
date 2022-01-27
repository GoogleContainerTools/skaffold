package buildah

import "github.com/GoogleContainerTools/skaffold/pkg/skaffold/podman"

// Builder is an artifact builder that uses buildah
type Builder struct {
	pushImages bool
	runtime    *podman.Buildah
}

const (
	gzipCompression  = "gzip"
	bzip2Compression = "bzip2"
	zstdCompression  = "zstd"
	xzCompression    = "xz"
	uncompressed     = "uncompressed"
)

func NewBuilder(runtime *podman.Buildah, pushImages bool) *Builder {
	return &Builder{
		runtime:    runtime,
		pushImages: pushImages,
	}
}
