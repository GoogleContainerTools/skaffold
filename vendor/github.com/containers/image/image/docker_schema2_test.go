package image

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/containers/image/docker/reference"
	"github.com/containers/image/manifest"
	"github.com/containers/image/types"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unusedImageSource is used when we don't expect the ImageSource to be used in our tests.
type unusedImageSource struct{}

func (f unusedImageSource) Reference() types.ImageReference {
	panic("Unexpected call to a mock function")
}
func (f unusedImageSource) Close() error {
	panic("Unexpected call to a mock function")
}
func (f unusedImageSource) GetManifest(*digest.Digest) ([]byte, string, error) {
	panic("Unexpected call to a mock function")
}
func (f unusedImageSource) GetBlob(info types.BlobInfo) (io.ReadCloser, int64, error) {
	panic("Unexpected call to a mock function")
}
func (f unusedImageSource) GetSignatures(context.Context, *digest.Digest) ([][]byte, error) {
	panic("Unexpected call to a mock function")
}
func (f unusedImageSource) LayerInfosForCopy() ([]types.BlobInfo, error) {
	panic("Unexpected call to a mock function")
}

func manifestSchema2FromFixture(t *testing.T, src types.ImageSource, fixture string) genericManifest {
	manifest, err := ioutil.ReadFile(filepath.Join("fixtures", fixture))
	require.NoError(t, err)

	m, err := manifestSchema2FromManifest(src, manifest)
	require.NoError(t, err)
	return m
}

func manifestSchema2FromComponentsLikeFixture(configBlob []byte) genericManifest {
	return manifestSchema2FromComponents(manifest.Schema2Descriptor{
		MediaType: "application/octet-stream",
		Size:      5940,
		Digest:    "sha256:9ca4bda0a6b3727a6ffcc43e981cad0f24e2ec79d338f6ba325b4dfd0756fb8f",
	}, nil, configBlob, []manifest.Schema2Descriptor{
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:6a5a5368e0c2d3e5909184fa28ddfd56072e7ff3ee9a945876f7eee5896ef5bb",
			Size:      51354364,
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:1bbf5d58d24c47512e234a5623474acf65ae00d4d1414272a893204f44cc680c",
			Size:      150,
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:8f5dc8a4b12c307ac84de90cdd9a7f3915d1be04c9388868ca118831099c67a9",
			Size:      11739507,
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:bbd6b22eb11afce63cc76f6bc41042d99f10d6024c96b655dafba930b8d25909",
			Size:      8841833,
		},
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Digest:    "sha256:960e52ecf8200cbd84e70eb2ad8678f4367e50d14357021872c10fa3fc5935fa",
			Size:      291,
		},
	})
}

func TestManifestSchema2FromManifest(t *testing.T) {
	// This just tests that the JSON can be loaded; we test that the parsed
	// values are correctly returned in tests for the individual getter methods.
	_ = manifestSchema2FromFixture(t, unusedImageSource{}, "schema2.json")

	_, err := manifestSchema2FromManifest(nil, []byte{})
	assert.Error(t, err)
}

func TestManifestSchema2FromComponents(t *testing.T) {
	// This just smoke-tests that the manifest can be created; we test that the parsed
	// values are correctly returned in tests for the individual getter methods.
	_ = manifestSchema2FromComponentsLikeFixture(nil)
}

func TestManifestSchema2Serialize(t *testing.T) {
	for _, m := range []genericManifest{
		manifestSchema2FromFixture(t, unusedImageSource{}, "schema2.json"),
		manifestSchema2FromComponentsLikeFixture(nil),
	} {
		serialized, err := m.serialize()
		require.NoError(t, err)
		var contents map[string]interface{}
		err = json.Unmarshal(serialized, &contents)
		require.NoError(t, err)

		original, err := ioutil.ReadFile("fixtures/schema2.json")
		require.NoError(t, err)
		var originalContents map[string]interface{}
		err = json.Unmarshal(original, &originalContents)
		require.NoError(t, err)

		// We would ideally like to compare “serialized” with some transformation of
		// “original”, but the ordering of fields in JSON maps is undefined, so this is
		// easier.
		assert.Equal(t, originalContents, contents)
	}
}

