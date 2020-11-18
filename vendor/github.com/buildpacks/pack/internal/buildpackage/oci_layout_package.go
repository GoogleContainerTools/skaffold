package buildpackage

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"path"
	"strings"

	"github.com/docker/docker/pkg/ioutils"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/archive"
	blob2 "github.com/buildpacks/pack/internal/blob"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/style"
)

// IsOCILayoutBlob checks whether a blob is in OCI layout format.
func IsOCILayoutBlob(blob blob2.Blob) (bool, error) {
	readCloser, err := blob.Open()
	if err != nil {
		return false, err
	}
	defer readCloser.Close()

	_, _, err = archive.ReadTarEntry(readCloser, "/oci-layout")
	if err != nil {
		if archive.IsEntryNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// BuildpackFromOCILayoutBlob constructs buildpacks from a blob in OCI layout format.
func BuildpacksFromOCILayoutBlob(blob dist.Blob) (mainBP dist.Buildpack, dependencies []dist.Buildpack, err error) {
	layoutPackage, err := newOCILayoutPackage(blob)
	if err != nil {
		return nil, nil, err
	}

	return ExtractBuildpacks(layoutPackage)
}

func ConfigFromOCILayoutBlob(blob dist.Blob) (config v1.ImageConfig, err error) {
	layoutPackage, err := newOCILayoutPackage(blob)
	if err != nil {
		return v1.ImageConfig{}, err
	}
	return layoutPackage.imageInfo.Config, nil
}

type ociLayoutPackage struct {
	imageInfo v1.Image
	manifest  v1.Manifest
	blob      dist.Blob
}

func newOCILayoutPackage(blob dist.Blob) (*ociLayoutPackage, error) {
	index := &v1.Index{}

	if err := unmarshalJSONFromBlob(blob, "/index.json", index); err != nil {
		return nil, err
	}

	var manifestDescriptor *v1.Descriptor
	for _, m := range index.Manifests {
		if m.MediaType == "application/vnd.docker.distribution.manifest.v2+json" {
			manifestDescriptor = &m // nolint:scopelint
			break
		}
	}

	if manifestDescriptor == nil {
		return nil, errors.New("unable to find manifest")
	}

	manifest := &v1.Manifest{}
	if err := unmarshalJSONFromBlob(blob, pathFromDescriptor(*manifestDescriptor), manifest); err != nil {
		return nil, err
	}

	imageInfo := &v1.Image{}
	if err := unmarshalJSONFromBlob(blob, pathFromDescriptor(manifest.Config), imageInfo); err != nil {
		return nil, err
	}

	layersLabel := imageInfo.Config.Labels[dist.BuildpackLayersLabel]
	if layersLabel == "" {
		return nil, errors.Errorf("label %s not found", style.Symbol(dist.BuildpackLayersLabel))
	}

	bpLayers := dist.BuildpackLayers{}
	if err := json.Unmarshal([]byte(layersLabel), &bpLayers); err != nil {
		return nil, errors.Wrap(err, "unmarshaling layers label")
	}

	return &ociLayoutPackage{
		imageInfo: *imageInfo,
		manifest:  *manifest,
		blob:      blob,
	}, nil
}

func (o *ociLayoutPackage) Label(name string) (value string, err error) {
	return o.imageInfo.Config.Labels[name], nil
}

func (o *ociLayoutPackage) GetLayer(diffID string) (io.ReadCloser, error) {
	index := -1
	for i, dID := range o.imageInfo.RootFS.DiffIDs {
		if dID.String() == diffID {
			index = i
			break
		}
	}
	if index == -1 {
		return nil, errors.Errorf("layer %s not found in rootfs", style.Symbol(diffID))
	}

	layerDescriptor := o.manifest.Layers[index]
	layerPath := pathFromDescriptor(layerDescriptor)

	blobReader, err := o.blob.Open()
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(blobReader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to get next tar entry")
		}

		if path.Clean(header.Name) == path.Clean(layerPath) {
			finalReader := blobReader

			if strings.HasSuffix(layerDescriptor.MediaType, ".gzip") {
				finalReader, err = gzip.NewReader(tr)
				if err != nil {
					return nil, err
				}
			}

			return ioutils.NewReadCloserWrapper(finalReader, func() error {
				if err := finalReader.Close(); err != nil {
					return err
				}

				return blobReader.Close()
			}), nil
		}
	}

	if err := blobReader.Close(); err != nil {
		return nil, err
	}

	return nil, errors.Errorf("layer blob %s not found", style.Symbol(layerPath))
}

func pathFromDescriptor(descriptor v1.Descriptor) string {
	return path.Join("/blobs", descriptor.Digest.Algorithm().String(), descriptor.Digest.Encoded())
}

func unmarshalJSONFromBlob(blob dist.Blob, path string, obj interface{}) error {
	reader, err := blob.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	_, contents, err := archive.ReadTarEntry(reader, path)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(contents, obj); err != nil {
		return err
	}

	return nil
}
