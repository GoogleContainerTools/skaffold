package testhelpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/fakes"
	imgutilRemote "github.com/buildpacks/imgutil/remote"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func NewRandomIndexRepoName() string {
	return "test-index-" + RandString(10)
}

func AssertPathExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("Expected %q to exist", path)
	} else if err != nil {
		t.Fatalf("Error stating %q: %v", path, err)
	}
}

func AssertPathDoesNotExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if err == nil {
		t.Errorf("Expected %q to not exists", path)
	}
}

func FetchImageIndexDescriptor(t *testing.T, repoName string) v1.ImageIndex {
	t.Helper()

	r, err := name.ParseReference(repoName, name.WeakValidation)
	AssertNil(t, err)

	auth, err := authn.DefaultKeychain.Resolve(r.Context().Registry)
	AssertNil(t, err)

	index, err := remote.Index(r, remote.WithTransport(http.DefaultTransport), remote.WithAuth(auth))
	AssertNil(t, err)

	return index
}

func AssertRemoteImageIndex(t *testing.T, repoName string, mediaType types.MediaType, expectedNumberOfManifests int) {
	t.Helper()

	remoteIndex := FetchImageIndexDescriptor(t, repoName)
	AssertNotNil(t, remoteIndex)
	remoteIndexMediaType, err := remoteIndex.MediaType()
	AssertNil(t, err)
	AssertEq(t, remoteIndexMediaType, mediaType)
	remoteIndexManifest, err := remoteIndex.IndexManifest()
	AssertNil(t, err)
	AssertNotNil(t, remoteIndexManifest)
	AssertEq(t, len(remoteIndexManifest.Manifests), expectedNumberOfManifests)
}

func CreateRemoteImage(t *testing.T, repoName, tag, baseImage string) *imgutilRemote.Image {
	img1RepoName := fmt.Sprintf("%s:%s", repoName, tag)
	img1, err := imgutilRemote.NewImage(img1RepoName, authn.DefaultKeychain, imgutilRemote.FromBaseImage(baseImage))
	AssertNil(t, err)
	err = img1.Save()
	AssertNil(t, err)
	return img1
}

func ReadIndexManifest(t *testing.T, path string) *v1.IndexManifest {
	t.Helper()

	indexPath := filepath.Join(path, "index.json")
	AssertPathExists(t, filepath.Join(path, "oci-layout"))
	AssertPathExists(t, indexPath)

	// check index file
	data, err := os.ReadFile(indexPath)
	AssertNil(t, err)

	index := &v1.IndexManifest{}
	err = json.Unmarshal(data, index)
	AssertNil(t, err)
	return index
}

func RandomCNBIndex(t *testing.T, repoName string, layers, count int64) *imgutil.CNBIndex {
	t.Helper()

	randomIndex, err := random.Index(1024, layers, count)
	AssertNil(t, err)
	options := &imgutil.IndexOptions{
		BaseIndex: randomIndex,
		LayoutIndexOptions: imgutil.LayoutIndexOptions{
			XdgPath: os.Getenv("XDG_RUNTIME_DIR"),
		},
	}
	idx, err := imgutil.NewCNBIndex(repoName, *options)
	AssertNil(t, err)
	return idx
}

func RandomCNBIndexAndDigest(t *testing.T, repoName string, layers, count int64) (idx imgutil.ImageIndex, digest name.Digest) {
	idx = RandomCNBIndex(t, repoName, layers, count)

	imgIdx, ok := idx.(*imgutil.CNBIndex)
	AssertEq(t, ok, true)

	mfest, err := imgIdx.IndexManifest()
	AssertNil(t, err)

	digest, err = name.NewDigest(fmt.Sprintf("%s@%s", repoName, mfest.Manifests[0].Digest.String()))
	AssertNil(t, err)

	return idx, digest
}

// MockImageIndex wraps a real CNBIndex to record if some key methods are invoke
type MockImageIndex struct {
	imgutil.CNBIndex
	ErrorOnSave     bool
	PushCalled      bool
	PurgeOption     bool
	DeleteDirCalled bool
}

// NewMockImageIndex creates a random index with the given number of layers and manifests count
func NewMockImageIndex(t *testing.T, repoName string, layers, count int64) *MockImageIndex {
	cnbIdx := RandomCNBIndex(t, repoName, layers, count)
	idx := &MockImageIndex{
		CNBIndex: *cnbIdx,
	}
	return idx
}

func (i *MockImageIndex) SaveDir() error {
	if i.ErrorOnSave {
		return errors.New("something failed writing the index on disk")
	}
	return i.CNBIndex.SaveDir()
}

func (i *MockImageIndex) Push(ops ...imgutil.IndexOption) error {
	var pushOps = &imgutil.IndexOptions{}
	for _, op := range ops {
		if err := op(pushOps); err != nil {
			return err
		}
	}

	i.PushCalled = true
	i.PurgeOption = pushOps.Purge
	return nil
}

func (i *MockImageIndex) DeleteDir() error {
	i.DeleteDirCalled = true
	return nil
}

func NewFakeWithRandomUnderlyingV1Image(t *testing.T, repoName string, identifier imgutil.Identifier) *FakeWithRandomUnderlyingImage {
	fakeCNBImage := fakes.NewImage(repoName, "", identifier)
	underlyingImage, err := random.Image(1024, 1)
	AssertNil(t, err)
	return &FakeWithRandomUnderlyingImage{
		Image:           fakeCNBImage,
		underlyingImage: underlyingImage,
	}
}

type FakeWithRandomUnderlyingImage struct {
	*fakes.Image
	underlyingImage v1.Image
}

func (t *FakeWithRandomUnderlyingImage) UnderlyingImage() v1.Image {
	return t.underlyingImage
}

func (t *FakeWithRandomUnderlyingImage) GetLayer(sha string) (io.ReadCloser, error) {
	hash, err := v1.NewHash(sha)
	if err != nil {
		return nil, err
	}

	layer, err := t.UnderlyingImage().LayerByDiffID(hash)
	if err != nil {
		return nil, err
	}
	return layer.Uncompressed()
}