func TestManifestSchema2ManifestMIMEType(t *testing.T) {
	for _, m := range []genericManifest{
		manifestSchema2FromFixture(t, unusedImageSource{}, "schema2.json"),
		manifestSchema2FromComponentsLikeFixture(nil),
	} {
		assert.Equal(t, manifest.DockerV2Schema2MediaType, m.manifestMIMEType())
	}
}

func TestManifestSchema2ConfigInfo(t *testing.T) {
	for _, m := range []genericManifest{
		manifestSchema2FromFixture(t, unusedImageSource{}, "schema2.json"),
		manifestSchema2FromComponentsLikeFixture(nil),
	} {
		assert.Equal(t, types.BlobInfo{
			Size:   5940,
			Digest: "sha256:9ca4bda0a6b3727a6ffcc43e981cad0f24e2ec79d338f6ba325b4dfd0756fb8f",
		}, m.ConfigInfo())
	}
}

// configBlobImageSource allows testing various GetBlob behaviors in .ConfigBlob()
type configBlobImageSource struct {
	unusedImageSource // We inherit almost all of the methods, which just panic()
	f                 func(digest digest.Digest) (io.ReadCloser, int64, error)
}

func (f configBlobImageSource) GetBlob(info types.BlobInfo) (io.ReadCloser, int64, error) {
	if info.Digest.String() != "sha256:9ca4bda0a6b3727a6ffcc43e981cad0f24e2ec79d338f6ba325b4dfd0756fb8f" {
		panic("Unexpected digest in GetBlob")
	}
	return f.f(info.Digest)
}

func TestManifestSchema2ConfigBlob(t *testing.T) {
	realConfigJSON, err := ioutil.ReadFile("fixtures/schema2-config.json")
	require.NoError(t, err)

	for _, c := range []struct {
		cbISfn func(digest digest.Digest) (io.ReadCloser, int64, error)
		blob   []byte
	}{
		// Success
		{func(digest digest.Digest) (io.ReadCloser, int64, error) {
			return ioutil.NopCloser(bytes.NewReader(realConfigJSON)), int64(len(realConfigJSON)), nil
		}, realConfigJSON},
		// Various kinds of failures
		{nil, nil},
		{func(digest digest.Digest) (io.ReadCloser, int64, error) {
			return nil, -1, errors.New("Error returned from GetBlob")
		}, nil},
		{func(digest digest.Digest) (io.ReadCloser, int64, error) {
			reader, writer := io.Pipe()
			writer.CloseWithError(errors.New("Expected error reading input in ConfigBlob"))
			return reader, 1, nil
		}, nil},
		{func(digest digest.Digest) (io.ReadCloser, int64, error) {
			nonmatchingJSON := []byte("This does not match ConfigDescriptor.Digest")
			return ioutil.NopCloser(bytes.NewReader(nonmatchingJSON)), int64(len(nonmatchingJSON)), nil
		}, nil},
	} {
		var src types.ImageSource
		if c.cbISfn != nil {
			src = configBlobImageSource{unusedImageSource{}, c.cbISfn}
		} else {
			src = nil
		}
		m := manifestSchema2FromFixture(t, src, "schema2.json")
		blob, err := m.ConfigBlob()
		if c.blob != nil {
			assert.NoError(t, err)
			assert.Equal(t, c.blob, blob)
		} else {
			assert.Error(t, err)
		}
	}

	// Generally conficBlob should match ConfigInfo; we don’t quite need it to, and this will
	// guarantee that the returned object is returning the original contents instead
	// of reading an object from elsewhere.
	configBlob := []byte("config blob which does not match ConfigInfo")
	// This just tests that the manifest can be created; we test that the parsed
	// values are correctly returned in tests for the individual getter methods.
	m := manifestSchema2FromComponentsLikeFixture(configBlob)
	cb, err := m.ConfigBlob()
	require.NoError(t, err)
	assert.Equal(t, configBlob, cb)
}

