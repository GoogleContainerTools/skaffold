package layout_test

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/layout"
	imgutilRemote "github.com/buildpacks/imgutil/remote"
	h "github.com/buildpacks/imgutil/testhelpers"
)

func TestLayoutIndex(t *testing.T) {
	dockerConfigDir, err := os.MkdirTemp("", "test.docker.config.dir")
	h.AssertNil(t, err)
	defer os.RemoveAll(dockerConfigDir)

	dockerRegistry = h.NewDockerRegistry(h.WithAuth(dockerConfigDir))
	dockerRegistry.Start(t)
	defer dockerRegistry.Stop(t)

	os.Setenv("DOCKER_CONFIG", dockerConfigDir)
	defer os.Unsetenv("DOCKER_CONFIG")

	spec.Run(t, "LayoutNewIndex", testNewIndex, spec.Parallel(), spec.Report(report.Terminal{}))
	spec.Run(t, "LayoutIndex", testIndex, spec.Parallel(), spec.Report(report.Terminal{}))
}

var (
	dockerRegistry *h.DockerRegistry

	// global directory and paths
	testDataDir = filepath.Join("testdata", "layout")
)

func testNewIndex(t *testing.T, when spec.G, it spec.S) {
	var (
		idx      imgutil.ImageIndex
		tempDir  string
		repoName string
		err      error
	)

	it.Before(func() {
		// creates the directory to save all the OCI images on disk
		tempDir, err = os.MkdirTemp("", "image-indexes")
		h.AssertNil(t, err)

		// global directory and paths
		testDataDir = filepath.Join("testdata", "layout")
		_ = idx
	})

	it.After(func() {
		err := os.RemoveAll(tempDir)
		h.AssertNil(t, err)
	})

	when("#NewIndex", func() {
		it.Before(func() {
			repoName = "some/index"
		})

		when("index doesn't exists on disk", func() {
			it("creates empty image index", func() {
				idx, err = layout.NewIndex(
					repoName,
					imgutil.WithXDGRuntimePath(tempDir),
				)
				h.AssertNil(t, err)
			})

			it("ignores FromBaseIndex if it doesn't exist", func() {
				idx, err = layout.NewIndex(
					repoName,
					imgutil.WithXDGRuntimePath(tempDir),
					imgutil.FromBaseIndex("non-existent/index"),
				)
				h.AssertNil(t, err)
			})

			it("creates empty image index with Docker media-types", func() {
				idx, err = layout.NewIndex(
					repoName,
					imgutil.WithXDGRuntimePath(tempDir),
					imgutil.WithMediaType(types.DockerManifestList),
				)
				h.AssertNil(t, err)
			})
		})
	})
}

