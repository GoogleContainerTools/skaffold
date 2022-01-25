package buildah

// Builder is an artifact builder that uses buildah
type Builder struct {
	pushImages bool
}

const (
	gzipCompression  = "gzip"
	bzip2Compression = "bzip2"
	zstdCompression  = "zstd"
	xzCompression    = "xz"
	uncompressed     = "uncompressed"
)

func NewBuilder(pushImages bool) *Builder {

	return &Builder{
		pushImages: pushImages,
	}
}
