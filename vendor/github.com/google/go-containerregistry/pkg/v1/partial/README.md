# `partial`

[![GoDoc](https://godoc.org/github.com/google/go-containerregistry/pkg/v1/partial?status.svg)](https://godoc.org/github.com/google/go-containerregistry/pkg/v1/partial)

## Partial Implementations

There are roughly two kinds of image representations: compressed and uncompressed.

The implementations for these kinds of images are almost identical, with the only
major difference being how blobs (config and layers) are fetched. This common
code lives in this package, where you provide a _partial_ implementation of a
compressed or uncompressed image, and you get back a full `v1.Image` implementation.

### Examples

In a registry, blobs are compressed, so it's easiest to implement a `v1.Image` in terms
of compressed layers. `remote.remoteImage` does this by implementing `CompressedImageCore`:

```go
type CompressedImageCore interface {
	RawConfigFile() ([]byte, error)
	MediaType() (types.MediaType, error)
	RawManifest() ([]byte, error)
	LayerByDigest(v1.Hash) (CompressedLayer, error)
}
```

In a tarball, blobs are (often) uncompressed, so it's easiest to implement a `v1.Image` in terms
of uncompressed layers. `tarball.uncompressedImage` does this by implementing `UncompressedImageCore`:

```go
type CompressedImageCore interface {
	RawConfigFile() ([]byte, error)
	MediaType() (types.MediaType, error)
	LayerByDiffID(v1.Hash) (UncompressedLayer, error)
}
```

## Optional Methods

Where possible, we access some information via optional methods as an optimization.

### [`partial.Descriptor`](https://godoc.org/github.com/google/go-containerregistry/pkg/v1/partial#Descriptor)

There are some properties of a [`Descriptor`](https://github.com/opencontainers/image-spec/blob/master/descriptor.md#properties) that aren't derivable from just image data:

* `MediaType`
* `Platform`
* `URLs`
* `Annotations`

For example, in a `tarball.Image`, there is a `LayerSources` field that contains
an entire layer descriptor with `URLs` information for foreign layers. This
information can be passed through to callers by implementing this optional
`Descriptor` method.

See [`#654`](https://github.com/google/go-containerregistry/pull/654).

### [`partial.UncompressedSize`](https://godoc.org/github.com/google/go-containerregistry/pkg/v1/partial#UncompressedSize)

Usually, you don't need to know the uncompressed size of a layer, since that
information isn't stored in a config file (just he sha256 is needed); however,
there are cases where it is very helpful to know the layer size, e.g. when
writing the uncompressed layer into a tarball.

See [`#655`](https://github.com/google/go-containerregistry/pull/655).
