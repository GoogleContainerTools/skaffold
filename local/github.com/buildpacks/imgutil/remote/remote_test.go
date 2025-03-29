package remote_test

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/remote"
	h "github.com/buildpacks/imgutil/testhelpers"
)

var dockerRegistry, readonlyDockerRegistry, customRegistry *h.DockerRegistry

const (
	readWriteImage    = "image-readable-writable"
	readOnlyImage     = "image-readable"
	writeOnlyImage    = "image-writable"
	inaccessibleImage = "image-inaccessible"
	someSHA           = "sha256:aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f"
)

func newTestImageName(providedPrefix ...string) string {
	prefix := "pack-image-test"
	if len(providedPrefix) > 0 {
		prefix = providedPrefix[0]
	}

	return dockerRegistry.RepoName(prefix + "-" + h.RandString(10))
}

// FIXME: relevant tests in this file should be moved into new_test.go and save_test.go to mirror the implementation
func TestRemote(t *testing.T) {
	dockerConfigDir, err := os.MkdirTemp("", "test.docker.config.dir")
	h.AssertNil(t, err)
	defer os.RemoveAll(dockerConfigDir)

	sharedRegistryHandler := registry.New(registry.Logger(log.New(io.Discard, "", log.Lshortfile)))
	dockerRegistry = h.NewDockerRegistry(h.WithAuth(dockerConfigDir), h.WithSharedHandler(sharedRegistryHandler))
	dockerRegistry.Start(t)
	defer dockerRegistry.Stop(t)

	readonlyDockerRegistry = h.NewDockerRegistry(h.WithSharedHandler(sharedRegistryHandler))
	readonlyDockerRegistry.Start(t)
	defer readonlyDockerRegistry.Stop(t)

	customDockerConfigDir, err := os.MkdirTemp("", "test.docker.config.custom.dir")
	h.AssertNil(t, err)
	defer os.RemoveAll(customDockerConfigDir)
	customRegistry = h.NewDockerRegistry(h.WithAuth(customDockerConfigDir), h.WithSharedHandler(sharedRegistryHandler),
		h.WithImagePrivileges())

	customRegistry.SetReadWrite(readWriteImage)
	customRegistry.SetReadOnly(readOnlyImage)
	customRegistry.SetWriteOnly(writeOnlyImage)
	customRegistry.SetInaccessible(inaccessibleImage)
	customRegistry.Start(t)

	os.Setenv("DOCKER_CONFIG", dockerRegistry.DockerDirectory)
	defer os.Unsetenv("DOCKER_CONFIG")

	spec.Run(t, "Image", testImage, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testImage(t *testing.T, when spec.G, it spec.S) {
	var repoName string

	it.Before(func() {
		repoName = newTestImageName()
	})

	when("#NewImage", func() {
		when("no base image or platform is given", func() {
			it("returns an empty image", func() {
				_, err := remote.NewImage(newTestImageName(), authn.DefaultKeychain)
				h.AssertNil(t, err)
			})

			it("sets sensible defaults for all required fields", func() {
				// os, architecture, and rootfs are required per https://github.com/opencontainers/image-spec/blob/master/config.md
				img, err := remote.NewImage(newTestImageName(), authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, img.Save())

				osName, err := img.OS()
				h.AssertNil(t, err)
				h.AssertEq(t, osName, "linux")

				osVersion, err := img.OSVersion()
				h.AssertNil(t, err)
				h.AssertEq(t, osVersion, "")

				arch, err := img.Architecture()
				h.AssertNil(t, err)
				h.AssertEq(t, arch, runtime.GOARCH)
			})

			it("fails to save to read-only registry", func() {
				readOnlyRepoName := readonlyDockerRegistry.RepoName("pack-image-test" + h.RandString(10))
				img, err := remote.NewImage(readOnlyRepoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertError(t, img.Save(), "Method Not Allowed")
			})
		})

		when("#WithDefaultPlatform", func() {
			it("sets all platform required fields for windows", func() {
				img, err := remote.NewImage(
					newTestImageName(),
					authn.DefaultKeychain,
					remote.WithDefaultPlatform(imgutil.Platform{
						Architecture: "arm",
						OS:           "windows",
						Variant:      "v1",
						OSVersion:    "10.0.17763.316",
					}),
				)
				h.AssertNil(t, err)
				h.AssertNil(t, img.Save())

				arch, err := img.Architecture()
				h.AssertNil(t, err)
				h.AssertEq(t, arch, "arm")

				os, err := img.OS()
				h.AssertNil(t, err)
				h.AssertEq(t, os, "windows")

				variant, err := img.Variant()
				h.AssertNil(t, err)
				h.AssertEq(t, variant, "v1")

				osVersion, err := img.OSVersion()
				h.AssertNil(t, err)
				h.AssertEq(t, osVersion, "10.0.17763.316")

				// base layer is added for windows
				topLayerDiffID, err := img.TopLayer()
				h.AssertNil(t, err)

				h.AssertNotEq(t, topLayerDiffID, "")
			})

			it("sets all platform required fields for linux", func() {
				img, err := remote.NewImage(
					newTestImageName(),
					authn.DefaultKeychain,
					remote.WithDefaultPlatform(imgutil.Platform{
						Architecture: "arm",
						OS:           "linux",
						Variant:      "v6",
						OSVersion:    "21.01",
					}),
				)
				h.AssertNil(t, err)
				h.AssertNil(t, img.Save())

				arch, err := img.Architecture()
				h.AssertNil(t, err)
				h.AssertEq(t, arch, "arm")

				osName, err := img.OS()
				h.AssertNil(t, err)
				h.AssertEq(t, osName, "linux")

				variant, err := img.Variant()
				h.AssertNil(t, err)
				h.AssertEq(t, variant, "v6")

				osVersion, err := img.OSVersion()
				h.AssertNil(t, err)
				h.AssertEq(t, osVersion, "21.01")

				_, err = img.TopLayer()
				h.AssertError(t, err, "has no layers")
			})
		})

		when("#FromBaseImage", func() {
			when("no platform is specified", func() {
				when("base image is an individual image manifest", func() {
					it("sets the initial state from a linux/arm base image", func() {
						baseImageName := "arm64v8/busybox@sha256:50edf1d080946c6a76989d1c3b0e753b62f7d9b5f5e66e88bef23ebbd1e9709c"
						existingLayerSha := "sha256:5a0b973aa300cd2650869fd76d8546b361fcd6dfc77bd37b9d4f082cca9874e4"

						img, err := remote.NewImage(
							repoName,
							authn.DefaultKeychain,
							remote.FromBaseImage(baseImageName),
						)
						h.AssertNil(t, err)

						osName, err := img.OS()
						h.AssertNil(t, err)
						h.AssertEq(t, osName, "linux")

						osVersion, err := img.OSVersion()
						h.AssertNil(t, err)
						h.AssertEq(t, osVersion, "")

						arch, err := img.Architecture()
						h.AssertNil(t, err)
						h.AssertEq(t, arch, "arm64")

						readCloser, err := img.GetLayer(existingLayerSha)
						h.AssertNil(t, err)
						defer readCloser.Close()
					})
					it("tries to pull the image from an insecure registry if WithRegistrySettings insecure has been set", func() {
						_, err := remote.NewImage(
							repoName,
							authn.DefaultKeychain,
							remote.FromBaseImage("host.docker.internal/bar"),
							remote.WithRegistrySetting("host.docker.internal", true))

						h.AssertError(t, err, "http://")
					})

					it("tries to pull the image from an insecure registry if WithRegistrySettings insecure has been set, it works with multiple registries", func() {
						_, err := remote.NewImage(
							repoName,
							authn.DefaultKeychain,
							remote.FromBaseImage("myother-insecure-registry.com/repo/superbase"),
							remote.WithRegistrySetting("myregistry.domain.com", true),
							remote.WithRegistrySetting("myother-insecure-registry.com", true),
						)

						h.AssertError(t, err, "http://myother-insecure-registry.com")
					})

					it("sets the initial state from a windows/amd64 base image", func() {
						baseImageName := "mcr.microsoft.com/windows/nanoserver@sha256:06281772b6a561411d4b338820d94ab1028fdeb076c85350bbc01e80c4bfa2b4"
						existingLayerSha := "sha256:26fd2d9d4c64a4f965bbc77939a454a31b607470f430b5d69fc21ded301fa55e"

						img, err := remote.NewImage(
							repoName,
							authn.DefaultKeychain,
							remote.FromBaseImage(baseImageName),
						)
						h.AssertNil(t, err)

						osName, err := img.OS()
						h.AssertNil(t, err)
						h.AssertEq(t, osName, "windows")

						osVersion, err := img.OSVersion()
						h.AssertNil(t, err)
						h.AssertEq(t, osVersion, "10.0.17763.1040")

						arch, err := img.Architecture()
						h.AssertNil(t, err)
						h.AssertEq(t, arch, "amd64")

						readCloser, err := img.GetLayer(existingLayerSha)
						h.AssertNil(t, err)
						defer readCloser.Close()
					})
				})

				when("base image is a multi-OS/Arch manifest list", func() {
					it("returns a base image matching linux/amd64", func() {
						manifestListName := "golang:1.13.8"
						existingLayerSha := "sha256:427da4a135b0869c1a274ba38e23d45bdbda93134c4ad99c8900cb0cfe9f0c9e"

						img, err := remote.NewImage(
							repoName,
							authn.DefaultKeychain,
							remote.FromBaseImage(manifestListName),
						)
						h.AssertNil(t, err)

						osName, err := img.OS()
						h.AssertNil(t, err)
						h.AssertEq(t, osName, "linux")

						osVersion, err := img.OSVersion()
						h.AssertNil(t, err)
						h.AssertEq(t, osVersion, "")

						arch, err := img.Architecture()
						h.AssertNil(t, err)
						h.AssertEq(t, arch, runtime.GOARCH)

						readCloser, err := img.GetLayer(existingLayerSha)
						h.AssertNil(t, err)
						defer readCloser.Close()
					})
				})

				when("base image does not exist", func() {
					it("returns an empty image", func() {
						img, err := remote.NewImage(
							repoName,
							authn.DefaultKeychain,
							remote.FromBaseImage("some-bad-repo-name"),
						)

						h.AssertNil(t, err)

						_, err = img.TopLayer()
						h.AssertError(t, err, "has no layers")
					})
				})
			})

			when("#WithDefaultPlatform", func() {
				when("base image is an individual image manifest", func() {
					when("platform matches image values", func() {
						it("returns the base image", func() {
							// "golang:1.15.0-nanoserver-1809" image manifest (windows/amd64/10.0.17763.1397)
							windowsImageManifestName := "golang@sha256:6c09e9b9ca3aa2e939c8d0e695d6378f31d42c2473fd57293ba03c254257d9e3"

							img, err := remote.NewImage(
								repoName,
								authn.DefaultKeychain,
								remote.FromBaseImage(windowsImageManifestName),
								remote.WithDefaultPlatform(imgutil.Platform{
									Architecture: "amd64",
									OS:           "windows",
									OSVersion:    "10.0.17763.1397",
								}),
							)
							h.AssertNil(t, err)

							arch, err := img.Architecture()
							h.AssertNil(t, err)
							h.AssertEq(t, arch, "amd64")

							osName, err := img.OS()
							h.AssertNil(t, err)
							h.AssertEq(t, osName, "windows")

							osVersion, err := img.OSVersion()
							h.AssertNil(t, err)
							h.AssertEq(t, osVersion, "10.0.17763.1397")
						})
					})

					when("platform conflicts with image values", func() {
						it("returns the base image, regardless of platform values", func() {
							// "golang:1.15.0-nanoserver-1809" image manifest (windows/amd64/10.0.17763.1397)
							windowsImageManifestName := "golang@sha256:6c09e9b9ca3aa2e939c8d0e695d6378f31d42c2473fd57293ba03c254257d9e3"

							img, err := remote.NewImage(
								repoName,
								authn.DefaultKeychain,
								remote.FromBaseImage(windowsImageManifestName),
								remote.WithDefaultPlatform(imgutil.Platform{
									OS:           "linux",
									Architecture: "arm",
								}),
							)
							h.AssertNil(t, err)

							arch, err := img.Architecture()
							h.AssertNil(t, err)
							h.AssertEq(t, arch, "amd64")

							osName, err := img.OS()
							h.AssertNil(t, err)
							h.AssertEq(t, osName, "windows")

							osVersion, err := img.OSVersion()
							h.AssertNil(t, err)
							h.AssertEq(t, osVersion, "10.0.17763.1397")
						})
					})
				})

				when("base image is a multi-OS/Arch manifest list", func() {
					when("image with matching platform fields exists", func() {
						it("returns the image whose index matches the platform", func() {
							manifestListName := "golang:1.13.8"

							img, err := remote.NewImage(
								repoName,
								authn.DefaultKeychain,
								remote.FromBaseImage(manifestListName),
								remote.WithDefaultPlatform(imgutil.Platform{
									OS:           "linux",
									Architecture: "amd64",
								}),
							)
							h.AssertNil(t, err)

							arch, err := img.Architecture()
							h.AssertNil(t, err)
							h.AssertEq(t, arch, "amd64")

							osName, err := img.OS()
							h.AssertNil(t, err)
							h.AssertEq(t, osName, "linux")
						})
					})

					when("no image with matching platform exists", func() {
						it("returns an empty image with platform fields set", func() {
							manifestListName := "golang:1.13.8"

							img, err := remote.NewImage(
								repoName,
								authn.DefaultKeychain,
								remote.FromBaseImage(manifestListName),
								remote.WithDefaultPlatform(imgutil.Platform{
									OS:           "windows",
									Architecture: "arm",
								}),
							)

							h.AssertNil(t, err)

							arch, err := img.Architecture()
							h.AssertNil(t, err)
							h.AssertEq(t, arch, "arm")

							osName, err := img.OS()
							h.AssertNil(t, err)
							h.AssertEq(t, osName, "windows")

							osVersion, err := img.OSVersion()
							h.AssertNil(t, err)
							h.AssertEq(t, osVersion, "")

							// base layer is added for Windows
							topLayerDiffID, err := img.TopLayer()
							h.AssertNil(t, err)

							h.AssertNotEq(t, topLayerDiffID, "")
						})
					})
				})

				when("base image does not exist", func() {
					it("returns an empty linux image based on platform fields", func() {
						img, err := remote.NewImage(
							repoName,
							authn.DefaultKeychain,
							remote.FromBaseImage("some-bad-repo-name"),
							remote.WithDefaultPlatform(imgutil.Platform{
								Architecture: "arm",
								OS:           "linux",
							}),
						)

						h.AssertNil(t, err)

						arch, err := img.Architecture()
						h.AssertNil(t, err)
						h.AssertEq(t, arch, "arm")

						osName, err := img.OS()
						h.AssertNil(t, err)
						h.AssertEq(t, osName, "linux")

						osVersion, err := img.OSVersion()
						h.AssertNil(t, err)
						h.AssertEq(t, osVersion, "")

						_, err = img.TopLayer()
						h.AssertError(t, err, "has no layers")
					})

					it("returns an empty windows image based on platform fields", func() {
						img, err := remote.NewImage(
							repoName,
							authn.DefaultKeychain,
							remote.FromBaseImage("some-bad-repo-name"),
							remote.WithDefaultPlatform(imgutil.Platform{
								Architecture: "arm",
								OS:           "windows",
								OSVersion:    "10.0.99999.9999",
							}),
						)

						h.AssertNil(t, err)

						arch, err := img.Architecture()
						h.AssertNil(t, err)
						h.AssertEq(t, arch, "arm")

						osName, err := img.OS()
						h.AssertNil(t, err)
						h.AssertEq(t, osName, "windows")

						osVersion, err := img.OSVersion()
						h.AssertNil(t, err)
						h.AssertEq(t, osVersion, "10.0.99999.9999")

						// base layer is added for windows
						topLayerDiffID, err := img.TopLayer()
						h.AssertNil(t, err)

						h.AssertNotEq(t, topLayerDiffID, "")
					})
				})
			})
		})

		when("#WithPreviousImage", func() {
			when("previous image is exists", func() {
				it("provides reusable layers", func() {
					baseImageName := "arm64v8/busybox@sha256:50edf1d080946c6a76989d1c3b0e753b62f7d9b5f5e66e88bef23ebbd1e9709c"
					existingLayerSha := "sha256:5a0b973aa300cd2650869fd76d8546b361fcd6dfc77bd37b9d4f082cca9874e4"

					img, err := remote.NewImage(
						repoName,
						authn.DefaultKeychain,
						remote.WithPreviousImage(baseImageName),
					)

					h.AssertNil(t, err)

					h.AssertNil(t, img.ReuseLayer(existingLayerSha))
				})

				when("#WithDefaultPlatform", func() {
					it("provides reusable layers from image in a manifest list with specific platform", func() {
						manifestListName := "golang:1.13.8"
						existingLayerSha := "sha256:cba908afa240240fceb312eb682bd7660fd5a404ddfd22843852dfdef141314b"

						img, err := remote.NewImage(
							repoName,
							authn.DefaultKeychain,
							remote.WithPreviousImage(manifestListName),
							remote.WithDefaultPlatform(imgutil.Platform{
								OS:           "windows",
								Architecture: "amd64",
							}),
						)
						h.AssertNil(t, err)

						h.AssertNil(t, img.ReuseLayer(existingLayerSha))
					})
				})
			})

			when("previous image does not exist", func() {
				it("does not error", func() {
					_, err := remote.NewImage(
						repoName,
						authn.DefaultKeychain,
						remote.WithPreviousImage("some-bad-repo-name"),
					)

					h.AssertNil(t, err)
				})
			})
		})

		when("#AddEmptyLayerOnSave", func() {
			it("an empty layer was added on save", func() {
				image, err := remote.NewImage(
					repoName,
					authn.DefaultKeychain,
					remote.AddEmptyLayerOnSave(),
				)

				h.AssertNil(t, err)
				h.AssertNil(t, image.Save())
				h.AssertEq(t, len(h.FetchManifestLayers(t, repoName)), 1)
			})
		})

		when("#WithMediaTypes", func() {
			it("sets the requested media types", func() {
				img, err := remote.NewImage(
					newTestImageName(),
					authn.DefaultKeychain,
					remote.WithMediaTypes(imgutil.OCITypes),
				)
				h.AssertNil(t, err)
				h.AssertOCIMediaTypes(t, img.UnderlyingImage()) // before saving
				// add a random layer
				newLayerPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", "linux")
				h.AssertNil(t, err)
				defer os.Remove(newLayerPath)
				err = img.AddLayer(newLayerPath)
				h.AssertNil(t, err)
				h.AssertOCIMediaTypes(t, img.UnderlyingImage()) // after adding a layer
				h.AssertNil(t, img.Save())
				h.AssertOCIMediaTypes(t, img.UnderlyingImage()) // after saving
			})

			when("using a base image", func() {
				it("sets the requested media types", func() {
					baseImageName := newTestImageName()
					baseImage, err := remote.NewImage(
						baseImageName,
						authn.DefaultKeychain,
						remote.WithMediaTypes(imgutil.DockerTypes),
					)
					h.AssertNil(t, err)
					h.AssertNil(t, baseImage.Save())

					img, err := remote.NewImage(
						newTestImageName(),
						authn.DefaultKeychain,
						remote.WithMediaTypes(imgutil.OCITypes),
						remote.FromBaseImage(baseImageName),
					)
					h.AssertNil(t, err)
					h.AssertOCIMediaTypes(t, img.UnderlyingImage()) // before saving
					// add a random layer
					newLayerPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", "linux")
					h.AssertNil(t, err)
					defer os.Remove(newLayerPath)
					err = img.AddLayer(newLayerPath)
					h.AssertNil(t, err)
					h.AssertOCIMediaTypes(t, img.UnderlyingImage()) // after adding a layer
					h.AssertNil(t, img.Save())
					h.AssertOCIMediaTypes(t, img.UnderlyingImage()) // after saving
				})
			})
		})

		when("#WithConfig", func() {
			var config = &v1.Config{Entrypoint: []string{"some-entrypoint"}}

			it("sets the image config", func() {
				remoteImage, err := remote.NewImage(newTestImageName(), authn.DefaultKeychain, remote.WithConfig(config))
				h.AssertNil(t, err)

				entrypoint, err := remoteImage.Entrypoint()
				h.AssertNil(t, err)
				h.AssertEq(t, entrypoint, []string{"some-entrypoint"})
			})

			when("#FromBaseImage", func() {
				var baseImageName = newTestImageName()

				it("overrides the base image config", func() {
					baseImage, err := remote.NewImage(baseImageName, authn.DefaultKeychain)
					h.AssertNil(t, err)
					h.AssertNil(t, baseImage.Save())

					remoteImage, err := remote.NewImage(
						newTestImageName(),
						authn.DefaultKeychain,
						remote.WithConfig(config),
						remote.FromBaseImage(baseImageName),
					)
					h.AssertNil(t, err)

					entrypoint, err := remoteImage.Entrypoint()
					h.AssertNil(t, err)
					h.AssertEq(t, entrypoint, []string{"some-entrypoint"})
				})
			})
		})
	})

	when("#WorkingDir", func() {
		when("image exists", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				baseImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertNil(t, baseImage.SetWorkingDir("/testWorkingDir"))
				h.AssertNil(t, baseImage.Save())
			})

			it("returns the WorkingDir value", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.WorkingDir()
				h.AssertNil(t, err)

				h.AssertEq(t, val, "/testWorkingDir")
			})

			it("returns empty string for a missing WorkingDir", func() {
				baseImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, baseImage.Save())

				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.WorkingDir()
				h.AssertNil(t, err)
				var expected string
				h.AssertEq(t, val, expected)
			})
		})

		when("image NOT exists", func() {
			it("returns empty string", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				val, err := img.WorkingDir()
				h.AssertNil(t, err)
				var expected string
				h.AssertEq(t, val, expected)
			})
		})
	})

	when("#Entrypoint", func() {
		when("image exists with entrypoint", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				baseImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertNil(t, baseImage.SetEntrypoint("entrypoint1", "entrypoint2"))
				h.AssertNil(t, baseImage.Save())
			})

			it("returns the entrypoint value", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.Entrypoint()
				h.AssertNil(t, err)
				h.AssertEq(t, val, []string{"entrypoint1", "entrypoint2"})
			})

			it("returns nil slice for a missing entrypoint", func() {
				baseImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, baseImage.Save())

				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.Entrypoint()
				h.AssertNil(t, err)
				var expected []string
				h.AssertEq(t, val, expected)
			})
		})

		when("image NOT exists", func() {
			it("returns nil slice", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				val, err := img.Entrypoint()
				h.AssertNil(t, err)
				var expected []string
				h.AssertEq(t, val, expected)
			})
		})
	})

	when("#Labels", func() {
		when("image exists with labels", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				baseImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertNil(t, baseImage.SetLabel("mykey", "myvalue"))
				h.AssertNil(t, baseImage.SetLabel("other", "data"))
				h.AssertNil(t, baseImage.Save())
			})

			it("returns all the labels", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				labels, err := img.Labels()
				h.AssertNil(t, err)
				h.AssertEq(t, labels["mykey"], "myvalue")
				h.AssertEq(t, labels["other"], "data")
			})
		})

		when("image exists with no labels", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				baseImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, baseImage.Save())
			})

			it("returns nil", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				labels, err := img.Labels()
				h.AssertNil(t, err)
				h.AssertEq(t, 0, len(labels))
			})
		})

		when("image NOT exists", func() {
			it("returns nil", func() {
				img, err := remote.NewImage(newTestImageName(), authn.DefaultKeychain)
				h.AssertNil(t, err)

				labels, err := img.Labels()
				h.AssertNil(t, err)
				h.AssertEq(t, 0, len(labels))
			})
		})
	})

	when("#Label", func() {
		when("image exists", func() {
			it.Before(func() {
				baseImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertNil(t, baseImage.SetLabel("mykey", "myvalue"))
				h.AssertNil(t, baseImage.SetLabel("other", "data"))
				h.AssertNil(t, baseImage.Save())
			})

			it("returns the label value", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				label, err := img.Label("mykey")
				h.AssertNil(t, err)
				h.AssertEq(t, label, "myvalue")
			})

			it("returns an empty string for a missing label", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				label, err := img.Label("missing-label")
				h.AssertNil(t, err)
				h.AssertEq(t, label, "")
			})
		})

		when("image is empty", func() {
			it("returns an empty label", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				label, err := img.Label("some-label")
				h.AssertNil(t, err)
				h.AssertEq(t, label, "")
			})
		})
	})

	when("#Env", func() {
		when("image exists", func() {
			var baseImageName = newTestImageName()

			it.Before(func() {
				baseImage, err := remote.NewImage(baseImageName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, baseImage.SetEnv("MY_VAR", "my_val"))
				h.AssertNil(t, baseImage.Save())
			})

			it("returns the label value", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(baseImageName))
				h.AssertNil(t, err)

				val, err := img.Env("MY_VAR")
				h.AssertNil(t, err)
				h.AssertEq(t, val, "my_val")
			})

			it("returns an empty string for a missing label", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(baseImageName))
				h.AssertNil(t, err)

				val, err := img.Env("MISSING_VAR")
				h.AssertNil(t, err)
				h.AssertEq(t, val, "")
			})
		})

		when("image is empty", func() {
			it("returns an empty string", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				val, err := img.Env("SOME_VAR")
				h.AssertNil(t, err)
				h.AssertEq(t, val, "")
			})
		})
	})

	when("#Name", func() {
		it("always returns the original name", func() {
			img, err := remote.NewImage(repoName, authn.DefaultKeychain)
			h.AssertNil(t, err)
			h.AssertEq(t, img.Name(), repoName)
		})
	})

	when("#CreatedAt", func() {
		const reference = "busybox@sha256:f79f7a10302c402c052973e3fa42be0344ae6453245669783a9e16da3d56d5b4"
		it("returns the containers created at time", func() {
			img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(reference))
			h.AssertNil(t, err)

			expectedTime := time.Date(2019, 4, 2, 23, 32, 10, 727183061, time.UTC)

			createdTime, err := img.CreatedAt()

			h.AssertNil(t, err)
			h.AssertEq(t, createdTime, expectedTime)
		})
	})

	when("#Identifier", func() {
		it("returns a digest reference", func() {
			img, err := remote.NewImage(
				repoName+":some-tag",
				authn.DefaultKeychain,
				remote.FromBaseImage("busybox@sha256:915f390a8912e16d4beb8689720a17348f3f6d1a7b659697df850ab625ea29d5"),
			)
			h.AssertNil(t, err)

			identifier, err := img.Identifier()
			h.AssertNil(t, err)
			h.AssertEq(t, identifier.String(), repoName+"@sha256:915f390a8912e16d4beb8689720a17348f3f6d1a7b659697df850ab625ea29d5")
		})

		it("accurately parses the reference for an image with a sha", func() {
			img, err := remote.NewImage(
				repoName+"@sha256:915f390a8912e16d4beb8689720a17348f3f6d1a7b659697df850ab625ea29d5",
				authn.DefaultKeychain,
				remote.FromBaseImage("busybox@sha256:915f390a8912e16d4beb8689720a17348f3f6d1a7b659697df850ab625ea29d5"),
			)
			h.AssertNil(t, err)

			identifier, err := img.Identifier()
			h.AssertNil(t, err)
			h.AssertEq(t, identifier.String(), repoName+"@sha256:915f390a8912e16d4beb8689720a17348f3f6d1a7b659697df850ab625ea29d5")
		})

		when("the image has been modified and saved", func() {
			it("returns the new digest reference", func() {
				img, err := remote.NewImage(
					repoName+":some-tag",
					authn.DefaultKeychain,
					remote.FromBaseImage("busybox@sha256:915f390a8912e16d4beb8689720a17348f3f6d1a7b659697df850ab625ea29d5"),
				)
				h.AssertNil(t, err)

				h.AssertNil(t, img.SetLabel("new", "label"))

				h.AssertNil(t, img.Save())

				id, err := img.Identifier()
				h.AssertNil(t, err)

				testImg, err := remote.NewImage(
					"test",
					authn.DefaultKeychain,
					remote.FromBaseImage(id.String()),
				)
				h.AssertNil(t, err)

				remoteLabel, err := testImg.Label("new")
				h.AssertNil(t, err)

				h.AssertEq(t, remoteLabel, "label")
			})
		})
	})

	when("#SetLabel", func() {
		when("image exists", func() {
			it("sets label on img object", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertNil(t, img.SetLabel("mykey", "new-val"))
				label, err := img.Label("mykey")
				h.AssertNil(t, err)
				h.AssertEq(t, label, "new-val")
			})

			it("saves label", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertNil(t, img.SetLabel("mykey", "new-val"))

				h.AssertNil(t, img.Save())

				testImg, err := remote.NewImage(
					"test",
					authn.DefaultKeychain,
					remote.FromBaseImage(repoName),
				)
				h.AssertNil(t, err)

				remoteLabel, err := testImg.Label("mykey")
				h.AssertNil(t, err)

				h.AssertEq(t, remoteLabel, "new-val")
			})
		})
	})

	when("#RemoveLabel", func() {
		when("image exists", func() {
			var baseImageName = newTestImageName()

			it.Before(func() {
				baseImage, err := remote.NewImage(baseImageName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, baseImage.SetLabel("custom.label", "new-val"))
				h.AssertNil(t, baseImage.Save())
			})

			it("removes label on img object", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(baseImageName))
				h.AssertNil(t, err)

				h.AssertNil(t, img.RemoveLabel("custom.label"))

				labels, err := img.Labels()
				h.AssertNil(t, err)
				_, exists := labels["my.custom.label"]
				h.AssertEq(t, exists, false)
			})

			it("saves removal of label", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(baseImageName))
				h.AssertNil(t, err)

				h.AssertNil(t, img.RemoveLabel("custom.label"))
				h.AssertNil(t, img.Save())

				testImg, err := remote.NewImage(
					"test",
					authn.DefaultKeychain,
					remote.FromBaseImage(repoName),
				)
				h.AssertNil(t, err)

				remoteLabel, err := testImg.Label("custom.label")
				h.AssertNil(t, err)
				h.AssertEq(t, remoteLabel, "")
			})
		})
	})

	when("#SetEnv", func() {
		it("sets the environment", func() {
			img, err := remote.NewImage(repoName, authn.DefaultKeychain)
			h.AssertNil(t, err)

			err = img.SetEnv("ENV_KEY", "ENV_VAL")
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			configFile := h.FetchManifestImageConfigFile(t, repoName)
			h.AssertContains(t, configFile.Config.Env, "ENV_KEY=ENV_VAL")
		})

		when("the key already exists", func() {
			it("overrides the existing key", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				err = img.SetEnv("ENV_KEY", "SOME_VAL")
				h.AssertNil(t, err)

				err = img.SetEnv("ENV_KEY", "SOME_OTHER_VAL")
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				configFile := h.FetchManifestImageConfigFile(t, repoName)
				h.AssertContains(t, configFile.Config.Env, "ENV_KEY=SOME_OTHER_VAL")
				h.AssertDoesNotContain(t, configFile.Config.Env, "ENV_KEY=SOME_VAL")
			})
		})

		when("windows", func() {
			it("ignores case", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				imgOS, err := img.OS()
				h.AssertNil(t, err)

				if imgOS != "windows" {
					t.Skip("windows test")
				}

				err = img.SetEnv("ENV_KEY", "SOME_VAL")
				h.AssertNil(t, err)

				err = img.SetEnv("env_key", "SOME_OTHER_VAL")
				h.AssertNil(t, err)

				err = img.SetEnv("env_key2", "SOME_VAL")
				h.AssertNil(t, err)

				err = img.SetEnv("ENV_KEY2", "SOME_OTHER_VAL")
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				configFile := h.FetchManifestImageConfigFile(t, repoName)

				h.AssertContains(t, configFile.Config.Env, "env_key=SOME_OTHER_VAL")
				h.AssertDoesNotContain(t, configFile.Config.Env, "ENV_KEY=SOME_VAL")
				h.AssertDoesNotContain(t, configFile.Config.Env, "ENV_KEY=SOME_OTHER_VAL")

				h.AssertContains(t, configFile.Config.Env, "ENV_KEY2=SOME_OTHER_VAL")
				h.AssertDoesNotContain(t, configFile.Config.Env, "env_key2=SOME_OTHER_VAL")
				h.AssertDoesNotContain(t, configFile.Config.Env, "env_key2=SOME_VAL")
			})
		})
	})

	when("#SetWorkingDir", func() {
		it("sets the environment", func() {
			img, err := remote.NewImage(repoName, authn.DefaultKeychain)
			h.AssertNil(t, err)

			err = img.SetWorkingDir("/some/work/dir")
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			configFile := h.FetchManifestImageConfigFile(t, repoName)
			h.AssertEq(t, configFile.Config.WorkingDir, "/some/work/dir")
		})
	})

	when("#SetEntrypoint", func() {
		it("sets the entrypoint", func() {
			img, err := remote.NewImage(repoName, authn.DefaultKeychain)
			h.AssertNil(t, err)

			err = img.SetEntrypoint("some", "entrypoint")
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			configFile := h.FetchManifestImageConfigFile(t, repoName)
			h.AssertEq(t, configFile.Config.Entrypoint, []string{"some", "entrypoint"})
		})
	})

	when("#SetCmd", func() {
		it("sets the cmd", func() {
			img, err := remote.NewImage(repoName, authn.DefaultKeychain)
			h.AssertNil(t, err)

			err = img.SetCmd("some", "cmd")
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			configFile := h.FetchManifestImageConfigFile(t, repoName)
			h.AssertEq(t, configFile.Config.Cmd, []string{"some", "cmd"})
		})
	})

	when("#SetOS #SetOSVersion #SetArchitecture", func() {
		it("sets the os/arch", func() {
			var (
				os        = "foobaros"
				arch      = "arm64"
				osVersion = "1.2.3.4"
			)
			img, err := remote.NewImage(repoName, authn.DefaultKeychain)
			h.AssertNil(t, err)

			err = img.SetOS(os)
			h.AssertNil(t, err)
			err = img.SetOSVersion(osVersion)
			h.AssertNil(t, err)
			err = img.SetArchitecture(arch)
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			configFile := h.FetchManifestImageConfigFile(t, repoName)
			h.AssertEq(t, configFile.OS, os)
			h.AssertEq(t, configFile.OSVersion, osVersion)
			h.AssertEq(t, configFile.Architecture, arch)
		})
	})

	when("#Rebase", func() {
		when("image exists", func() {
			var oldBase, newBase, oldTopLayerDiffID string
			var oldBaseLayers, newBaseLayers, repoTopLayers []string
			it.Before(func() {
				// new base
				newBase = newTestImageName("pack-newbase-test")
				newBaseLayer1Path, err := h.CreateSingleFileLayerTar("/new-base.txt", "new-base", "linux")
				h.AssertNil(t, err)
				defer os.Remove(newBaseLayer1Path)

				newBaseLayer2Path, err := h.CreateSingleFileLayerTar("/otherfile.txt", "text-new-base", "linux")
				h.AssertNil(t, err)
				defer os.Remove(newBaseLayer2Path)

				newBaseImage, err := remote.NewImage(newBase, authn.DefaultKeychain)
				h.AssertNil(t, err)

				err = newBaseImage.AddLayer(newBaseLayer1Path)
				h.AssertNil(t, err)

				err = newBaseImage.AddLayer(newBaseLayer2Path)
				h.AssertNil(t, err)

				h.AssertNil(t, newBaseImage.Save())

				newBaseLayers = h.FetchManifestLayers(t, newBase)

				// old base image
				oldBase = newTestImageName("pack-oldbase-test")
				oldBaseLayer1Path, err := h.CreateSingleFileLayerTar("/old-base.txt", "old-base", "linux")
				h.AssertNil(t, err)
				defer os.Remove(oldBaseLayer1Path)

				oldBaseLayer2Path, err := h.CreateSingleFileLayerTar("/otherfile.txt", "text-old-base", "linux")
				h.AssertNil(t, err)
				defer os.Remove(oldBaseLayer2Path)

				oldBaseImage, err := remote.NewImage(oldBase, authn.DefaultKeychain)
				h.AssertNil(t, err)

				err = oldBaseImage.AddLayer(oldBaseLayer1Path)
				h.AssertNil(t, err)

				err = oldBaseImage.AddLayer(oldBaseLayer2Path)
				h.AssertNil(t, err)

				oldTopLayerDiffID = h.FileDiffID(t, oldBaseLayer2Path)

				h.AssertNil(t, oldBaseImage.Save())

				oldBaseLayers = h.FetchManifestLayers(t, oldBase)

				// original image
				origLayer1Path, err := h.CreateSingleFileLayerTar("/bmyimage.txt", "text-from-image-1", "linux")
				h.AssertNil(t, err)
				defer os.Remove(origLayer1Path)

				origLayer2Path, err := h.CreateSingleFileLayerTar("/myimage2.txt", "text-from-image-2", "linux")
				h.AssertNil(t, err)
				defer os.Remove(origLayer2Path)

				origImage, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(oldBase))
				h.AssertNil(t, err)

				err = origImage.AddLayer(origLayer1Path)
				h.AssertNil(t, err)

				err = origImage.AddLayer(origLayer2Path)
				h.AssertNil(t, err)

				h.AssertNil(t, origImage.Save())

				repoLayers := h.FetchManifestLayers(t, repoName)
				repoTopLayers = repoLayers[len(oldBaseLayers):]
			})

			it("switches the base", func() {
				// Before
				h.AssertEq(t,
					h.FetchManifestLayers(t, repoName),
					append(oldBaseLayers, repoTopLayers...),
				)

				// Run rebase
				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				newBaseImg, err := remote.NewImage(newBase, authn.DefaultKeychain, remote.FromBaseImage(newBase))
				h.AssertNil(t, err)

				h.AssertNil(t, newBaseImg.MutateConfigFile(func(c *v1.ConfigFile) {
					c.History = []v1.History{
						{CreatedBy: "/new-base.txt"},
						{CreatedBy: "FOOBAR", EmptyLayer: true}, // add empty layer history
						{CreatedBy: "/otherfile.txt"},
					}
				})) // don't save the image, as that will strip the empty layer history

				err = img.Rebase(oldTopLayerDiffID, newBaseImg)
				h.AssertNil(t, err)
				h.AssertNil(t, img.Save())

				// After
				h.AssertEq(t,
					h.FetchManifestLayers(t, repoName),
					append(newBaseLayers, repoTopLayers...),
				)

				newBaseConfig := h.FetchManifestImageConfigFile(t, newBase)
				rebasedImgConfig := h.FetchManifestImageConfigFile(t, repoName)
				h.AssertEq(t, rebasedImgConfig.OS, newBaseConfig.OS)
				h.AssertEq(t, rebasedImgConfig.OSVersion, newBaseConfig.OSVersion)
				h.AssertEq(t, rebasedImgConfig.Architecture, newBaseConfig.Architecture)
				h.AssertEq(t, len(rebasedImgConfig.History), len(rebasedImgConfig.RootFS.DiffIDs))
			})
		})
	})

	when("#TopLayer", func() {
		when("image exists", func() {
			it("returns the digest for the top layer (useful for rebasing)", func() {
				baseLayerPath, err := h.CreateSingleFileLayerTar("/old-base.txt", "old-base", "linux")
				h.AssertNil(t, err)
				defer os.Remove(baseLayerPath)

				topLayerPath, err := h.CreateSingleFileLayerTar("/top-layer.txt", "top-layer", "linux")
				h.AssertNil(t, err)
				defer os.Remove(topLayerPath)

				expectedTopLayerDiffID := h.FileDiffID(t, topLayerPath)

				existingImage, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				err = existingImage.AddLayer(baseLayerPath)
				h.AssertNil(t, err)

				err = existingImage.AddLayer(topLayerPath)
				h.AssertNil(t, err)

				h.AssertNil(t, existingImage.Save())

				img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.FromBaseImage(repoName))
				h.AssertNil(t, err)

				actualTopLayerDiffID, err := img.TopLayer()
				h.AssertNil(t, err)

				h.AssertEq(t, actualTopLayerDiffID, expectedTopLayerDiffID)
			})
		})

		when("the image has no layers", func() {
			it("returns an error", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				_, err = img.TopLayer()
				h.AssertError(t, err, "has no layers")
			})
		})
	})

	when("#AddLayer", func() {
		it("appends a layer", func() {
			existingImage, err := remote.NewImage(
				repoName,
				authn.DefaultKeychain,
			)
			h.AssertNil(t, err)

			oldLayerPath, err := h.CreateSingleFileLayerTar("/old-layer.txt", "old-layer", "linux")
			h.AssertNil(t, err)
			defer os.Remove(oldLayerPath)

			oldLayerDiffID := h.FileDiffID(t, oldLayerPath)

			h.AssertNil(t, existingImage.AddLayer(oldLayerPath))

			h.AssertNil(t, existingImage.Save())
			img, err := remote.NewImage(
				repoName,
				authn.DefaultKeychain,
				remote.FromBaseImage(repoName),
			)
			h.AssertNil(t, err)

			newLayerPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", "linux")
			h.AssertNil(t, err)
			defer os.Remove(newLayerPath)

			newLayerDiffID := h.FileDiffID(t, newLayerPath)

			err = img.AddLayer(newLayerPath)
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			manifestLayerDiffIDs := h.FetchManifestLayers(t, repoName)

			h.AssertEq(t, oldLayerDiffID, h.StringElementAt(manifestLayerDiffIDs, -2))
			h.AssertEq(t, newLayerDiffID, h.StringElementAt(manifestLayerDiffIDs, -1))
		})
	})

	when("#AddLayerWithDiffID", func() {
		it("appends a layer", func() {
			existingImage, err := remote.NewImage(
				repoName,
				authn.DefaultKeychain,
			)
			h.AssertNil(t, err)

			oldLayerPath, err := h.CreateSingleFileLayerTar("/old-layer.txt", "old-layer", "linux")
			h.AssertNil(t, err)
			defer os.Remove(oldLayerPath)
			oldLayerDiffID := h.FileDiffID(t, oldLayerPath)

			h.AssertNil(t, existingImage.AddLayer(oldLayerPath))

			h.AssertNil(t, existingImage.Save())

			img, err := remote.NewImage(
				repoName,
				authn.DefaultKeychain,
				remote.FromBaseImage(repoName),
			)
			h.AssertNil(t, err)

			newLayerPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", "linux")
			h.AssertNil(t, err)
			defer os.Remove(newLayerPath)

			newLayerDiffID := h.FileDiffID(t, newLayerPath)

			err = img.AddLayerWithDiffID(newLayerPath, newLayerDiffID)
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			manifestLayerDiffIDs := h.FetchManifestLayers(t, repoName)

			h.AssertEq(t, oldLayerDiffID, h.StringElementAt(manifestLayerDiffIDs, -2))
			h.AssertEq(t, newLayerDiffID, h.StringElementAt(manifestLayerDiffIDs, -1))
		})
	})

	when("#AddLayerWithDiffIDAndHistory", func() {
		it("appends a layer with history", func() {
			existingImage, err := remote.NewImage(
				repoName,
				authn.DefaultKeychain,
			)
			h.AssertNil(t, err)

			oldLayerPath, err := h.CreateSingleFileLayerTar("/old-layer.txt", "old-layer", "linux")
			h.AssertNil(t, err)
			defer os.Remove(oldLayerPath)
			oldLayerDiffID := h.FileDiffID(t, oldLayerPath)

			h.AssertNil(t, existingImage.AddLayer(oldLayerPath))

			h.AssertNil(t, existingImage.Save())

			img, err := remote.NewImage(
				repoName,
				authn.DefaultKeychain,
				remote.FromBaseImage(repoName),
				remote.WithHistory(),
			)
			h.AssertNil(t, err)

			newLayerPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", "linux")
			h.AssertNil(t, err)
			defer os.Remove(newLayerPath)

			newLayerDiffID := h.FileDiffID(t, newLayerPath)

			history, err := img.History()
			h.AssertNil(t, err)
			oldNHistory := len(history)
			addedHistory := v1.History{
				Author:     "some-author",
				Created:    v1.Time{Time: imgutil.NormalizedDateTime},
				CreatedBy:  "some-history",
				Comment:    "some-comment",
				EmptyLayer: false,
			}
			err = img.AddLayerWithDiffIDAndHistory(newLayerPath, newLayerDiffID, addedHistory)
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			// check history
			history, err = img.History()
			h.AssertNil(t, err)
			h.AssertEq(t, len(history), oldNHistory+1)
			h.AssertEq(t, history[len(history)-1], addedHistory)

			manifestLayerDiffIDs := h.FetchManifestLayers(t, repoName)

			h.AssertEq(t, oldLayerDiffID, h.StringElementAt(manifestLayerDiffIDs, -2))
			h.AssertEq(t, newLayerDiffID, h.StringElementAt(manifestLayerDiffIDs, -1))
		})
	})

	when("#ReuseLayer", func() {
		when("previous image", func() {
			var (
				prevImage     *remote.Image
				prevImageName string
				prevLayer1SHA string
				prevLayer2SHA string
			)

			it.Before(func() {
				prevImageName = newTestImageName()
				var err error
				prevImage, err = remote.NewImage(
					prevImageName,
					authn.DefaultKeychain,
					remote.WithHistory(),
				)
				h.AssertNil(t, err)

				layer1Path, err := h.CreateSingleFileLayerTar("/layer-1.txt", "old-layer-1", "linux")
				h.AssertNil(t, err)
				defer os.Remove(layer1Path)

				prevLayer1SHA = h.FileDiffID(t, layer1Path)

				layer2Path, err := h.CreateSingleFileLayerTar("/layer-2.txt", "old-layer-2", "linux")
				h.AssertNil(t, err)
				defer os.Remove(layer2Path)

				prevLayer2SHA = h.FileDiffID(t, layer2Path)

				h.AssertNil(t, prevImage.AddLayer(layer1Path))
				h.AssertNil(t, prevImage.AddLayer(layer2Path))

				h.AssertNil(t, prevImage.Save())
			})

			it("reuses a layer", func() {
				img, err := remote.NewImage(
					repoName,
					authn.DefaultKeychain,
					remote.WithPreviousImage(prevImageName),
				)
				h.AssertNil(t, err)

				newBaseLayerPath, err := h.CreateSingleFileLayerTar("/new-base.txt", "base-content", "linux")
				h.AssertNil(t, err)
				defer os.Remove(newBaseLayerPath)

				h.AssertNil(t, img.AddLayer(newBaseLayerPath))

				err = img.ReuseLayer(prevLayer2SHA)
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				manifestLayers := h.FetchManifestLayers(t, repoName)

				newLayer1SHA := h.StringElementAt(manifestLayers, -2)
				reusedLayer2SHA := h.StringElementAt(manifestLayers, -1)

				h.AssertNotEq(t, prevLayer1SHA, newLayer1SHA)
				h.AssertEq(t, prevLayer2SHA, reusedLayer2SHA)
			})

			it("returns error on nonexistent layer", func() {
				img, err := remote.NewImage(
					repoName,
					authn.DefaultKeychain,
					remote.WithPreviousImage(prevImageName),
				)
				h.AssertNil(t, err)

				img.Rename(repoName)

				err = img.ReuseLayer(someSHA)

				h.AssertError(t, err, fmt.Sprintf("failed to find diffID %s in config file", someSHA))
			})

			when("there is history", func() {
				var prevHistory []v1.History

				it.Before(func() {
					layers, err := prevImage.UnderlyingImage().Layers()
					h.AssertNil(t, err)
					prevHistory = make([]v1.History, len(layers))
					for idx := range prevHistory {
						prevHistory[idx].CreatedBy = fmt.Sprintf("some-history-%d", idx)
					}
					h.AssertNil(t, prevImage.SetHistory(prevHistory))
					h.AssertNil(t, prevImage.Save())
				})

				it("reuses a layer with history", func() {
					img, err := remote.NewImage(
						repoName,
						authn.DefaultKeychain,
						remote.WithPreviousImage(prevImageName),
						remote.WithHistory(),
					)
					h.AssertNil(t, err)

					newBaseLayerPath, err := h.CreateSingleFileLayerTar("/new-base.txt", "base-content", "linux")
					h.AssertNil(t, err)
					defer os.Remove(newBaseLayerPath)

					h.AssertNil(t, img.AddLayer(newBaseLayerPath))

					err = img.ReuseLayer(prevLayer2SHA)
					h.AssertNil(t, err)

					h.AssertNil(t, img.Save())

					manifestLayers := h.FetchManifestLayers(t, repoName)

					newLayer1SHA := h.StringElementAt(manifestLayers, -2)
					reusedLayer2SHA := h.StringElementAt(manifestLayers, -1)

					h.AssertNotEq(t, prevLayer1SHA, newLayer1SHA)
					h.AssertEq(t, prevLayer2SHA, reusedLayer2SHA)

					history, err := img.History()
					h.AssertNil(t, err)
					reusedLayer2History := history[len(history)-1]
					newLayer1History := history[len(history)-2]
					h.AssertEq(t, strings.Contains(reusedLayer2History.CreatedBy, "some-history-"), true)
					h.AssertEq(t, newLayer1History, v1.History{Created: v1.Time{Time: imgutil.NormalizedDateTime}})
				})
			})
		})
	})

	when("#ReuseLayerWithHistory", func() {
		when("previous image", func() {
			var (
				prevImage     *remote.Image
				prevImageName string
				prevLayer1SHA string
				prevLayer2SHA string
			)

			it.Before(func() {
				prevImageName = newTestImageName()
				var err error
				prevImage, err = remote.NewImage(
					prevImageName,
					authn.DefaultKeychain,
					remote.WithHistory(),
				)
				h.AssertNil(t, err)

				layer1Path, err := h.CreateSingleFileLayerTar("/layer-1.txt", "old-layer-1", "linux")
				h.AssertNil(t, err)
				defer os.Remove(layer1Path)

				prevLayer1SHA = h.FileDiffID(t, layer1Path)

				layer2Path, err := h.CreateSingleFileLayerTar("/layer-2.txt", "old-layer-2", "linux")
				h.AssertNil(t, err)
				defer os.Remove(layer2Path)

				prevLayer2SHA = h.FileDiffID(t, layer2Path)

				h.AssertNil(t, prevImage.AddLayer(layer1Path))
				h.AssertNil(t, prevImage.AddLayer(layer2Path))

				h.AssertNil(t, prevImage.Save())
			})

			it("reuses a layer with history", func() {
				img, err := remote.NewImage(
					repoName,
					authn.DefaultKeychain,
					remote.WithPreviousImage(prevImageName),
					remote.WithHistory(),
				)
				h.AssertNil(t, err)

				newBaseLayerPath, err := h.CreateSingleFileLayerTar("/new-base.txt", "base-content", "linux")
				h.AssertNil(t, err)
				defer os.Remove(newBaseLayerPath)

				h.AssertNil(t, img.AddLayer(newBaseLayerPath))

				err = img.ReuseLayerWithHistory(prevLayer2SHA, v1.History{CreatedBy: "some-new-history"})
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				manifestLayers := h.FetchManifestLayers(t, repoName)

				newLayer1SHA := h.StringElementAt(manifestLayers, -2)
				reusedLayer2SHA := h.StringElementAt(manifestLayers, -1)

				h.AssertNotEq(t, prevLayer1SHA, newLayer1SHA)
				h.AssertEq(t, prevLayer2SHA, reusedLayer2SHA)

				history, err := img.History()
				h.AssertNil(t, err)
				reusedLayer2History := history[len(history)-1]
				newLayer1History := history[len(history)-2]
				h.AssertEq(t, strings.Contains(reusedLayer2History.CreatedBy, "some-new-history"), true)
				h.AssertEq(t, newLayer1History, v1.History{Created: v1.Time{Time: imgutil.NormalizedDateTime}})
			})
		})
	})

	when("#Save", func() {
		when("image exists", func() {
			it("can be pulled by digest", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				err = img.SetLabel("mykey", "newValue")
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				identifier, err := img.Identifier()
				h.AssertNil(t, err)

				testImg, err := remote.NewImage(
					"test",
					authn.DefaultKeychain,
					remote.FromBaseImage(identifier.String()),
				)
				h.AssertNil(t, err)

				remoteLabel, err := testImg.Label("mykey")
				h.AssertNil(t, err)

				h.AssertEq(t, remoteLabel, "newValue")
			})

			it("can be pulled from unauthenticated registry", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				err = img.SetLabel("mykey", "newValue")
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				// convert authenticated repo name to unauthenticated repo name
				authRepoRef, err := name.ParseReference(repoName, name.WeakValidation)
				h.AssertNil(t, err)
				sharedImageName := authRepoRef.Context().RepositoryStr()
				readonlyRepoName := readonlyDockerRegistry.RepoName(sharedImageName)

				testImg, err := remote.NewImage(
					"test",
					authn.DefaultKeychain,
					remote.FromBaseImage(readonlyRepoName),
				)
				h.AssertNil(t, err)

				remoteLabel, err := testImg.Label("mykey")
				h.AssertNil(t, err)

				h.AssertEq(t, remoteLabel, "newValue")
			})

			it("zeroes all times and client specific fields", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				tarPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", "linux")
				h.AssertNil(t, err)
				defer os.Remove(tarPath)

				h.AssertNil(t, img.AddLayer(tarPath))

				h.AssertNil(t, img.Save())

				configFile := h.FetchManifestImageConfigFile(t, repoName)

				h.AssertEq(t, configFile.Created.Time, imgutil.NormalizedDateTime)
				h.AssertEq(t, configFile.Container, "")

				h.AssertEq(t, len(configFile.History), len(configFile.RootFS.DiffIDs))
				for _, item := range configFile.History {
					h.AssertEq(t, item.Created.Unix(), imgutil.NormalizedDateTime.Unix())
				}
			})

			when("the WithCreatedAt option is used", func() {
				it("uses the value for all times and client specific fields", func() {
					expectedTime := time.Date(2022, 1, 5, 5, 5, 5, 0, time.UTC)
					img, err := remote.NewImage(repoName, authn.DefaultKeychain,
						remote.WithCreatedAt(expectedTime),
					)
					h.AssertNil(t, err)

					tarPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", "linux")
					h.AssertNil(t, err)
					defer os.Remove(tarPath)

					h.AssertNil(t, img.AddLayer(tarPath))

					h.AssertNil(t, img.Save())

					configFile := h.FetchManifestImageConfigFile(t, repoName)

					h.AssertEq(t, configFile.Created.Time, expectedTime)
					h.AssertEq(t, configFile.Container, "")

					h.AssertEq(t, len(configFile.History), len(configFile.RootFS.DiffIDs))
					for _, item := range configFile.History {
						h.AssertEq(t, item.Created.Unix(), expectedTime.Unix())
					}
				})
			})
		})

		when("additional names are provided", func() {
			var (
				repoName            = newTestImageName()
				additionalRepoNames = []string{
					repoName + ":" + h.RandString(5),
					newTestImageName(),
					newTestImageName(),
				}
				successfulRepoNames = append([]string{repoName}, additionalRepoNames...)
			)

			it("saves to multiple names", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertNil(t, image.Save(additionalRepoNames...))
				for _, n := range successfulRepoNames {
					testImg, err := remote.NewImage(n, authn.DefaultKeychain)
					h.AssertNil(t, err)
					h.AssertEq(t, testImg.Found(), true)
				}
			})

			when("a single image name fails", func() {
				it("returns results with errors for those that failed", func() {
					failingName := newTestImageName() + ":🧨"

					image, err := remote.NewImage(repoName, authn.DefaultKeychain)
					h.AssertNil(t, err)

					err = image.Save(append([]string{failingName}, additionalRepoNames...)...)
					h.AssertError(t, err, fmt.Sprintf("failed to write image to the following tags: [%s:", failingName))

					// check all but failing name
					saveErr, ok := err.(imgutil.SaveError)
					h.AssertEq(t, ok, true)
					h.AssertEq(t, len(saveErr.Errors), 1)
					h.AssertEq(t, saveErr.Errors[0].ImageName, failingName)
					h.AssertError(t, saveErr.Errors[0].Cause, "could not parse reference")

					for _, n := range successfulRepoNames {
						testImg, err := remote.NewImage(n, authn.DefaultKeychain)
						h.AssertNil(t, err)
						h.AssertEq(t, testImg.Found(), true)
					}
				})
			})
		})
	})

	when("#Found", func() {
		when("it exists", func() {
			it("returns true, nil", func() {
				origImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, origImage.Save())

				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertEq(t, image.Found(), true)
			})
		})

		when("it does not exist", func() {
			it("returns false, nil", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertEq(t, image.Found(), false)
			})
		})
	})

	when("#Valid", func() {
		when("it exists", func() {
			it("returns true", func() {
				origImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, origImage.Save())

				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertEq(t, image.Valid(), true)
			})
		})

		when("it is corrupt", func() {
			it("returns false", func() {
				origImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				tarPath, _, _ := h.RandomLayer(t, t.TempDir())
				defer os.Remove(tarPath)
				h.AssertNil(t, origImage.AddLayer(tarPath))
				h.AssertNil(t, origImage.Save())

				// delete the top layer from the registry
				layers, err := origImage.UnderlyingImage().Layers()
				h.AssertNil(t, err)
				digest, err := layers[0].Digest()
				h.AssertNil(t, err)
				h.DeleteRegistryBlob(t, repoName, digest, dockerRegistry.EncodedAuth())

				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertEq(t, image.Valid(), false)
			})
		})

		when("it does not exist", func() {
			it("returns false", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertEq(t, image.Valid(), false)
			})
		})

		when("windows image index", func() {
			it("returns true", func() {
				ref := "mcr.microsoft.com/windows/nanoserver@sha256:eea54849888c8070ea35f8df39b3a5e126bc9a5bd30afdcad6f430408b2c786d"
				image, err := remote.NewImage(ref, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertEq(t, image.Valid(), true)
			})
		})
	})

	when("#Delete", func() {
		when("it exists", func() {
			var img imgutil.Image
			it("returns nil and is deleted", func() {
				origImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, origImage.SetLabel("some-label", "some-val"))
				h.AssertNil(t, origImage.Save())

				identifier, err := origImage.Identifier()
				h.AssertNil(t, err)

				repoID := identifier.String()

				img, err = remote.NewImage(
					repoID,
					authn.DefaultKeychain,
					remote.FromBaseImage(repoID),
				)

				h.AssertNil(t, err)
				h.AssertEq(t, img.Found(), true)

				h.AssertNil(t, img.Delete())

				h.AssertEq(t, img.Found(), false)
			})
		})

		when("it does not exists", func() {
			it("returns an error", func() {
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)

				h.AssertEq(t, img.Found(), false)
				h.AssertError(t, img.Delete(), "NAME_UNKNOWN")
			})
		})
	})

	when("#CheckReadAccess", func() {
		when("image exists in the registry and client has read access", func() {
			it.Before(func() {
				origImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, origImage.Save())
			})

			it("returns true", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				canRead, err := image.CheckReadAccess()
				h.AssertNil(t, err)
				h.AssertEq(t, canRead, true)
			})
		})

		when("image does not exist in the registry and client has read access", func() {
			it("returns true", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				canRead, err := image.CheckReadAccess()
				h.AssertNil(t, err)
				h.AssertEq(t, canRead, true)
			})
		})

		when("image does not exist in the registry and client doesn't have read access", func() {
			it.Before(func() {
				os.Unsetenv("DOCKER_CONFIG")
			})

			it.After(func() {
				os.Setenv("DOCKER_CONFIG", dockerRegistry.DockerDirectory)
			})

			it("returns false", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				canRead, _ := image.CheckReadAccess()
				h.AssertEq(t, canRead, false)
			})
		})

		when("using custom handler", func() {
			it.Before(func() {
				os.Setenv("DOCKER_CONFIG", customRegistry.DockerDirectory)
			})

			it.After(func() {
				os.Setenv("DOCKER_CONFIG", dockerRegistry.DockerDirectory)
			})

			when("image has read and write access", func() {
				it("returns true", func() {
					image, err := remote.NewImage(customRegistry.RepoName(readWriteImage), authn.DefaultKeychain)
					h.AssertNil(t, err)
					canRead, err := image.CheckReadAccess()
					h.AssertNil(t, err)
					h.AssertEq(t, canRead, true)
				})
			})

			when("image has read access but no write access", func() {
				it("returns true", func() {
					image, err := remote.NewImage(customRegistry.RepoName(readOnlyImage), authn.DefaultKeychain)
					h.AssertNil(t, err)
					canRead, err := image.CheckReadAccess()
					h.AssertNil(t, err)
					h.AssertEq(t, canRead, true)
				})
			})

			when("image doesn't have read access but has write access", func() {
				it("returns false", func() {
					image, err := remote.NewImage(customRegistry.RepoName(writeOnlyImage), authn.DefaultKeychain)
					h.AssertNil(t, err)
					canReadWrite, _ := image.CheckReadAccess()
					h.AssertEq(t, canReadWrite, false)
				})
			})

			when("image doesn't have read nor write access", func() {
				it("returns false", func() {
					image, err := remote.NewImage(customRegistry.RepoName(inaccessibleImage), authn.DefaultKeychain)
					h.AssertNil(t, err)
					canReadWrite, _ := image.CheckReadAccess()
					h.AssertEq(t, canReadWrite, false)
				})
			})
		})
	})

	when("#CheckReadWriteAccess", func() {
		when("image exists in the registry and client has read/write access", func() {
			it.Before(func() {
				origImage, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				h.AssertNil(t, origImage.Save())
			})

			it("returns true", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				canRead, err := image.CheckReadWriteAccess()
				h.AssertNil(t, err)
				h.AssertEq(t, canRead, true)
			})
		})

		when("image does not exist in the registry and client only has read access", func() {
			it.Before(func() {
				os.Setenv("DOCKER_CONFIG", readonlyDockerRegistry.DockerDirectory)
			})

			it.After(func() {
				os.Setenv("DOCKER_CONFIG", dockerRegistry.DockerDirectory)
			})

			it("returns false", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				canReadWrite, _ := image.CheckReadWriteAccess()
				h.AssertEq(t, canReadWrite, false)
			})
		})

		when("image does not exist in the registry and client has read/write access", func() {
			it("returns true", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				canRead, err := image.CheckReadWriteAccess()
				h.AssertNil(t, err)
				h.AssertEq(t, canRead, true)
			})
		})

		when("image does not exist in the registry and client doesn't have read/write access", func() {
			it.Before(func() {
				os.Unsetenv("DOCKER_CONFIG")
			})

			it.After(func() {
				os.Setenv("DOCKER_CONFIG", dockerRegistry.DockerDirectory)
			})

			it("returns false", func() {
				image, err := remote.NewImage(repoName, authn.DefaultKeychain)
				h.AssertNil(t, err)
				canReadWrite, _ := image.CheckReadWriteAccess()
				h.AssertEq(t, canReadWrite, false)
			})
		})

		when("using custom handler", func() {
			it.Before(func() {
				os.Setenv("DOCKER_CONFIG", customRegistry.DockerDirectory)
			})

			it.After(func() {
				os.Setenv("DOCKER_CONFIG", dockerRegistry.DockerDirectory)
			})

			when("image has read and write access", func() {
				it("returns true", func() {
					image, err := remote.NewImage(customRegistry.RepoName(readWriteImage), authn.DefaultKeychain)
					h.AssertNil(t, err)
					canRead, err := image.CheckReadWriteAccess()
					h.AssertNil(t, err)
					h.AssertEq(t, canRead, true)
				})
			})

			when("image has read access but no write access", func() {
				it("returns false", func() {
					image, err := remote.NewImage(customRegistry.RepoName(readOnlyImage), authn.DefaultKeychain)
					h.AssertNil(t, err)
					canReadWrite, _ := image.CheckReadWriteAccess()
					h.AssertEq(t, canReadWrite, false)
				})
			})

			when("image doesn't have read access but has write access", func() {
				it("returns true", func() {
					image, err := remote.NewImage(customRegistry.RepoName(writeOnlyImage), authn.DefaultKeychain)
					h.AssertNil(t, err)
					canReadWrite, _ := image.CheckReadWriteAccess()
					h.AssertEq(t, canReadWrite, false)
				})
			})

			when("image doesn't have read nor write access", func() {
				it("returns false", func() {
					image, err := remote.NewImage(customRegistry.RepoName(inaccessibleImage), authn.DefaultKeychain)
					h.AssertNil(t, err)
					canReadWrite, _ := image.CheckReadWriteAccess()
					h.AssertEq(t, canReadWrite, false)
				})
			})
		})
	})
}
