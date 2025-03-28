package imgutil_test

import (
	"encoding/json"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/imgutil"
	h "github.com/buildpacks/imgutil/testhelpers"
)

func TestUtils(t *testing.T) {
	spec.Run(t, "Utils", testUtils, spec.Parallel(), spec.Report(report.Terminal{}))
}

type FakeIndentifier struct {
	hash string
}

func NewFakeIdentifier(hash string) FakeIndentifier {
	return FakeIndentifier{
		hash: hash,
	}
}

func (f FakeIndentifier) String() string {
	return f.hash
}

func testUtils(t *testing.T, when spec.G, it spec.S) {
	when("#TaggableIndex", func() {
		var (
			taggableIndex *imgutil.TaggableIndex
			amd64Hash, _  = v1.NewHash("sha256:b9d056b83bb6446fee29e89a7fcf10203c562c1f59586a6e2f39c903597bda34")
			armv6Hash, _  = v1.NewHash("sha256:0bcc1b827b855c65eaf6e031e894e682b6170160b8a676e1df7527a19d51fb1a")
			indexManifest = v1.IndexManifest{
				SchemaVersion: 2,
				MediaType:     types.OCIImageIndex,
				Annotations: map[string]string{
					"test-key": "test-value",
				},
				Manifests: []v1.Descriptor{
					{
						MediaType: types.OCIManifestSchema1,
						Size:      832,
						Digest:    amd64Hash,
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						},
					},
					{
						MediaType: types.OCIManifestSchema1,
						Size:      926,
						Digest:    armv6Hash,
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "arm",
							OSVersion:    "v6",
						},
					},
				},
			}
		)
		it.Before(func() {
			taggableIndex = imgutil.NewTaggableIndex(&indexManifest)
		})
		it("should return RawManifest in expected format", func() {
			mfestBytes, err := taggableIndex.RawManifest()
			h.AssertNil(t, err)

			expectedMfestBytes, err := json.Marshal(indexManifest)
			h.AssertNil(t, err)

			h.AssertEq(t, mfestBytes, expectedMfestBytes)
		})
		it("should return expected digest", func() {
			digest, err := taggableIndex.Digest()
			h.AssertNil(t, err)
			h.AssertEq(t, digest.String(), "sha256:2375c0dfd06dd51b313fd97df5ecf3b175380e895287dd9eb2240b13eb0b5703")
		})
		it("should return expected size", func() {
			size, err := taggableIndex.Size()
			h.AssertNil(t, err)
			h.AssertEq(t, size, int64(547))
		})
		it("should return expected media type", func() {
			format, err := taggableIndex.MediaType()
			h.AssertNil(t, err)
			h.AssertEq(t, format, indexManifest.MediaType)
		})
	})

	when("#StringSet", func() {
		when("#NewStringSet", func() {
			it("should return not nil StringSet instance", func() {
				stringSet := imgutil.NewStringSet()
				h.AssertNotNil(t, stringSet)
				h.AssertEq(t, stringSet.StringSlice(), []string(nil))
			})
		})

		when("#Add", func() {
			var (
				stringSet *imgutil.StringSet
			)
			it.Before(func() {
				stringSet = imgutil.NewStringSet()
			})
			it("should add items", func() {
				item := "item1"
				stringSet.Add(item)

				h.AssertEq(t, stringSet.StringSlice(), []string{item})
			})
			it("should return added items", func() {
				items := []string{"item1", "item2", "item3"}
				for _, item := range items {
					stringSet.Add(item)
				}
				h.AssertEq(t, len(stringSet.StringSlice()), 3)
				h.AssertContains(t, stringSet.StringSlice(), items...)
			})
			it("should not support duplicates", func() {
				stringSet := imgutil.NewStringSet()
				item1 := "item1"
				item2 := "item2"
				items := []string{item1, item2, item1}
				for _, item := range items {
					stringSet.Add(item)
				}
				h.AssertEq(t, len(stringSet.StringSlice()), 2)
				h.AssertContains(t, stringSet.StringSlice(), []string{item1, item2}...)
			})
		})

		when("#Remove", func() {
			var (
				stringSet *imgutil.StringSet
				item      string
			)
			it.Before(func() {
				stringSet = imgutil.NewStringSet()
				item = "item1"
				stringSet.Add(item)
				h.AssertEq(t, stringSet.StringSlice(), []string{item})
			})
			it("should remove item", func() {
				stringSet.Remove(item)
				h.AssertEq(t, stringSet.StringSlice(), []string(nil))
			})
		})
	})

	when("#NewEmptyDockerIndex", func() {
		it("should return an empty docker index", func() {
			idx := imgutil.NewEmptyDockerIndex()
			h.AssertNotNil(t, idx)

			digest, err := idx.Digest()
			h.AssertNil(t, err)
			h.AssertNotEq(t, digest, v1.Hash{})

			format, err := idx.MediaType()
			h.AssertNil(t, err)
			h.AssertEq(t, format, types.DockerManifestList)
		})
	})
}