func TestManifestSchema2LayerInfo(t *testing.T) {
	for _, m := range []genericManifest{
		manifestSchema2FromFixture(t, unusedImageSource{}, "schema2.json"),
		manifestSchema2FromComponentsLikeFixture(nil),
	} {
		assert.Equal(t, []types.BlobInfo{
			{
				Digest: "sha256:6a5a5368e0c2d3e5909184fa28ddfd56072e7ff3ee9a945876f7eee5896ef5bb",
				Size:   51354364,
			},
			{
				Digest: "sha256:1bbf5d58d24c47512e234a5623474acf65ae00d4d1414272a893204f44cc680c",
				Size:   150,
			},
			{
				Digest: "sha256:8f5dc8a4b12c307ac84de90cdd9a7f3915d1be04c9388868ca118831099c67a9",
				Size:   11739507,
			},
			{
				Digest: "sha256:bbd6b22eb11afce63cc76f6bc41042d99f10d6024c96b655dafba930b8d25909",
				Size:   8841833,
			},
			{
				Digest: "sha256:960e52ecf8200cbd84e70eb2ad8678f4367e50d14357021872c10fa3fc5935fa",
				Size:   291,
			},
		}, m.LayerInfos())
	}
}

func TestManifestSchema2EmbeddedDockerReferenceConflicts(t *testing.T) {
	for _, m := range []genericManifest{
		manifestSchema2FromFixture(t, unusedImageSource{}, "schema2.json"),
		manifestSchema2FromComponentsLikeFixture(nil),
	} {
		for _, name := range []string{"busybox", "example.com:5555/ns/repo:tag"} {
			ref, err := reference.ParseNormalizedNamed(name)
			require.NoError(t, err)
			conflicts := m.EmbeddedDockerReferenceConflicts(ref)
			assert.False(t, conflicts)
		}
	}
}

func TestManifestSchema2ImageInspectInfo(t *testing.T) {
	configJSON, err := ioutil.ReadFile("fixtures/schema2-config.json")
	require.NoError(t, err)

	m := manifestSchema2FromComponentsLikeFixture(configJSON)
	ii, err := m.imageInspectInfo()
	require.NoError(t, err)
	assert.Equal(t, types.ImageInspectInfo{
		Tag:           "",
		Created:       time.Date(2016, 9, 23, 23, 20, 45, 789764590, time.UTC),
		DockerVersion: "1.12.1",
		Labels:        map[string]string{},
		Architecture:  "amd64",
		Os:            "linux",
		Layers: []string{
			"sha256:6a5a5368e0c2d3e5909184fa28ddfd56072e7ff3ee9a945876f7eee5896ef5bb",
			"sha256:1bbf5d58d24c47512e234a5623474acf65ae00d4d1414272a893204f44cc680c",
			"sha256:8f5dc8a4b12c307ac84de90cdd9a7f3915d1be04c9388868ca118831099c67a9",
			"sha256:bbd6b22eb11afce63cc76f6bc41042d99f10d6024c96b655dafba930b8d25909",
			"sha256:960e52ecf8200cbd84e70eb2ad8678f4367e50d14357021872c10fa3fc5935fa",
		},
	}, *ii)

	// nil configBlob will trigger an error in m.ConfigBlob()
	m = manifestSchema2FromComponentsLikeFixture(nil)
	_, err = m.imageInspectInfo()
	assert.Error(t, err)

	m = manifestSchema2FromComponentsLikeFixture([]byte("invalid JSON"))
	_, err = m.imageInspectInfo()
	assert.Error(t, err)
}

func TestManifestSchema2UpdatedImageNeedsLayerDiffIDs(t *testing.T) {
	for _, m := range []genericManifest{
		manifestSchema2FromFixture(t, unusedImageSource{}, "schema2.json"),
		manifestSchema2FromComponentsLikeFixture(nil),
	} {
		assert.False(t, m.UpdatedImageNeedsLayerDiffIDs(types.ManifestUpdateOptions{
			ManifestMIMEType: manifest.DockerV2Schema1SignedMediaType,
		}))
	}
}

// schema2ImageSource is plausible enough for schema conversions in manifestSchema2.UpdatedImage() to work.
type schema2ImageSource struct {
	configBlobImageSource
	ref reference.Named
}

func (s2is *schema2ImageSource) Reference() types.ImageReference {
	return refImageReferenceMock{s2is.ref}
}

// refImageReferenceMock is a mock of types.ImageReference which returns itself in DockerReference.
type refImageReferenceMock struct{ reference.Named }