func testIndex(t *testing.T, when spec.G, it spec.S) {
	var (
		idx           imgutil.ImageIndex
		tmpDir        string
		localPath     string
		baseIndexPath string
		err           error
	)

	it.Before(func() {
		// creates the directory to save all the OCI images on disk
		tmpDir, err = os.MkdirTemp("", "layout-image-indexes")
		h.AssertNil(t, err)

		// image index directory on disk
		baseIndexPath = filepath.Join(testDataDir, "busybox-multi-platform")
		// global directory and paths
		testDataDir = filepath.Join("testdata", "layout")
	})

	it.After(func() {
		err := os.RemoveAll(tmpDir)
		h.AssertNil(t, err)
	})

	when("Getters", func() {
		var (
			attribute   string
			attributes  []string
			annotations map[string]string
			digest      name.Digest
		)
		when("index exists on disk", func() {
			when("#FromBaseIndex", func() {
				it.Before(func() {
					idx, err = layout.NewIndex("busybox-multi-platform", imgutil.WithXDGRuntimePath(tmpDir), imgutil.FromBaseIndex(baseIndexPath))
					h.AssertNil(t, err)
					localPath = filepath.Join(tmpDir, "busybox-multi-platform")
				})

				// See spec: https://github.com/opencontainers/image-spec/blob/main/image-index.md#image-index-property-descriptions
				when("linux/amd64", func() {
					it.Before(func() {
						digest, err = name.NewDigest("busybox-multi-platform@sha256:f5b920213fc6498c0c5eaee7e04f8424202b565bb9e5e4de9e617719fb7bd873")
						h.AssertNil(t, err)
					})

					it("existing platform attributes are readable", func() {
						// #Architecture
						attribute, err = idx.Architecture(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, attribute, "amd64")

						// #OS
						attribute, err = idx.OS(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, attribute, "linux")

						// #Variant
						attribute, err = idx.Variant(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, attribute, "v1")

						// #OSVersion
						attribute, err = idx.OSVersion(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, attribute, "4.5.6")

						// #OSFeatures
						attributes, err = idx.OSFeatures(digest)
						h.AssertNil(t, err)
						h.AssertContains(t, attributes, "os-feature-1", "os-feature-2")
					})

					it("existing annotations are readable", func() {
						annotations, err = idx.Annotations(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, annotations["com.docker.official-images.bashbrew.arch"], "amd64")
						h.AssertEq(t, annotations["org.opencontainers.image.url"], "https://hub.docker.com/_/busybox")
						h.AssertEq(t, annotations["org.opencontainers.image.revision"], "d0b7d566eb4f1fa9933984e6fc04ab11f08f4592")
					})
				})

				when("linux/arm64", func() {
					it.Before(func() {
						digest, err = name.NewDigest("busybox-multi-platform@sha256:e18f2c12bb4ea582045415243370a3d9cf3874265aa2867f21a35e630ebe45a7")
						h.AssertNil(t, err)
					})

					it("existing platform attributes are readable", func() {
						// #Architecture
						attribute, err = idx.Architecture(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, attribute, "arm")

						// #OS
						attribute, err = idx.OS(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, attribute, "linux")

						// #Variant
						attribute, err = idx.Variant(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, attribute, "v7")

						// #OSVersion
						attribute, err = idx.OSVersion(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, attribute, "1.2.3")

						// #OSFeatures
						attributes, err = idx.OSFeatures(digest)
						h.AssertNil(t, err)
						h.AssertContains(t, attributes, "os-feature-3", "os-feature-4")
					})

					it("existing annotations are readable", func() {
						annotations, err = idx.Annotations(digest)
						h.AssertNil(t, err)
						h.AssertEq(t, annotations["com.docker.official-images.bashbrew.arch"], "arm32v7")
						h.AssertEq(t, annotations["org.opencontainers.image.url"], "https://hub.docker.com/_/busybox")
						h.AssertEq(t, annotations["org.opencontainers.image.revision"], "185a3f7f21c307b15ef99b7088b228f004ff5f11")
					})
				})

				when("non-existent digest is provided", func() {
					it.Before(func() {
						// Just changed the last number of a valid digest
						digest, err = name.NewDigest("busybox-multi-platform@sha256:f5b920213fc6498c0c5eaee7e04f8424202b565bb9e5e4de9e617719fb7bd872")
						h.AssertNil(t, err)
					})

					it("error is returned", func() {
						// #Architecture
						attribute, err = idx.Architecture(digest)
						h.AssertNotNil(t, err)

						// #OS
						attribute, err = idx.OS(digest)
						h.AssertNotNil(t, err)

						// #Variant
						attribute, err = idx.Variant(digest)
						h.AssertNotNil(t, err)

						// #OSVersion
						attribute, err = idx.OSVersion(digest)
						h.AssertNotNil(t, err)

						// #OSFeatures
						attributes, err = idx.OSFeatures(digest)
						h.AssertNotNil(t, err)

						// #Annotations
						annotations, err = idx.Annotations(digest)
						h.AssertNotNil(t, err)
					})
				})
			})
		})
	})

	when("#Setters", func() {
		var (
			descriptor1 v1.Descriptor
			digest1     name.Digest
		)

		when("index is created from scratch", func() {
			it.Before(func() {
				repoName := newRepoName()
				idx = setupIndex(t, repoName, imgutil.WithXDGRuntimePath(tmpDir))
				localPath = filepath.Join(tmpDir, repoName)
			})

			when("digest is provided", func() {
				it.Before(func() {
					image1, err := random.Image(1024, 1)
					h.AssertNil(t, err)
					idx.AddManifest(image1)

					h.AssertNil(t, idx.SaveDir())

					index := h.ReadIndexManifest(t, localPath)
					h.AssertEq(t, len(index.Manifests), 1)
					descriptor1 = index.Manifests[0]

					digest1, err = name.NewDigest(fmt.Sprintf("%s@%s", "random", descriptor1.Digest.String()))
					h.AssertNil(t, err)
				})

				it("platform attributes are written on disk", func() {
					h.AssertNil(t, idx.SetOS(digest1, "linux"))
					h.AssertNil(t, idx.SetArchitecture(digest1, "arm"))
					h.AssertNil(t, idx.SetVariant(digest1, "v6"))
					h.AssertNil(t, idx.SaveDir())

					index := h.ReadIndexManifest(t, localPath)
					h.AssertEq(t, len(index.Manifests), 1)
					h.AssertEq(t, index.Manifests[0].Digest.String(), descriptor1.Digest.String())
					h.AssertEq(t, index.Manifests[0].Platform.OS, "linux")
					h.AssertEq(t, index.Manifests[0].Platform.Architecture, "arm")
					h.AssertEq(t, index.Manifests[0].Platform.Variant, "v6")
				})

				it("annotations are written on disk", func() {
					annotations := map[string]string{
						"some-key": "some-value",
					}
					h.AssertNil(t, idx.SetAnnotations(digest1, annotations))
					h.AssertNil(t, idx.SaveDir())

					index := h.ReadIndexManifest(t, localPath)
					h.AssertEq(t, len(index.Manifests), 1)
					h.AssertEq(t, index.Manifests[0].Digest.String(), descriptor1.Digest.String())
					h.AssertEq(t, reflect.DeepEqual(index.Manifests[0].Annotations, annotations), true)
				})
			})
		})

		when("index exists on disk", func() {
			when("#FromBaseIndex", func() {
				when("digest is provided", func() {
					when("attributes already exists", func() {
						when("oci media-type is used", func() {
							it.Before(func() {
								idx = setupIndex(t, "busybox-multi-platform", imgutil.WithXDGRuntimePath(tmpDir), imgutil.FromBaseIndex(baseIndexPath))
								localPath = filepath.Join(tmpDir, "busybox-multi-platform")
								digest1, err = name.NewDigest("busybox@sha256:e18f2c12bb4ea582045415243370a3d9cf3874265aa2867f21a35e630ebe45a7")
								h.AssertNil(t, err)
							})

							it("platform attributes are updated on disk", func() {
								h.AssertNil(t, idx.SetOS(digest1, "linux-2"))
								h.AssertNil(t, idx.SetArchitecture(digest1, "arm-2"))
								h.AssertNil(t, idx.SetVariant(digest1, "v6-2"))
								h.AssertNil(t, idx.SaveDir())

								index := h.ReadIndexManifest(t, localPath)
								h.AssertEq(t, len(index.Manifests), 2)
								h.AssertEq(t, index.Manifests[1].Digest.String(), "sha256:e18f2c12bb4ea582045415243370a3d9cf3874265aa2867f21a35e630ebe45a7")
								h.AssertEq(t, index.Manifests[1].Platform.OS, "linux-2")
								h.AssertEq(t, index.Manifests[1].Platform.Architecture, "arm-2")
								h.AssertEq(t, index.Manifests[1].Platform.Variant, "v6-2")
							})

							it("new annotation are appended on disk", func() {
								annotations := map[string]string{
									"some-key": "some-value",
								}
								h.AssertNil(t, idx.SetAnnotations(digest1, annotations))
								h.AssertNil(t, idx.SaveDir())

								index := h.ReadIndexManifest(t, localPath)
								h.AssertEq(t, len(index.Manifests), 2)

								// When updating a digest, it will be appended at the end
								h.AssertEq(t, index.Manifests[1].Digest.String(), "sha256:e18f2c12bb4ea582045415243370a3d9cf3874265aa2867f21a35e630ebe45a7")

								// in testdata we have 7 annotations + 1 new
								h.AssertEq(t, len(index.Manifests[1].Annotations), 8)
								h.AssertEq(t, index.Manifests[1].Annotations["some-key"], "some-value")
							})
						})

						when("docker media-type is used", func() {
							it.Before(func() {
								baseIndexPath = filepath.Join(testDataDir, "index-with-docker-media-type")
								idx = setupIndex(t, "some-docker-index", imgutil.WithXDGRuntimePath(tmpDir), imgutil.FromBaseIndex(baseIndexPath))
								localPath = filepath.Join(tmpDir, imgutil.MakeFileSafeName("some-docker-index"))
								digest1, err = name.NewDigest("some-docker-manifest@sha256:a564fd8f0684d2e119b73db7fb89280a665ebb18e8c30f26d163b4c0da8a8090")
								h.AssertNil(t, err)
							})

							it("new annotation are appended on disk and media-type is not override", func() {
								annotations := map[string]string{
									"some-key": "some-value",
								}
								h.AssertNil(t, idx.SetAnnotations(digest1, annotations))
								h.AssertNil(t, idx.SaveDir())

								index := h.ReadIndexManifest(t, localPath)
								h.AssertEq(t, len(index.Manifests), 1)
								h.AssertEq(t, index.MediaType, types.DockerManifestList)

								// When updating a digest, it will be appended at the end
								h.AssertEq(t, index.Manifests[0].Digest.String(), "sha256:a564fd8f0684d2e119b73db7fb89280a665ebb18e8c30f26d163b4c0da8a8090")

								// in testdata we have 7 annotations + 1 new
								h.AssertEq(t, len(index.Manifests[0].Annotations), 1)
								h.AssertEq(t, index.Manifests[0].Annotations["some-key"], "some-value")
							})
						})
					})
				})
			})
		})
	})

	when("#Save", func() {
		when("index exists on disk", func() {
			when("#FromBaseIndex", func() {
				it.Before(func() {
					idx, err = layout.NewIndex("busybox-multi-platform", imgutil.WithXDGRuntimePath(tmpDir), imgutil.FromBaseIndex(baseIndexPath))
					h.AssertNil(t, err)

					localPath = filepath.Join(tmpDir, "busybox-multi-platform")
				})

				it("manifests from base image index are saved on disk", func() {
					err = idx.SaveDir()
					h.AssertNil(t, err)

					// assert linux/amd64 and linux/arm64 manifests were saved
					index := h.ReadIndexManifest(t, localPath)
					h.AssertEq(t, len(index.Manifests), 2)
					h.AssertEq(t, index.Manifests[0].Digest.String(), "sha256:f5b920213fc6498c0c5eaee7e04f8424202b565bb9e5e4de9e617719fb7bd873")
					h.AssertEq(t, index.Manifests[1].Digest.String(), "sha256:e18f2c12bb4ea582045415243370a3d9cf3874265aa2867f21a35e630ebe45a7")
				})
			})

			when("#FromBaseIndexInstance", func() {
				it.Before(func() {
					localIndex := h.ReadImageIndex(t, baseIndexPath)

					idx, err = layout.NewIndex("busybox-multi-platform", imgutil.WithXDGRuntimePath(tmpDir), imgutil.FromBaseIndexInstance(localIndex))
					h.AssertNil(t, err)

					localPath = filepath.Join(tmpDir, "busybox-multi-platform")
				})

				it("manifests from base image index instance are saved on disk", func() {
					err = idx.SaveDir()
					h.AssertNil(t, err)

					// assert linux/amd64 and linux/arm64 manifests were saved
					index := h.ReadIndexManifest(t, localPath)
					h.AssertEq(t, len(index.Manifests), 2)
					h.AssertEq(t, index.Manifests[0].Digest.String(), "sha256:f5b920213fc6498c0c5eaee7e04f8424202b565bb9e5e4de9e617719fb7bd873")
					h.AssertEq(t, index.Manifests[1].Digest.String(), "sha256:e18f2c12bb4ea582045415243370a3d9cf3874265aa2867f21a35e630ebe45a7")
				})
			})
		})
	})

	when("#Add", func() {
		var (
			imagePath         string
			fullBaseImagePath string
		)

		it.Before(func() {
			imagePath, err = os.MkdirTemp(tmpDir, "layout-test-image-index")
			h.AssertNil(t, err)

			fullBaseImagePath = filepath.Join(testDataDir, "busybox")
		})

		when("index is created from scratch", func() {
			it.Before(func() {
				repoName := newRepoName()
				idx = setupIndex(t, repoName, imgutil.WithXDGRuntimePath(tmpDir))
				localPath = filepath.Join(tmpDir, repoName)
			})

			when("manifest in OCI layout format is added", func() {
				var editableImage v1.Image
				it.Before(func() {
					editableImage, err = layout.NewImage(imagePath, layout.FromBaseImagePath(fullBaseImagePath))
					h.AssertNil(t, err)
				})

				it("adds one manifest to the index", func() {
					idx.AddManifest(editableImage)
					h.AssertNil(t, idx.SaveDir())
					// manifest was added
					index := h.ReadIndexManifest(t, localPath)
					h.AssertEq(t, len(index.Manifests), 1)
				})

				it("add more than one manifest to the index", func() {
					image1, err := random.Image(1024, 1)
					h.AssertNil(t, err)
					idx.AddManifest(image1)

					image2, err := random.Image(1024, 1)
					h.AssertNil(t, err)
					idx.AddManifest(image2)

					h.AssertNil(t, idx.SaveDir())

					// manifest was added
					index := h.ReadIndexManifest(t, localPath)
					h.AssertEq(t, len(index.Manifests), 2)
				})
			})
		})

		when("index exists on disk", func() {
			when("#FromBaseIndex", func() {
				it.Before(func() {
					idx = setupIndex(t, "busybox-multi-platform", imgutil.WithXDGRuntimePath(tmpDir), imgutil.FromBaseIndex(baseIndexPath))
					localPath = filepath.Join(tmpDir, "busybox-multi-platform")
				})

				when("manifest in OCI layout format is added", func() {
					var editableImage v1.Image
					it.Before(func() {
						editableImage, err = layout.NewImage(imagePath, layout.FromBaseImagePath(fullBaseImagePath))
						h.AssertNil(t, err)
					})

					it("adds the manifest to the index", func() {
						idx.AddManifest(editableImage)
						h.AssertNil(t, idx.SaveDir())
						index := h.ReadIndexManifest(t, localPath)
						// manifest was added
						// initially it has 2 manifest + 1 new
						h.AssertEq(t, len(index.Manifests), 3)
					})
				})
			})
		})
	})

	when("#Push", func() {
		var repoName string

		// Index under test is created with this number of manifests on it
		const expectedNumberOfManifests = 2

		when("index is created from scratch", func() {
			it.Before(func() {
				repoName = newTestImageIndexName("push-index-test")
				idx = setupIndex(t, repoName, imgutil.WithXDGRuntimePath(tmpDir), imgutil.WithKeychain(authn.DefaultKeychain))

				// Note: It will only push IndexManifest, assuming all the images it refers exists in registry
				// We need to push each individual image first

				// Manifest 1
				img1 := createRemoteImage(t, repoName, "busybox-amd64", "busybox@sha256:f5b920213fc6498c0c5eaee7e04f8424202b565bb9e5e4de9e617719fb7bd873")
				idx.AddManifest(img1)

				// Manifest 2
				img2 := createRemoteImage(t, repoName, "busybox-arm64", "busybox@sha256:e18f2c12bb4ea582045415243370a3d9cf3874265aa2867f21a35e630ebe45a7")
				idx.AddManifest(img2)
			})

			when("no options are provided", func() {
				it("index is pushed to the registry", func() {
					err = idx.Push()
					h.AssertNil(t, err)
					h.AssertRemoteImageIndex(t, repoName, types.OCIImageIndex, expectedNumberOfManifests)
				})
			})

			when("#WithMediaType", func() {
				it("index is pushed to the registry using docker media types", func() {
					// By default, OCI media types is used
					err = idx.Push(imgutil.WithMediaType(types.DockerManifestList))
					h.AssertNil(t, err)
					h.AssertRemoteImageIndex(t, repoName, types.DockerManifestList, expectedNumberOfManifests)
				})

				it("error when media-type doesn't refer to an index", func() {
					err = idx.Push(imgutil.WithMediaType(types.DockerConfigJSON))
					h.AssertNotNil(t, err)
				})
			})

			when("#WithTags", func() {
				it("index is pushed to the registry with the additional tag provided", func() {
					// By default, OCI media types is used
					err = idx.Push(imgutil.WithTags("some-cool-tag"))
					h.AssertNil(t, err)
					addionalRepoName := fmt.Sprintf("%s:%s", repoName, "some-cool-tag")
					h.AssertRemoteImageIndex(t, addionalRepoName, types.OCIImageIndex, expectedNumberOfManifests)
				})
			})

			when("#WithPurge", func() {
				it("index is pushed to the registry and remove from local storage", func() {
					// By default, OCI media types is used
					err = idx.Push(imgutil.WithPurge(true))
					h.AssertNil(t, err)
					h.AssertRemoteImageIndex(t, repoName, types.OCIImageIndex, expectedNumberOfManifests)
					h.AssertPathDoesNotExists(t, path.Join(tmpDir, imgutil.MakeFileSafeName(repoName)))
				})
			})
		})
	})

	when("#Delete", func() {
		when("index exists on disk", func() {
			when("#FromBaseIndex", func() {
				it.Before(func() {
					idx = setupIndex(t, "busybox-multi-platform", imgutil.WithXDGRuntimePath(tmpDir), imgutil.FromBaseIndex(baseIndexPath))
					localPath = filepath.Join(tmpDir, "busybox-multi-platform")
				})

				it("deletes the imange index from disk", func() {
					// Verify the index exists
					h.ReadIndexManifest(t, localPath)

					err = idx.DeleteDir()
					h.AssertNil(t, err)

					_, err = os.Stat(localPath)
					h.AssertNotNil(t, err)
					h.AssertEq(t, true, os.IsNotExist(err))
				})
			})
		})
	})

	when("#Remove", func() {
		var digest name.Digest
		when("index exists on disk", func() {
			when("#FromBaseIndex", func() {
				it.Before(func() {
					idx = setupIndex(t, "busybox-multi-platform", imgutil.WithXDGRuntimePath(tmpDir), imgutil.FromBaseIndex(baseIndexPath), imgutil.WithKeychain(authn.DefaultKeychain))
					localPath = filepath.Join(tmpDir, "busybox-multi-platform")
					digest, err = name.NewDigest("busybox@sha256:f5b920213fc6498c0c5eaee7e04f8424202b565bb9e5e4de9e617719fb7bd873")
					h.AssertNil(t, err)
				})

				it("given manifest is removed", func() {
					err = idx.RemoveManifest(digest)
					h.AssertNil(t, err)

					// After removing any operation to get something about the digest must fail
					_, err = idx.OS(digest)
					h.AssertNotNil(t, err)
					h.AssertError(t, err, "failed to find image with digest")

					// After saving, the index on disk must reflect the change
					err = idx.SaveDir()
					h.AssertNil(t, err)

					index := h.ReadIndexManifest(t, localPath)
					h.AssertEq(t, len(index.Manifests), 1)
					h.AssertEq(t, index.Manifests[0].Digest.String(), "sha256:e18f2c12bb4ea582045415243370a3d9cf3874265aa2867f21a35e630ebe45a7")
				})
			})
		})
	})

	when("#Inspect", func() {
		var indexString string
		when("index exists on disk", func() {
			when("#FromBaseIndex", func() {
				it.Before(func() {
					idx = setupIndex(t, "busybox-multi-platform", imgutil.WithXDGRuntimePath(tmpDir), imgutil.FromBaseIndex(baseIndexPath))
					localPath = filepath.Join(tmpDir, "busybox-multi-platform")
				})

				it("returns an image index string representation", func() {
					indexString, err = idx.Inspect()
					h.AssertNil(t, err)

					idxFromString := parseIndex(t, indexString)
					h.AssertEq(t, len(idxFromString.Manifests), 2)
				})
			})
		})
	})
}

func createRemoteImage(t *testing.T, repoName, tag, baseImage string) *imgutilRemote.Image {
	img1RepoName := fmt.Sprintf("%s:%s", repoName, tag)
	img1, err := imgutilRemote.NewImage(img1RepoName, authn.DefaultKeychain, imgutilRemote.FromBaseImage(baseImage))
	h.AssertNil(t, err)
	err = img1.Save()
	h.AssertNil(t, err)
	return img1
}

func setupIndex(t *testing.T, repoName string, ops ...imgutil.IndexOption) imgutil.ImageIndex {
	idx, err := layout.NewIndex(repoName, ops...)
	h.AssertNil(t, err)

	err = idx.SaveDir()
	h.AssertNil(t, err)
	return idx
}

func newRepoName() string {
	return "test-layout-index-" + h.RandString(10)
}

func newTestImageIndexName(name string) string {
	return dockerRegistry.RepoName(name + "-" + h.RandString(10))
}

func parseIndex(t *testing.T, index string) *v1.IndexManifest {
	r := strings.NewReader(index)
	idx, err := v1.ParseIndexManifest(r)
	h.AssertNil(t, err)
	return idx
}