func (ref refImageReferenceMock) Transport() types.ImageTransport {
	panic("unexpected call to a mock function")
}
func (ref refImageReferenceMock) StringWithinTransport() string {
	panic("unexpected call to a mock function")
}
func (ref refImageReferenceMock) DockerReference() reference.Named {
	return ref.Named
}
func (ref refImageReferenceMock) PolicyConfigurationIdentity() string {
	panic("unexpected call to a mock function")
}
func (ref refImageReferenceMock) PolicyConfigurationNamespaces() []string {
	panic("unexpected call to a mock function")
}
func (ref refImageReferenceMock) NewImage(ctx *types.SystemContext) (types.ImageCloser, error) {
	panic("unexpected call to a mock function")
}
func (ref refImageReferenceMock) NewImageSource(ctx *types.SystemContext) (types.ImageSource, error) {
	panic("unexpected call to a mock function")
}
func (ref refImageReferenceMock) NewImageDestination(ctx *types.SystemContext) (types.ImageDestination, error) {
	panic("unexpected call to a mock function")
}
func (ref refImageReferenceMock) DeleteImage(ctx *types.SystemContext) error {
	panic("unexpected call to a mock function")
}

func newSchema2ImageSource(t *testing.T, dockerRef string) *schema2ImageSource {
	realConfigJSON, err := ioutil.ReadFile("fixtures/schema2-config.json")
	require.NoError(t, err)

	ref, err := reference.ParseNormalizedNamed(dockerRef)
	require.NoError(t, err)

	return &schema2ImageSource{
		configBlobImageSource: configBlobImageSource{
			f: func(digest digest.Digest) (io.ReadCloser, int64, error) {
				return ioutil.NopCloser(bytes.NewReader(realConfigJSON)), int64(len(realConfigJSON)), nil
			},
		},
		ref: ref,
	}
}

type memoryImageDest struct {
	ref         reference.Named
	storedBlobs map[digest.Digest][]byte
}

func (d *memoryImageDest) Reference() types.ImageReference {
	return refImageReferenceMock{d.ref}
}
func (d *memoryImageDest) Close() error {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) SupportedManifestMIMETypes() []string {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) SupportsSignatures() error {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) ShouldCompressLayers() bool {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) AcceptsForeignLayerURLs() bool {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) MustMatchRuntimeOS() bool {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) PutBlob(stream io.Reader, inputInfo types.BlobInfo) (types.BlobInfo, error) {
	if d.storedBlobs == nil {
		d.storedBlobs = make(map[digest.Digest][]byte)
	}
	if inputInfo.Digest.String() == "" {
		panic("inputInfo.Digest unexpectedly empty")
	}
	contents, err := ioutil.ReadAll(stream)
	if err != nil {
		return types.BlobInfo{}, err
	}
	d.storedBlobs[inputInfo.Digest] = contents
	return types.BlobInfo{Digest: inputInfo.Digest, Size: int64(len(contents))}, nil
}
func (d *memoryImageDest) HasBlob(inputInfo types.BlobInfo) (bool, int64, error) {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) ReapplyBlob(inputInfo types.BlobInfo) (types.BlobInfo, error) {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) PutManifest([]byte) error {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) PutSignatures(signatures [][]byte) error {
	panic("Unexpected call to a mock function")
}
func (d *memoryImageDest) Commit() error {
	panic("Unexpected call to a mock function")
}

func TestManifestSchema2UpdatedImage(t *testing.T) {
	originalSrc := newSchema2ImageSource(t, "httpd:latest")
	original := manifestSchema2FromFixture(t, originalSrc, "schema2.json")

	// LayerInfos:
	layerInfos := append(original.LayerInfos()[1:], original.LayerInfos()[0])
	res, err := original.UpdatedImage(types.ManifestUpdateOptions{
		LayerInfos: layerInfos,
	})
	require.NoError(t, err)
	assert.Equal(t, layerInfos, res.LayerInfos())
	_, err = original.UpdatedImage(types.ManifestUpdateOptions{
		LayerInfos: append(layerInfos, layerInfos[0]),
	})
	assert.Error(t, err)

	// EmbeddedDockerReference:
	// … is ignored
	embeddedRef, err := reference.ParseNormalizedNamed("busybox")
	require.NoError(t, err)
	res, err = original.UpdatedImage(types.ManifestUpdateOptions{
		EmbeddedDockerReference: embeddedRef,
	})
	require.NoError(t, err)
	nonEmbeddedRef, err := reference.ParseNormalizedNamed("notbusybox:notlatest")
	require.NoError(t, err)
	conflicts := res.EmbeddedDockerReferenceConflicts(nonEmbeddedRef)
	assert.False(t, conflicts)

	// ManifestMIMEType:
	// Only smoke-test the valid conversions, detailed tests are below. (This also verifies that “original” is not affected.)
	for _, mime := range []string{
		manifest.DockerV2Schema1MediaType,
		manifest.DockerV2Schema1SignedMediaType,
	} {
		_, err = original.UpdatedImage(types.ManifestUpdateOptions{
			ManifestMIMEType: mime,
			InformationOnly: types.ManifestUpdateInformation{
				Destination: &memoryImageDest{ref: originalSrc.ref},
			},
		})
		assert.NoError(t, err, mime)
	}
	for _, mime := range []string{
		manifest.DockerV2Schema2MediaType, // This indicates a confused caller, not a no-op
		"this is invalid",
	} {
		_, err = original.UpdatedImage(types.ManifestUpdateOptions{
			ManifestMIMEType: mime,
		})
		assert.Error(t, err, mime)
	}

	// m hasn’t been changed:
	m2 := manifestSchema2FromFixture(t, originalSrc, "schema2.json")
	typedOriginal, ok := original.(*manifestSchema2)
	require.True(t, ok)
	typedM2, ok := m2.(*manifestSchema2)
	require.True(t, ok)
	assert.Equal(t, *typedM2, *typedOriginal)
}

func TestConvertToManifestOCI(t *testing.T) {
	originalSrc := newSchema2ImageSource(t, "httpd-copy:latest")
	original := manifestSchema2FromFixture(t, originalSrc, "schema2.json")
	res, err := original.UpdatedImage(types.ManifestUpdateOptions{
		ManifestMIMEType: imgspecv1.MediaTypeImageManifest,
	})
	require.NoError(t, err)

	convertedJSON, mt, err := res.Manifest()
	require.NoError(t, err)
	assert.Equal(t, imgspecv1.MediaTypeImageManifest, mt)

	byHandJSON, err := ioutil.ReadFile("fixtures/schema2-to-oci1.json")
	require.NoError(t, err)
	var converted, byHand map[string]interface{}
	err = json.Unmarshal(byHandJSON, &byHand)
	require.NoError(t, err)
	err = json.Unmarshal(convertedJSON, &converted)
	require.NoError(t, err)
	assert.Equal(t, byHand, converted)
}

func TestConvertToManifestSchema1(t *testing.T) {
	originalSrc := newSchema2ImageSource(t, "httpd-copy:latest")
	original := manifestSchema2FromFixture(t, originalSrc, "schema2.json")
	memoryDest := &memoryImageDest{ref: originalSrc.ref}
	res, err := original.UpdatedImage(types.ManifestUpdateOptions{
		ManifestMIMEType: manifest.DockerV2Schema1SignedMediaType,
		InformationOnly: types.ManifestUpdateInformation{
			Destination: memoryDest,
		},
	})
	require.NoError(t, err)

	convertedJSON, mt, err := res.Manifest()
	require.NoError(t, err)
	assert.Equal(t, manifest.DockerV2Schema1SignedMediaType, mt)

	// byDockerJSON is the result of asking the Docker Hub for a schema1 manifest,
	// except that we have replaced "name" to verify that the ref from
	// memoryDest, not from originalSrc, is used.
	byDockerJSON, err := ioutil.ReadFile("fixtures/schema2-to-schema1-by-docker.json")
	require.NoError(t, err)
	var converted, byDocker map[string]interface{}
	err = json.Unmarshal(byDockerJSON, &byDocker)
	require.NoError(t, err)
	err = json.Unmarshal(convertedJSON, &converted)
	require.NoError(t, err)
	delete(byDocker, "signatures")
	delete(converted, "signatures")
	assert.Equal(t, byDocker, converted)

	assert.Equal(t, gzippedEmptyLayer, memoryDest.storedBlobs[gzippedEmptyLayerDigest])

	// FIXME? Test also the various failure cases, if only to see that we don't crash?
}
