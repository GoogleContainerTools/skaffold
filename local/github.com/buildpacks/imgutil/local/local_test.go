package local_test

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/imgutil"
	local "github.com/buildpacks/imgutil/local"
	"github.com/buildpacks/imgutil/remote"
	h "github.com/buildpacks/imgutil/testhelpers"
)

const someSHA = "sha256:aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f"

var localTestRegistry *h.DockerRegistry

func TestLocal(t *testing.T) {
	localTestRegistry = h.NewDockerRegistry()
	localTestRegistry.Start(t)
	defer localTestRegistry.Stop(t)

	spec.Run(t, "Image", testImage, spec.Sequential(), spec.Report(report.Terminal{}))
}

func newTestImageName() string {
	return localTestRegistry.RepoName("pack-image-test-" + h.RandString(10))
}

func testImage(t *testing.T, when spec.G, it spec.S) {
	var (
		dockerClient          client.CommonAPIClient
		daemonOS              string
		daemonArchitecture    string
		runnableBaseImageName string
	)

	it.Before(func() {
		var err error
		dockerClient = h.DockerCli(t)

		daemonInfo, err := dockerClient.ServerVersion(context.TODO())
		h.AssertNil(t, err)

		daemonOS = daemonInfo.Os
		daemonArchitecture = daemonInfo.Arch
		runnableBaseImageName = h.RunnableBaseImage(daemonOS)
		h.PullIfMissing(t, dockerClient, runnableBaseImageName)
	})

	when("#NewImage", func() {
		when("no base image or platform is given", func() {
			it("returns an empty image", func() {
				_, err := local.NewImage(newTestImageName(), dockerClient)
				h.AssertNil(t, err)
			})

			it("sets sensible defaults from daemon for all required fields", func() {
				// os, architecture, and rootfs are required per https://github.com/opencontainers/image-spec/blob/master/config.md
				img, err := local.NewImage(newTestImageName(), dockerClient)
				h.AssertNil(t, err)
				h.AssertNil(t, img.Save())

				defer func() {
					err = h.DockerRmi(dockerClient, img.Name())
					h.AssertNil(t, err)
				}()
				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), img.Name())
				h.AssertNil(t, err)

				daemonInfo, err := dockerClient.ServerVersion(context.TODO())
				h.AssertNil(t, err)

				h.AssertEq(t, inspect.Os, daemonInfo.Os)
				h.AssertEq(t, inspect.Architecture, daemonInfo.Arch)
				h.AssertEq(t, inspect.RootFS.Type, "layers")
			})
		})

		when("#WithDefaultPlatform", func() {
			it("sets all available platform fields", func() {
				expectedArmArch := "arm64"
				expectedOSVersion := ""
				if daemonOS == "windows" {
					// windows/arm nanoserver image
					expectedArmArch = "arm"
					expectedOSVersion = "10.0.17763.1040"
				}

				img, err := local.NewImage(
					newTestImageName(),
					dockerClient,
					local.WithDefaultPlatform(imgutil.Platform{
						Architecture: expectedArmArch,
						OS:           daemonOS,
						OSVersion:    expectedOSVersion,
					}),
				)
				h.AssertNil(t, err)
				h.AssertNil(t, img.Save())

				defer func() {
					err = h.DockerRmi(dockerClient, img.Name())
					h.AssertNil(t, err)
				}()
				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), img.Name())
				h.AssertNil(t, err)

				daemonInfo, err := dockerClient.Info(context.TODO())
				h.AssertNil(t, err)

				// image os must match daemon
				h.AssertEq(t, inspect.Os, daemonInfo.OSType)
				h.AssertEq(t, inspect.Architecture, expectedArmArch)
				h.AssertEq(t, inspect.OsVersion, expectedOSVersion)
				h.AssertEq(t, inspect.RootFS.Type, "layers")

				// base layer is added for windows
				if daemonOS == "windows" {
					h.AssertEq(t, len(inspect.RootFS.Layers), 1)
				} else {
					h.AssertEq(t, len(inspect.RootFS.Layers), 0)
				}
			})
		})

		when("#FromBaseImage", func() {
			when("no platform is specified", func() {
				when("base image exists", func() {
					var baseImageName = newTestImageName()
					var repoName = newTestImageName()

					it.After(func() {
						h.AssertNil(t, h.DockerRmi(dockerClient, baseImageName))
					})

					it("returns the local image", func() {
						baseImage, err := local.NewImage(baseImageName, dockerClient)
						h.AssertNil(t, err)

						h.AssertNil(t, baseImage.SetEnv("MY_VAR", "my_val"))
						h.AssertNil(t, baseImage.SetLabel("some.label", "some.value"))
						h.AssertNil(t, baseImage.Save())

						localImage, err := local.NewImage(
							repoName,
							dockerClient,
							local.FromBaseImage(baseImageName),
						)
						h.AssertNil(t, err)

						labelValue, err := localImage.Label("some.label")
						h.AssertNil(t, err)
						h.AssertEq(t, labelValue, "some.value")
					})
				})

				when("base image does not exist", func() {
					it("returns an empty image", func() {
						img, err := local.NewImage(
							newTestImageName(),
							dockerClient,
							local.FromBaseImage("some-bad-repo-name"),
						)

						h.AssertNil(t, err)

						// base layer is added for windows
						if daemonOS == "windows" {
							topLayerDiffID, err := img.TopLayer()
							h.AssertNil(t, err)

							h.AssertNotEq(t, topLayerDiffID, "")
						} else {
							_, err = img.TopLayer()
							h.AssertError(t, err, "has no layers")
						}
					})
				})

				when("base image and daemon os/architecture match", func() {
					it("uses the base image architecture/OS", func() {
						img, err := local.NewImage(
							newTestImageName(),
							dockerClient,
							local.FromBaseImage(runnableBaseImageName),
						)
						h.AssertNil(t, err)
						h.AssertNil(t, img.Save())
						defer func() {
							err = h.DockerRmi(dockerClient, img.Name())
							h.AssertNil(t, err)
						}()

						imgOS, err := img.OS()
						h.AssertNil(t, err)
						h.AssertEq(t, imgOS, daemonOS)

						inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), img.Name())
						h.AssertNil(t, err)
						h.AssertEq(t, inspect.Os, daemonOS)
						h.AssertEq(t, inspect.Architecture, daemonArchitecture)
						h.AssertEq(t, inspect.RootFS.Type, "layers")

						h.AssertEq(t, img.Found(), true)
					})
				})

				when("base image and daemon architecture do not match", func() {
					it("uses the base image architecture/OS", func() {
						armBaseImageName := "busybox@sha256:50edf1d080946c6a76989d1c3b0e753b62f7d9b5f5e66e88bef23ebbd1e9709c"
						expectedArmArch := "arm64"
						expectedOSVersion := ""
						if daemonOS == "windows" {
							// windows/arm nanoserver image
							armBaseImageName = "mcr.microsoft.com/windows/nanoserver@sha256:29e2270953589a12de7a77a7e77d39e3b3e9cdfd243c922b3b8a63e2d8a71026"
							expectedArmArch = "arm"
							expectedOSVersion = "10.0.17763.1040"
						}

						h.PullIfMissing(t, dockerClient, armBaseImageName)

						img, err := local.NewImage(
							newTestImageName(),
							dockerClient,
							local.FromBaseImage(armBaseImageName),
						)
						h.AssertNil(t, err)
						h.AssertNil(t, img.Save())
						defer h.DockerRmi(dockerClient, img.Name())

						imgArch, err := img.Architecture()
						h.AssertNil(t, err)
						h.AssertEq(t, imgArch, expectedArmArch)

						imgOSVersion, err := img.OSVersion()
						h.AssertNil(t, err)
						h.AssertEq(t, imgOSVersion, expectedOSVersion)
					})
				})
			})

			when("#WithDefaultPlatform", func() {
				when("base image and platform architecture/OS do not match", func() {
					it("uses the base image architecture/OS, ignoring platform", func() {
						// linux/arm64 busybox image
						armBaseImageName := "busybox@sha256:50edf1d080946c6a76989d1c3b0e753b62f7d9b5f5e66e88bef23ebbd1e9709c"
						expectedArchitecture := "arm64"
						expectedOSVersion := ""
						if daemonOS == "windows" {
							// windows/arm nanoserver image
							armBaseImageName = "mcr.microsoft.com/windows/nanoserver@sha256:29e2270953589a12de7a77a7e77d39e3b3e9cdfd243c922b3b8a63e2d8a71026"
							expectedArchitecture = "arm"
							expectedOSVersion = "10.0.17763.1040"
						}

						h.PullIfMissing(t, dockerClient, armBaseImageName)

						img, err := local.NewImage(
							newTestImageName(),
							dockerClient,
							local.FromBaseImage(armBaseImageName),
							local.WithDefaultPlatform(imgutil.Platform{
								Architecture: "not-an-arch",
								OSVersion:    "10.0.99999.9999",
							}),
						)
						h.AssertNil(t, err)
						h.AssertNil(t, img.Save())
						defer h.DockerRmi(dockerClient, img.Name())

						imgArch, err := img.Architecture()
						h.AssertNil(t, err)
						h.AssertEq(t, imgArch, expectedArchitecture)

						imgOSVersion, err := img.OSVersion()
						h.AssertNil(t, err)
						h.AssertEq(t, imgOSVersion, expectedOSVersion)
					})
				})

				when("base image does not exist", func() {
					it("returns an empty image based on platform fields", func() {
						img, err := local.NewImage(
							newTestImageName(),
							dockerClient,
							local.FromBaseImage("some-bad-repo-name"),
							local.WithDefaultPlatform(imgutil.Platform{
								Architecture: "arm64",
								OS:           daemonOS,
								OSVersion:    "10.0.99999.9999",
							}),
						)

						h.AssertNil(t, err)
						h.AssertNil(t, img.Save())
						defer h.DockerRmi(dockerClient, img.Name())

						inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), img.Name())
						h.AssertNil(t, err)

						daemonInfo, err := dockerClient.Info(context.TODO())
						h.AssertNil(t, err)

						// image os must match daemon
						h.AssertEq(t, inspect.Os, daemonInfo.OSType)
						h.AssertEq(t, inspect.Architecture, "arm64")
						h.AssertEq(t, inspect.OsVersion, "10.0.99999.9999")

						// base layer is added for windows
						if daemonOS == "windows" {
							h.AssertEq(t, len(inspect.RootFS.Layers), 1)
						} else {
							h.AssertEq(t, len(inspect.RootFS.Layers), 0)
						}
					})
				})
			})
		})

		when("#WithPreviousImage", func() {
			when("previous image is exists", func() {
				var armBaseImageName string
				var existingLayerSHA string

				it.Before(func() {
					// linux/arm64 busybox image
					armBaseImageName = "busybox@sha256:50edf1d080946c6a76989d1c3b0e753b62f7d9b5f5e66e88bef23ebbd1e9709c"
					if daemonOS == "windows" {
						// windows/arm nanoserver image
						armBaseImageName = "mcr.microsoft.com/windows/nanoserver@sha256:29e2270953589a12de7a77a7e77d39e3b3e9cdfd243c922b3b8a63e2d8a71026"
					}
					h.PullIfMissing(t, dockerClient, armBaseImageName)

					refImage, err := local.NewImage(
						newTestImageName(),
						dockerClient,
						local.FromBaseImage(armBaseImageName),
					)
					h.AssertNil(t, err)

					existingLayerSHA, err = refImage.TopLayer()
					h.AssertNil(t, err)
				})

				it("provides reusable layers", func() {
					img, err := local.NewImage(
						newTestImageName(),
						dockerClient,
						local.WithPreviousImage(armBaseImageName),
					)
					h.AssertNil(t, err)

					h.AssertNil(t, img.ReuseLayer(existingLayerSHA))
				})

				it("provides reusable layers, ignoring WithDefaultPlatform", func() {
					img, err := local.NewImage(
						newTestImageName(),
						dockerClient,
						local.WithPreviousImage(armBaseImageName),
						local.WithDefaultPlatform(imgutil.Platform{
							Architecture: "some-fake-os",
						}),
					)
					h.AssertNil(t, err)

					h.AssertNil(t, img.ReuseLayer(existingLayerSHA))
				})
			})

			when("previous image does not exist", func() {
				it("does not error", func() {
					_, err := local.NewImage(
						newTestImageName(),
						dockerClient,
						local.WithPreviousImage("some-bad-repo-name"),
					)

					h.AssertNil(t, err)
				})
			})
		})

		when("#WithConfig", func() {
			var config = &v1.Config{Entrypoint: []string{"some-entrypoint"}}

			it("sets the image config", func() {
				localImage, err := local.NewImage(newTestImageName(), dockerClient, local.WithConfig(config))
				h.AssertNil(t, err)

				entrypoint, err := localImage.Entrypoint()
				h.AssertNil(t, err)
				h.AssertEq(t, entrypoint, []string{"some-entrypoint"})
			})

			when("#FromBaseImage", func() {
				var baseImageName = newTestImageName()

				it("overrides the base image config", func() {
					baseImage, err := local.NewImage(baseImageName, dockerClient)
					h.AssertNil(t, err)
					h.AssertNil(t, baseImage.Save())

					localImage, err := local.NewImage(
						newTestImageName(),
						dockerClient,
						local.WithConfig(config),
						local.FromBaseImage(baseImageName),
					)
					h.AssertNil(t, err)

					entrypoint, err := localImage.Entrypoint()
					h.AssertNil(t, err)
					h.AssertEq(t, entrypoint, []string{"some-entrypoint"})
				})

				it.After(func() {
					h.AssertNil(t, h.DockerRmi(dockerClient, baseImageName))
				})
			})
		})
	})

	when("#Labels", func() {
		when("image exists with labels", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, existingImage.SetLabel("mykey", "myvalue"))
				h.AssertNil(t, existingImage.SetLabel("other", "data"))
				h.AssertNil(t, existingImage.Save())
			})

			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			it("returns all the labels", func() {
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
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
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)
				h.AssertNil(t, existingImage.Save())
			})

			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			it("returns nil", func() {
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				labels, err := img.Labels()
				h.AssertNil(t, err)
				h.AssertEq(t, 0, len(labels))
			})
		})

		when("image NOT exists", func() {
			it("returns an empty map", func() {
				img, err := local.NewImage(newTestImageName(), dockerClient)
				h.AssertNil(t, err)

				labels, err := img.Labels()
				h.AssertNil(t, err)
				h.AssertEq(t, 0, len(labels))
			})
		})
	})

	when("#Label", func() {
		when("image exists", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, existingImage.SetLabel("mykey", "myvalue"))
				h.AssertNil(t, existingImage.SetLabel("other", "data"))
				h.AssertNil(t, existingImage.Save())
			})

			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			it("returns the label value", func() {
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				label, err := img.Label("mykey")
				h.AssertNil(t, err)
				h.AssertEq(t, label, "myvalue")
			})

			it("returns an empty string for a missing label", func() {
				img, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				label, err := img.Label("missing-label")
				h.AssertNil(t, err)
				h.AssertEq(t, label, "")
			})
		})

		when("image NOT exists", func() {
			it("returns an empty string", func() {
				img, err := local.NewImage(newTestImageName(), dockerClient)
				h.AssertNil(t, err)

				label, err := img.Label("some-label")
				h.AssertNil(t, err)
				h.AssertEq(t, label, "")
			})
		})
	})

	when("#Env", func() {
		when("image exists", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, existingImage.SetEnv("MY_VAR", "my_val"))
				h.AssertNil(t, existingImage.Save())
			})

			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			it("returns the label value", func() {
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.Env("MY_VAR")
				h.AssertNil(t, err)
				h.AssertEq(t, val, "my_val")
			})

			it("returns an empty string for a missing label", func() {
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.Env("MISSING_VAR")
				h.AssertNil(t, err)
				h.AssertEq(t, val, "")
			})
		})

		when("image NOT exists", func() {
			it("returns an empty string", func() {
				img, err := local.NewImage(newTestImageName(), dockerClient)
				h.AssertNil(t, err)

				val, err := img.Env("SOME_VAR")
				h.AssertNil(t, err)
				h.AssertEq(t, val, "")
			})
		})
	})

	when("#WorkingDir", func() {
		when("image exists", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, existingImage.SetWorkingDir("/testWorkingDir"))
				h.AssertNil(t, existingImage.Save())
			})

			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			it("returns the WorkingDir value", func() {
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.WorkingDir()
				h.AssertNil(t, err)

				h.AssertEq(t, val, "/testWorkingDir")
			})

			it("returns empty string for missing WorkingDir", func() {
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)
				h.AssertNil(t, existingImage.Save())

				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.WorkingDir()
				h.AssertNil(t, err)
				var expected string
				h.AssertEq(t, val, expected)
			})
		})

		when("image NOT exists", func() {
			it("returns empty string", func() {
				img, err := local.NewImage(newTestImageName(), dockerClient)
				h.AssertNil(t, err)

				val, err := img.WorkingDir()
				h.AssertNil(t, err)
				var expected string
				h.AssertEq(t, val, expected)
			})
		})
	})

	when("#Entrypoint", func() {
		when("image exists", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, existingImage.SetEntrypoint("entrypoint1", "entrypoint2"))
				h.AssertNil(t, existingImage.Save())
			})

			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			it("returns the entrypoint value", func() {
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.Entrypoint()
				h.AssertNil(t, err)

				h.AssertEq(t, val, []string{"entrypoint1", "entrypoint2"})
			})

			it("returns nil slice for a missing entrypoint", func() {
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)
				h.AssertNil(t, existingImage.Save())

				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				val, err := img.Entrypoint()
				h.AssertNil(t, err)
				var expected []string
				h.AssertEq(t, val, expected)
			})
		})

		when("image NOT exists", func() {
			it("returns nil slice", func() {
				img, err := local.NewImage(newTestImageName(), dockerClient)
				h.AssertNil(t, err)

				val, err := img.Entrypoint()
				h.AssertNil(t, err)
				var expected []string
				h.AssertEq(t, val, expected)
			})
		})
	})

	when("#Name", func() {
		it("always returns the original name", func() {
			var repoName = newTestImageName()

			img, err := local.NewImage(repoName, dockerClient)
			h.AssertNil(t, err)

			h.AssertEq(t, img.Name(), repoName)
		})
	})

	when("#CreatedAt", func() {
		it("returns the containers created at time", func() {
			img, err := local.NewImage(newTestImageName(), dockerClient, local.FromBaseImage(runnableBaseImageName))
			h.AssertNil(t, err)

			// based on static base image refs
			expectedTime := time.Date(2018, 10, 2, 17, 19, 34, 239926273, time.UTC)
			if daemonOS == "windows" {
				expectedTime = time.Date(2020, 03, 04, 13, 28, 48, 673000000, time.UTC)
			}

			createdTime, err := img.CreatedAt()

			h.AssertNil(t, err)
			h.AssertEq(t, createdTime, expectedTime)
		})
	})

	when("#Identifier", func() {
		var repoName = newTestImageName()
		var baseImageName = newTestImageName()

		it.Before(func() {
			baseImage, err := local.NewImage(baseImageName, dockerClient)
			h.AssertNil(t, err)

			h.AssertNil(t, baseImage.SetLabel("existingLabel", "existingValue"))
			h.AssertNil(t, baseImage.Save())
		})

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, baseImageName))
		})

		it("returns an Docker Image ID type identifier", func() {
			img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(baseImageName))
			h.AssertNil(t, err)

			id, err := img.Identifier()
			h.AssertNil(t, err)

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), id.String())
			h.AssertNil(t, err)
			labelValue := inspect.Config.Labels["existingLabel"]
			h.AssertEq(t, labelValue, "existingValue")
		})

		when("the image has been modified and saved", func() {
			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			it("returns the new image ID", func() {
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(baseImageName))
				h.AssertNil(t, err)

				h.AssertNil(t, img.SetLabel("new", "label"))

				h.AssertNil(t, img.Save())
				h.AssertNil(t, err)

				id, err := img.Identifier()
				h.AssertNil(t, err)

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), id.String())
				h.AssertNil(t, err)

				label := inspect.Config.Labels["new"]
				h.AssertEq(t, strings.TrimSpace(label), "label")
			})
		})
	})

	when("#Kind", func() {
		it("returns local", func() {
			img, err := local.NewImage(newTestImageName(), dockerClient)
			h.AssertNil(t, err)
			h.AssertEq(t, img.Kind(), "local")
		})
	})

	when("#SetLabel", func() {
		var (
			img           imgutil.Image
			repoName      = newTestImageName()
			baseImageName = newTestImageName()
		)

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, repoName, baseImageName))
		})

		when("base image has labels", func() {
			it("sets label and saves label to docker daemon", func() {
				var err error

				baseImage, err := local.NewImage(baseImageName, dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, baseImage.SetLabel("some-key", "some-value"))
				h.AssertNil(t, baseImage.Save())

				img, err = local.NewImage(repoName, dockerClient, local.FromBaseImage(baseImageName))
				h.AssertNil(t, err)

				h.AssertNil(t, img.SetLabel("somekey", "new-val"))

				label, err := img.Label("somekey")
				h.AssertNil(t, err)
				h.AssertEq(t, label, "new-val")

				h.AssertNil(t, img.Save())

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)
				label = inspect.Config.Labels["somekey"]
				h.AssertEq(t, strings.TrimSpace(label), "new-val")
			})
		})

		when("no labels exists", func() {
			it("sets label and saves label to docker daemon", func() {
				var err error
				baseImage, err := local.NewImage(baseImageName, dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, baseImage.SetCmd("/usr/bin/run"))
				h.AssertNil(t, baseImage.Save())

				img, err = local.NewImage(repoName, dockerClient, local.FromBaseImage(baseImageName))
				h.AssertNil(t, err)

				h.AssertNil(t, img.SetLabel("somekey", "new-val"))

				label, err := img.Label("somekey")
				h.AssertNil(t, err)
				h.AssertEq(t, label, "new-val")

				h.AssertNil(t, img.Save())

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)

				label = inspect.Config.Labels["somekey"]
				h.AssertEq(t, strings.TrimSpace(label), "new-val")
			})
		})
	})

	when("#RemoveLabel", func() {
		var (
			img           imgutil.Image
			repoName      = newTestImageName()
			baseImageName = newTestImageName()
		)

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, repoName, baseImageName))
		})

		when("image exists", func() {
			it("removes matching label on img object", func() {
				var err error

				baseImage, err := local.NewImage(baseImageName, dockerClient)
				h.AssertNil(t, err)
				h.AssertNil(t, baseImage.SetLabel("my.custom.label", "old-value"))
				h.AssertNil(t, baseImage.Save())

				img, err = local.NewImage(repoName, dockerClient, local.FromBaseImage(baseImageName))
				h.AssertNil(t, err)

				h.AssertNil(t, img.RemoveLabel("my.custom.label"))
				h.AssertNil(t, img.Save())

				labels, err := img.Labels()
				h.AssertNil(t, err)
				_, exists := labels["my.custom.label"]
				h.AssertEq(t, exists, false)
			})

			it("saves removal of the label", func() {
				var err error

				baseImage, err := local.NewImage(baseImageName, dockerClient)
				h.AssertNil(t, err)
				h.AssertNil(t, baseImage.SetLabel("my.custom.label", "old-value"))
				h.AssertNil(t, baseImage.Save())

				img, err = local.NewImage(repoName, dockerClient, local.FromBaseImage(baseImageName))
				h.AssertNil(t, err)

				h.AssertNil(t, img.RemoveLabel("my.custom.label"))
				h.AssertNil(t, img.Save())

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)
				_, exists := inspect.Config.Labels["my.custom.label"]
				h.AssertEq(t, exists, false)
			})
		})
	})

	when("#SetEnv", func() {
		var repoName = newTestImageName()
		var skipCleanup bool

		it.After(func() {
			if !skipCleanup {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			}
		})

		it("sets the environment", func() {
			img, err := local.NewImage(repoName, dockerClient)
			h.AssertNil(t, err)

			err = img.SetEnv("ENV_KEY", "ENV_VAL")
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			h.AssertContains(t, inspect.Config.Env, "ENV_KEY=ENV_VAL")
		})

		when("the key already exists", func() {
			it("overrides the existing key", func() {
				img, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				err = img.SetEnv("ENV_KEY", "SOME_VAL")
				h.AssertNil(t, err)

				err = img.SetEnv("ENV_KEY", "SOME_OTHER_VAL")
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)

				h.AssertContains(t, inspect.Config.Env, "ENV_KEY=SOME_OTHER_VAL")
				h.AssertDoesNotContain(t, inspect.Config.Env, "ENV_KEY=SOME_VAL")
			})

			when("windows", func() {
				it("ignores case", func() {
					if daemonOS != "windows" {
						skipCleanup = true
						t.Skip("windows test")
					}

					img, err := local.NewImage(repoName, dockerClient)
					h.AssertNil(t, err)

					err = img.SetEnv("ENV_KEY", "SOME_VAL")
					h.AssertNil(t, err)

					err = img.SetEnv("env_key", "SOME_OTHER_VAL")
					h.AssertNil(t, err)

					err = img.SetEnv("env_key2", "SOME_VAL")
					h.AssertNil(t, err)

					err = img.SetEnv("ENV_KEY2", "SOME_OTHER_VAL")
					h.AssertNil(t, err)

					h.AssertNil(t, img.Save())

					inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
					h.AssertNil(t, err)

					h.AssertContains(t, inspect.Config.Env, "env_key=SOME_OTHER_VAL")
					h.AssertDoesNotContain(t, inspect.Config.Env, "ENV_KEY=SOME_VAL")
					h.AssertDoesNotContain(t, inspect.Config.Env, "ENV_KEY=SOME_OTHER_VAL")

					h.AssertContains(t, inspect.Config.Env, "ENV_KEY2=SOME_OTHER_VAL")
					h.AssertDoesNotContain(t, inspect.Config.Env, "env_key2=SOME_OTHER_VAL")
					h.AssertDoesNotContain(t, inspect.Config.Env, "env_key2=SOME_VAL")
				})
			})
		})
	})

	when("#SetWorkingDir", func() {
		var repoName = newTestImageName()

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
		})

		it("sets the environment", func() {
			img, err := local.NewImage(repoName, dockerClient)
			h.AssertNil(t, err)

			err = img.SetWorkingDir("/some/work/dir")
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			h.AssertEq(t, inspect.Config.WorkingDir, "/some/work/dir")
		})
	})

	when("#SetEntrypoint", func() {
		var repoName = newTestImageName()

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
		})

		it("sets the entrypoint", func() {
			img, err := local.NewImage(repoName, dockerClient)
			h.AssertNil(t, err)

			err = img.SetEntrypoint("some", "entrypoint")
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			h.AssertEq(t, []string(inspect.Config.Entrypoint), []string{"some", "entrypoint"})
		})
	})

	when("#SetCmd", func() {
		var repoName = newTestImageName()

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
		})

		it("sets the cmd", func() {
			img, err := local.NewImage(repoName, dockerClient)
			h.AssertNil(t, err)

			err = img.SetCmd("some", "cmd")
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			h.AssertEq(t, []string(inspect.Config.Cmd), []string{"some", "cmd"})
		})
	})

	when("#SetOS", func() {
		var repoName = newTestImageName()

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
		})

		it("allows noop sets for values that match the daemon", func() {
			img, err := local.NewImage(repoName, dockerClient)
			h.AssertNil(t, err)

			err = img.SetOS("fakeos")
			h.AssertError(t, err, "invalid os: must match the daemon")

			err = img.SetOS(daemonOS)
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			h.AssertEq(t, inspect.Os, daemonOS)
		})
	})

	when("#SetOSVersion #SetArchitecture", func() {
		var repoName = newTestImageName()

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
		})

		it("sets the os.version/arch", func() {
			img, err := local.NewImage(repoName, dockerClient)
			h.AssertNil(t, err)

			err = img.SetOSVersion("1.2.3.4")
			h.AssertNil(t, err)

			err = img.SetArchitecture("arm64")
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			h.AssertEq(t, inspect.OsVersion, "1.2.3.4")
			h.AssertEq(t, inspect.Architecture, "arm64")
		})
	})

	when("#Rebase", func() {
		when("image exists", func() {
			var (
				repoName                      = newTestImageName()
				oldBase, oldTopLayer, newBase string
				oldBaseLayer1DiffID           string
				oldBaseLayer2DiffID           string
				newBaseLayer1DiffID           string
				newBaseLayer2DiffID           string
				imgLayer1DiffID               string
				imgLayer2DiffID               string
				origNumLayers                 int
			)

			it.Before(func() {
				// new base image
				newBase = "pack-newbase-test-" + h.RandString(10)
				newBaseImage, err := local.NewImage(newBase, dockerClient, local.FromBaseImage(runnableBaseImageName))
				h.AssertNil(t, err)

				newBaseLayer1Path, err := h.CreateSingleFileLayerTar("/new-base.txt", "new-base", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(newBaseLayer1Path)

				newBaseLayer1DiffID = h.FileDiffID(t, newBaseLayer1Path)

				newBaseLayer2Path, err := h.CreateSingleFileLayerTar("/otherfile.txt", "text-new-base", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(newBaseLayer2Path)

				newBaseLayer2DiffID = h.FileDiffID(t, newBaseLayer2Path)

				h.AssertNil(t, newBaseImage.AddLayer(newBaseLayer1Path))
				h.AssertNil(t, newBaseImage.AddLayer(newBaseLayer2Path))

				h.AssertNil(t, newBaseImage.Save())

				// old base image
				oldBase = "pack-oldbase-test-" + h.RandString(10)
				oldBaseImage, err := local.NewImage(oldBase, dockerClient, local.FromBaseImage(runnableBaseImageName))
				h.AssertNil(t, err)

				oldBaseLayer1Path, err := h.CreateSingleFileLayerTar("/old-base.txt", "old-base", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(oldBaseLayer1Path)

				oldBaseLayer1DiffID = h.FileDiffID(t, oldBaseLayer1Path)

				oldBaseLayer2Path, err := h.CreateSingleFileLayerTar("/otherfile.txt", "text-old-base", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(oldBaseLayer2Path)

				oldBaseLayer2DiffID = h.FileDiffID(t, oldBaseLayer2Path)

				h.AssertNil(t, oldBaseImage.AddLayer(oldBaseLayer1Path))
				h.AssertNil(t, oldBaseImage.AddLayer(oldBaseLayer2Path))

				h.AssertNil(t, oldBaseImage.Save())

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), oldBase)
				h.AssertNil(t, err)
				oldTopLayer = h.StringElementAt(inspect.RootFS.Layers, -1)

				// original image
				origImage, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(oldBase))
				h.AssertNil(t, err)

				imgLayer1Path, err := h.CreateSingleFileLayerTar("/myimage.txt", "text-from-image", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(imgLayer1Path)

				imgLayer1DiffID = h.FileDiffID(t, imgLayer1Path)

				imgLayer2Path, err := h.CreateSingleFileLayerTar("/myimage2.txt", "text-from-image", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(imgLayer2Path)

				imgLayer2DiffID = h.FileDiffID(t, imgLayer2Path)

				h.AssertNil(t, origImage.AddLayer(imgLayer1Path))
				h.AssertNil(t, origImage.AddLayer(imgLayer2Path))

				h.AssertNil(t, origImage.Save())

				inspect, _, err = dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)
				origNumLayers = len(inspect.RootFS.Layers)
			})

			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName, oldBase, newBase))
			})

			it("switches the base", func() {
				// Before
				beforeInspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)

				beforeOldBaseLayer1DiffID := h.StringElementAt(beforeInspect.RootFS.Layers, -4)
				h.AssertEq(t, oldBaseLayer1DiffID, beforeOldBaseLayer1DiffID)

				beforeOldBaseLayer2DiffID := h.StringElementAt(beforeInspect.RootFS.Layers, -3)
				h.AssertEq(t, oldBaseLayer2DiffID, beforeOldBaseLayer2DiffID)

				beforeLayer3DiffID := h.StringElementAt(beforeInspect.RootFS.Layers, -2)
				h.AssertEq(t, imgLayer1DiffID, beforeLayer3DiffID)

				beforeLayer4DiffID := h.StringElementAt(beforeInspect.RootFS.Layers, -1)
				h.AssertEq(t, imgLayer2DiffID, beforeLayer4DiffID)

				// Run rebase
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)
				newBaseImg, err := local.NewImage(newBase, dockerClient, local.FromBaseImage(newBase))
				h.AssertNil(t, err)
				err = img.Rebase(oldTopLayer, newBaseImg)
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				// After
				afterInspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)

				numLayers := len(afterInspect.RootFS.Layers)
				h.AssertEq(t, numLayers, origNumLayers)

				afterLayer1DiffID := h.StringElementAt(afterInspect.RootFS.Layers, -4)
				h.AssertEq(t, newBaseLayer1DiffID, afterLayer1DiffID)

				afterLayer2DiffID := h.StringElementAt(afterInspect.RootFS.Layers, -3)
				h.AssertEq(t, newBaseLayer2DiffID, afterLayer2DiffID)

				afterLayer3DiffID := h.StringElementAt(afterInspect.RootFS.Layers, -2)
				h.AssertEq(t, imgLayer1DiffID, afterLayer3DiffID)

				afterLayer4DiffID := h.StringElementAt(afterInspect.RootFS.Layers, -1)
				h.AssertEq(t, imgLayer2DiffID, afterLayer4DiffID)

				h.AssertEq(t, afterInspect.Os, beforeInspect.Os)
				h.AssertEq(t, afterInspect.OsVersion, beforeInspect.OsVersion)
				h.AssertEq(t, afterInspect.Architecture, beforeInspect.Architecture)
			})
		})
	})

	when("#TopLayer", func() {
		when("image exists", func() {
			var (
				expectedTopLayer string
				repoName         = newTestImageName()
			)
			it.Before(func() {
				existingImage, err := local.NewImage(
					repoName,
					dockerClient,
					local.FromBaseImage(runnableBaseImageName),
				)
				h.AssertNil(t, err)

				layer1Path, err := h.CreateSingleFileLayerTar("/newfile.txt", "old-base", daemonOS)
				h.AssertNil(t, err)
				layer2Path, err := h.CreateSingleFileLayerTar("/otherfile.txt", "text-old-base", daemonOS)
				h.AssertNil(t, err)

				h.AssertNil(t, existingImage.AddLayer(layer1Path))
				h.AssertNil(t, existingImage.AddLayer(layer2Path))

				h.AssertNil(t, existingImage.Save())

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)
				expectedTopLayer = h.StringElementAt(inspect.RootFS.Layers, -1)
			})

			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			it("returns the digest for the top layer (useful for rebasing)", func() {
				img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				actualTopLayer, err := img.TopLayer()
				h.AssertNil(t, err)

				h.AssertEq(t, actualTopLayer, expectedTopLayer)
			})
		})

		when("image has no layers", func() {
			it("returns error", func() {
				img, err := local.NewImage(newTestImageName(), dockerClient)
				h.AssertNil(t, err)

				if daemonOS == "windows" {
					layer, err := img.TopLayer()
					h.AssertNil(t, err)
					h.AssertNotEq(t, layer, "")
				} else {
					_, err = img.TopLayer()
					h.AssertError(t, err, "has no layers")
				}
			})
		})
	})

	when("#AddLayer", func() {
		when("empty image", func() {
			var repoName = newTestImageName()

			it("appends a layer", func() {
				img, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				newLayerPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(newLayerPath)

				newLayerDiffID := h.FileDiffID(t, newLayerPath)

				h.AssertNil(t, img.AddLayer(newLayerPath))

				h.AssertNil(t, img.Save())
				defer h.DockerRmi(dockerClient, repoName)

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)

				h.AssertEq(t, newLayerDiffID, h.StringElementAt(inspect.RootFS.Layers, -1))
			})
		})

		when("base image exists", func() {
			var (
				repoName      = newTestImageName()
				baseImageName = newTestImageName()
			)

			it("appends a layer", func() {
				baseImage, err := local.NewImage(
					baseImageName,
					dockerClient,
					local.FromBaseImage(runnableBaseImageName),
				)
				h.AssertNil(t, err)

				oldLayerPath, err := h.CreateSingleFileLayerTar("/old-layer.txt", "old-layer", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(oldLayerPath)

				oldLayerDiffID := h.FileDiffID(t, oldLayerPath)

				h.AssertNil(t, baseImage.AddLayer(oldLayerPath))

				h.AssertNil(t, baseImage.Save())
				defer h.DockerRmi(dockerClient, baseImageName)

				img, err := local.NewImage(
					repoName,
					dockerClient,
					local.FromBaseImage(baseImageName),
				)
				h.AssertNil(t, err)

				newLayerPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(newLayerPath)

				newLayerDiffID := h.FileDiffID(t, newLayerPath)

				h.AssertNil(t, img.AddLayer(newLayerPath))

				h.AssertNil(t, img.Save())
				defer h.DockerRmi(dockerClient, repoName)

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)

				h.AssertEq(t, oldLayerDiffID, h.StringElementAt(inspect.RootFS.Layers, -2))
				h.AssertEq(t, newLayerDiffID, h.StringElementAt(inspect.RootFS.Layers, -1))
			})
		})
	})

	when("#AddLayerWithDiffID", func() {
		it("appends a layer", func() {
			repoName := newTestImageName()

			existingImage, err := local.NewImage(repoName, dockerClient)
			h.AssertNil(t, err)

			oldLayerPath, err := h.CreateSingleFileLayerTar("/old-layer.txt", "old-layer", daemonOS)
			h.AssertNil(t, err)
			defer os.Remove(oldLayerPath)

			oldLayerDiffID := h.FileDiffID(t, oldLayerPath)

			h.AssertNil(t, existingImage.AddLayer(oldLayerPath))

			h.AssertNil(t, existingImage.Save())

			id, err := existingImage.Identifier()
			h.AssertNil(t, err)

			existingImageID := id.String()
			defer h.DockerRmi(dockerClient, existingImageID)

			img, err := local.NewImage(
				repoName,
				dockerClient,
				local.FromBaseImage(repoName),
			)
			h.AssertNil(t, err)

			newLayerPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", daemonOS)
			h.AssertNil(t, err)
			defer os.Remove(newLayerPath)

			newLayerDiffID := h.FileDiffID(t, newLayerPath)

			h.AssertNil(t, img.AddLayerWithDiffID(newLayerPath, newLayerDiffID))
			h.AssertNil(t, img.Save())
			defer h.DockerRmi(dockerClient, repoName)

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			h.AssertEq(t, oldLayerDiffID, h.StringElementAt(inspect.RootFS.Layers, -2))
			h.AssertEq(t, newLayerDiffID, h.StringElementAt(inspect.RootFS.Layers, -1))
		})
	})

	when("#AddLayerWithDiffIDAndHistory", func() {
		it("appends a layer", func() {
			repoName := newTestImageName()

			existingImage, err := local.NewImage(repoName, dockerClient)
			h.AssertNil(t, err)

			oldLayerPath, err := h.CreateSingleFileLayerTar("/old-layer.txt", "old-layer", daemonOS)
			h.AssertNil(t, err)
			defer os.Remove(oldLayerPath)

			oldLayerDiffID := h.FileDiffID(t, oldLayerPath)

			h.AssertNil(t, existingImage.AddLayer(oldLayerPath))

			h.AssertNil(t, existingImage.Save())

			id, err := existingImage.Identifier()
			h.AssertNil(t, err)

			existingImageID := id.String()
			defer h.DockerRmi(dockerClient, existingImageID)

			img, err := local.NewImage(
				repoName,
				dockerClient,
				local.FromBaseImage(repoName),
				local.WithHistory(),
			)
			h.AssertNil(t, err)

			newLayerPath, err := h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", daemonOS)
			h.AssertNil(t, err)
			defer os.Remove(newLayerPath)

			newLayerDiffID := h.FileDiffID(t, newLayerPath)

			oldHistory, err := img.History()
			h.AssertNil(t, err)
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
			imageReportsHistory, err := img.History()
			h.AssertNil(t, err)
			h.AssertEq(t, len(imageReportsHistory), len(oldHistory)+1)
			h.AssertEq(t, imageReportsHistory[len(imageReportsHistory)-1], addedHistory)

			daemonReportsHistory, err := dockerClient.ImageHistory(context.TODO(), repoName)
			h.AssertNil(t, err)
			h.AssertEq(t, len(imageReportsHistory), len(daemonReportsHistory))
			lastHistory := daemonReportsHistory[0] // the daemon reports history in reverse order
			h.AssertEq(t, lastHistory.CreatedBy, "some-history")

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			h.AssertEq(t, oldLayerDiffID, h.StringElementAt(inspect.RootFS.Layers, -2))
			h.AssertEq(t, newLayerDiffID, h.StringElementAt(inspect.RootFS.Layers, -1))
		})
	})

	when("#GetLayer", func() {
		when("the layer exists", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				var err error

				existingImage, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(runnableBaseImageName))
				h.AssertNil(t, err)

				layerPath, err := h.CreateSingleFileLayerTar("/file.txt", "file-contents", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(layerPath)

				h.AssertNil(t, existingImage.AddLayer(layerPath))

				h.AssertNil(t, existingImage.Save())
			})

			it.After(func() {
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			when("the layer exists", func() {
				it("returns a layer tar", func() {
					img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
					h.AssertNil(t, err)

					topLayer, err := img.TopLayer()
					h.AssertNil(t, err)

					r, err := img.GetLayer(topLayer)
					h.AssertNil(t, err)
					tr := tar.NewReader(r)

					// continue until reader is at matching file
					for {
						header, err := tr.Next()
						h.AssertNil(t, err)

						if strings.HasSuffix(header.Name, "/file.txt") {
							break
						}
					}

					contents := make([]byte, len("file-contents"))
					_, err = tr.Read(contents)
					if err != io.EOF {
						t.Fatalf("expected end of file: %x", err)
					}
					h.AssertEq(t, string(contents), "file-contents")
				})
			})

			when("the layer does not exist", func() {
				it("returns an error", func() {
					img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
					h.AssertNil(t, err)
					h.AssertNil(t, err)
					_, err = img.GetLayer(someSHA)
					h.AssertError(t, err, fmt.Sprintf("failed to find layer with diff ID %q", someSHA))
					_, ok := err.(imgutil.ErrLayerNotFound)
					h.AssertEq(t, ok, true)
				})
			})
		})

		when("image does NOT exist", func() {
			it("returns error", func() {
				image, err := local.NewImage("not-exist", dockerClient)
				h.AssertNil(t, err)

				readCloser, err := image.GetLayer(someSHA)
				h.AssertNil(t, readCloser)
				h.AssertError(t, err, fmt.Sprintf("failed to find layer with diff ID %q", someSHA))
				_, ok := err.(imgutil.ErrLayerNotFound)
				h.AssertEq(t, ok, true)
			})
		})
	})

	when("#ReuseLayer", func() {
		var (
			prevImage     *local.Image
			prevImageName = newTestImageName()
			repoName      = newTestImageName()
			prevLayer1SHA string
			prevLayer2SHA string
			layer1Path    string
			layer2Path    string
		)

		it.Before(func() {
			var err error
			prevImage, err = local.NewImage(
				prevImageName,
				dockerClient,
				local.FromBaseImage(runnableBaseImageName),
				local.WithHistory(),
			)
			h.AssertNil(t, err)

			layer1Path, err = h.CreateSingleFileLayerTar("/layer-1.txt", "old-layer-1", daemonOS)
			h.AssertNil(t, err)

			layer2Path, err = h.CreateSingleFileLayerTar("/layer-2.txt", "old-layer-2", daemonOS)
			h.AssertNil(t, err)

			h.AssertNil(t, prevImage.AddLayer(layer1Path))
			h.AssertNil(t, prevImage.AddLayer(layer2Path))

			h.AssertNil(t, prevImage.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), prevImageName)
			h.AssertNil(t, err)

			prevLayer1SHA = h.StringElementAt(inspect.RootFS.Layers, -2)
			prevLayer2SHA = h.StringElementAt(inspect.RootFS.Layers, -1)
		})

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, repoName, prevImageName))
			h.AssertNil(t, os.RemoveAll(layer1Path))
			h.AssertNil(t, os.RemoveAll(layer2Path))
		})

		it("reuses a layer", func() {
			img, err := local.NewImage(
				repoName,
				dockerClient,
				local.WithPreviousImage(prevImageName),
				local.FromBaseImage(runnableBaseImageName),
			)
			h.AssertNil(t, err)

			newLayer1Path, err := h.CreateSingleFileLayerTar("/new-base.txt", "base-content", daemonOS)
			h.AssertNil(t, err)
			defer os.Remove(newLayer1Path)

			h.AssertNil(t, img.AddLayer(newLayer1Path))

			err = img.ReuseLayer(prevLayer2SHA)
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			newLayer1SHA := h.StringElementAt(inspect.RootFS.Layers, -2)
			reusedLayer2SHA := h.StringElementAt(inspect.RootFS.Layers, -1)

			h.AssertNotEq(t, prevLayer1SHA, newLayer1SHA)
			h.AssertEq(t, prevLayer2SHA, reusedLayer2SHA)
		})

		it("does not download the old image if layers are directly above (performance)", func() {
			// FIXME: npa: not sure this test validates what is claimed in the `it`; it looks like even the `local` package
			// always downloads the previous image layers whenever `ReuseLayer` is called.
			img, err := local.NewImage(
				repoName,
				dockerClient,
				local.WithPreviousImage(prevImageName),
			)
			h.AssertNil(t, err)

			err = img.ReuseLayer(prevLayer1SHA)
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			if daemonOS == "windows" {
				h.AssertEq(t, len(inspect.RootFS.Layers), 2)
			} else {
				h.AssertEq(t, len(inspect.RootFS.Layers), 1)
			}

			newLayer1SHA := h.StringElementAt(inspect.RootFS.Layers, -1)

			h.AssertEq(t, prevLayer1SHA, newLayer1SHA)
		})

		when("there is history", func() {
			var prevHistory []v1.History

			it.Before(func() {
				// get the number of layers
				baseImage, err := remote.NewImage(
					repoName,
					authn.DefaultKeychain,
					remote.FromBaseImage(runnableBaseImageName),
				)
				h.AssertNil(t, err)
				layers, err := baseImage.UnderlyingImage().Layers()
				h.AssertNil(t, err)
				nLayers := len(layers) + 2 // added two layers in the test setup
				// add history
				prevHistory = make([]v1.History, nLayers)
				for idx := range prevHistory {
					prevHistory[idx].CreatedBy = fmt.Sprintf("some-history-%d", idx)
				}
				h.AssertNil(t, prevImage.SetHistory(prevHistory))
				h.AssertNil(t, prevImage.Save())
			})

			it("reuses a layer with history", func() {
				img, err := local.NewImage(
					repoName,
					dockerClient,
					local.WithPreviousImage(prevImageName),
					local.FromBaseImage(runnableBaseImageName),
					local.WithHistory(),
				)
				h.AssertNil(t, err)

				newBaseLayerPath, err := h.CreateSingleFileLayerTar("/new-base.txt", "base-content", daemonOS)
				h.AssertNil(t, err)
				defer os.Remove(newBaseLayerPath)

				h.AssertNil(t, img.AddLayer(newBaseLayerPath))

				err = img.ReuseLayer(prevLayer2SHA)
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)

				newLayer1SHA := h.StringElementAt(inspect.RootFS.Layers, -2)
				reusedLayer2SHA := h.StringElementAt(inspect.RootFS.Layers, -1)

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

	when("#ReuseLayerWithHistory", func() {
		var (
			prevImage     *local.Image
			prevImageName = newTestImageName()
			repoName      = newTestImageName()
			prevLayer1SHA string
			prevLayer2SHA string
		)

		it.Before(func() {
			var err error
			prevImage, err = local.NewImage(
				prevImageName,
				dockerClient,
				local.FromBaseImage(runnableBaseImageName),
				local.WithHistory(),
			)
			h.AssertNil(t, err)

			layer1Path, err := h.CreateSingleFileLayerTar("/layer-1.txt", "old-layer-1", daemonOS)
			h.AssertNil(t, err)
			defer os.Remove(layer1Path)

			layer2Path, err := h.CreateSingleFileLayerTar("/layer-2.txt", "old-layer-2", daemonOS)
			h.AssertNil(t, err)
			defer os.Remove(layer2Path)

			h.AssertNil(t, prevImage.AddLayer(layer1Path))
			h.AssertNil(t, prevImage.AddLayer(layer2Path))

			h.AssertNil(t, prevImage.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), prevImageName)
			h.AssertNil(t, err)

			prevLayer1SHA = h.StringElementAt(inspect.RootFS.Layers, -2)
			prevLayer2SHA = h.StringElementAt(inspect.RootFS.Layers, -1)
		})

		it.After(func() {
			h.AssertNil(t, h.DockerRmi(dockerClient, repoName, prevImageName))
		})

		it("reuses a layer with history", func() {
			img, err := local.NewImage(
				repoName,
				dockerClient,
				local.WithPreviousImage(prevImageName),
				local.FromBaseImage(runnableBaseImageName),
				local.WithHistory(),
			)
			h.AssertNil(t, err)

			newBaseLayerPath, err := h.CreateSingleFileLayerTar("/new-base.txt", "base-content", daemonOS)
			h.AssertNil(t, err)
			defer os.Remove(newBaseLayerPath)

			h.AssertNil(t, img.AddLayer(newBaseLayerPath))

			err = img.ReuseLayerWithHistory(prevLayer2SHA, v1.History{CreatedBy: "some-new-history"})
			h.AssertNil(t, err)

			h.AssertNil(t, img.Save())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
			h.AssertNil(t, err)

			newLayer1SHA := h.StringElementAt(inspect.RootFS.Layers, -2)
			reusedLayer2SHA := h.StringElementAt(inspect.RootFS.Layers, -1)

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

	when("#Save", func() {
		when("image is valid", func() {
			var (
				img      imgutil.Image
				origID   string
				tarPath  string
				repoName = newTestImageName()
			)

			it.Before(func() {
				oldImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, oldImage.SetLabel("mykey", "oldValue"))
				h.AssertNil(t, oldImage.Save())

				origID = h.ImageID(t, repoName)

				img, err = local.NewImage(repoName, dockerClient, local.FromBaseImage(runnableBaseImageName))
				h.AssertNil(t, err)

				tarPath, err = h.CreateSingleFileLayerTar("/new-layer.txt", "new-layer", daemonOS)
				h.AssertNil(t, err)
			})

			it.After(func() {
				h.AssertNil(t, os.Remove(tarPath))
				h.AssertNil(t, h.DockerRmi(dockerClient, repoName))
			})

			it("saved image overrides image with new ID", func() {
				err := img.SetLabel("mykey", "newValue")
				h.AssertNil(t, err)

				err = img.AddLayer(tarPath)
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				identifier, err := img.Identifier()
				h.AssertNil(t, err)

				h.AssertEq(t, origID != identifier.String(), true)

				newImageID := h.ImageID(t, repoName)
				h.AssertNotEq(t, origID, newImageID)

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), identifier.String())
				h.AssertNil(t, err)
				label := inspect.Config.Labels["mykey"]
				h.AssertEq(t, strings.TrimSpace(label), "newValue")
			})

			it("zeroes times and client specific fields", func() {
				err := img.SetLabel("mykey", "newValue")
				h.AssertNil(t, err)

				err = img.AddLayer(tarPath)
				h.AssertNil(t, err)

				h.AssertNil(t, img.Save())

				inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
				h.AssertNil(t, err)

				h.AssertEq(t, inspect.Created, imgutil.NormalizedDateTime.Format(time.RFC3339))
				h.AssertEq(t, inspect.Container, "") //nolint

				history, err := dockerClient.ImageHistory(context.TODO(), repoName)
				h.AssertNil(t, err)
				h.AssertEq(t, len(history), len(inspect.RootFS.Layers))
				for i := range inspect.RootFS.Layers {
					h.AssertEq(t, history[i].Created, imgutil.NormalizedDateTime.Unix())
				}
			})

			when("the WithCreatedAt option is used", func() {
				it("uses the value for all times and client specific fields", func() {
					expectedTime := time.Date(2022, 1, 5, 5, 5, 5, 0, time.UTC)
					img, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(runnableBaseImageName),
						local.WithCreatedAt(expectedTime),
					)
					h.AssertNil(t, err)

					err = img.SetLabel("mykey", "newValue")
					h.AssertNil(t, err)

					err = img.AddLayer(tarPath)
					h.AssertNil(t, err)

					h.AssertNil(t, img.Save())

					inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), repoName)
					h.AssertNil(t, err)

					h.AssertEq(t, inspect.Created, expectedTime.Format(time.RFC3339))
					h.AssertEq(t, inspect.Container, "") //nolint

					history, err := dockerClient.ImageHistory(context.TODO(), repoName)
					h.AssertNil(t, err)
					h.AssertEq(t, len(history), len(inspect.RootFS.Layers))
					for i := range inspect.RootFS.Layers {
						h.AssertEq(t, history[i].Created, expectedTime.Unix())
					}
				})
			})

			when("additional names are provided", func() {
				var (
					additionalRepoNames = []string{
						repoName + ":" + h.RandString(5),
						newTestImageName(),
						newTestImageName(),
					}
					successfulRepoNames = append([]string{repoName}, additionalRepoNames...)
				)

				it.After(func() {
					h.AssertNil(t, h.DockerRmi(dockerClient, additionalRepoNames...))
				})

				it("saves to multiple names", func() {
					h.AssertNil(t, img.Save(additionalRepoNames...))

					for _, n := range successfulRepoNames {
						_, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), n)
						h.AssertNil(t, err)
					}
				})

				when("a single image name fails", func() {
					it("returns results with errors for those that failed", func() {
						failingName := newTestImageName() + ":🧨"

						err := img.Save(append([]string{failingName}, additionalRepoNames...)...)
						h.AssertError(t, err, fmt.Sprintf("failed to write image to the following tags: [%s:", failingName))

						saveErr, ok := err.(imgutil.SaveError)
						h.AssertEq(t, ok, true)
						h.AssertEq(t, len(saveErr.Errors), 1)
						h.AssertEq(t, saveErr.Errors[0].ImageName, failingName)
						h.AssertError(t, saveErr.Errors[0].Cause, "invalid reference format")

						for _, n := range successfulRepoNames {
							_, _, err = dockerClient.ImageInspectWithRaw(context.TODO(), n)
							h.AssertNil(t, err)
						}
					})
				})
			})
		})

		when("invalid image content for daemon", func() {
			it("returns errors from daemon", func() {
				repoName := newTestImageName()

				invalidLayerTarFile, err := os.CreateTemp("", "daemon-error-test")
				h.AssertNil(t, err)
				defer func() { invalidLayerTarFile.Close(); os.Remove(invalidLayerTarFile.Name()) }()

				invalidLayerTarFile.Write([]byte("NOT A TAR"))
				invalidLayerPath := invalidLayerTarFile.Name()

				img, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				err = img.AddLayer(invalidLayerPath)
				h.AssertNil(t, err)

				err = img.Save()
				h.AssertError(t, err, fmt.Sprintf("failed to write image to the following tags: [%s:", repoName))
				h.AssertError(t, err, "daemon response")
			})
		})
	})

	when("#SaveFile", func() {
		var (
			img      imgutil.Image
			tarPath1 string
			tarPath2 string
			repoName = newTestImageName()
		)

		it.After(func() {
			os.Remove(tarPath1)
			os.Remove(tarPath2)
		})

		saveFileTest := func() {
			h.AssertNil(t, img.AddLayer(tarPath1))
			h.AssertNil(t, img.AddLayer(tarPath2))

			path, err := img.SaveFile()
			h.AssertNil(t, err)
			defer os.Remove(path)

			f, err := os.Open(path)
			h.AssertNil(t, err)
			defer f.Close()

			_, err = dockerClient.ImageLoad(context.TODO(), f, true)
			h.AssertNil(t, err)
			f.Close()
			defer h.DockerRmi(dockerClient, img.Name())

			inspect, _, err := dockerClient.ImageInspectWithRaw(context.TODO(), img.Name())
			h.AssertNil(t, err)

			for _, diffID := range inspect.RootFS.Layers {
				rc, err := img.GetLayer(diffID)
				h.AssertNil(t, err)
				rc.Close()
			}

			f, err = os.Open(path)
			h.AssertNil(t, err)
			defer f.Close()
			tr := tar.NewReader(f)
			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				h.AssertNil(t, err)
				h.AssertNotEq(t, strings.Contains(hdr.Name, "blank_"), true)
			}
		}

		when("no previous image or base image is configured", func() {
			it.Before(func() {
				var err error

				img, err = local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)

				tarPath1, err = h.CreateSingleFileLayerTar("/foo", "foo", daemonOS)
				h.AssertNil(t, err)

				tarPath2, err = h.CreateSingleFileLayerTar("/bar", "bar", daemonOS)
				h.AssertNil(t, err)
			})

			it("creates an archive that can be imported and has correct diffIDs", saveFileTest)
		})

		when("previous image is configured and layers are reused", func() {
			var prevImageName string

			it.Before(func() {
				var err error

				prevImageName = newTestImageName()
				prevImg, err := local.NewImage(prevImageName, dockerClient)
				h.AssertNil(t, err)

				prevImgBase, err := h.CreateSingleFileLayerTar("/root", "root", daemonOS)
				h.AssertNil(t, err)

				h.AssertNil(t, prevImg.AddLayer(prevImgBase))
				h.AssertNil(t, prevImg.Save())

				img, err = local.NewImage(repoName, dockerClient, local.WithPreviousImage(prevImg.Name()))
				h.AssertNil(t, err)

				prevImgTopLayer, err := prevImg.TopLayer()
				h.AssertNil(t, err)

				err = img.ReuseLayer(prevImgTopLayer)
				h.AssertNil(t, err)

				tarPath1, err = h.CreateSingleFileLayerTar("/foo", "foo", daemonOS)
				h.AssertNil(t, err)

				tarPath2, err = h.CreateSingleFileLayerTar("/bar", "bar", daemonOS)
				h.AssertNil(t, err)
			})

			it("creates an archive that can be imported and has correct diffIDs", saveFileTest)

			it.After(func() {
				defer h.DockerRmi(dockerClient, prevImageName)
			})
		})

		when("base image is configured", func() {
			it.Before(func() {
				var err error

				img, err = local.NewImage(repoName, dockerClient, local.FromBaseImage(runnableBaseImageName))
				h.AssertNil(t, err)

				tarPath1, err = h.CreateSingleFileLayerTar("/foo", "foo", daemonOS)
				h.AssertNil(t, err)

				tarPath2, err = h.CreateSingleFileLayerTar("/bar", "bar", daemonOS)
				h.AssertNil(t, err)
			})

			it("creates an archive that can be imported and has correct diffIDs", saveFileTest)
		})
	})

	when("#Found", func() {
		when("it exists", func() {
			var repoName = newTestImageName()

			it.Before(func() {
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)
				h.AssertNil(t, existingImage.Save())
			})

			it.After(func() {
				h.DockerRmi(dockerClient, repoName)
			})

			it("returns true, nil", func() {
				image, err := local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				h.AssertEq(t, image.Found(), true)
			})
		})

		when("it does not exist", func() {
			it("returns false, nil", func() {
				image, err := local.NewImage(newTestImageName(), dockerClient)
				h.AssertNil(t, err)

				h.AssertEq(t, image.Found(), false)
			})
		})
	})

	when("#Delete", func() {
		when("the image does not exist", func() {
			it("should not error", func() {
				img, err := local.NewImage("image-does-not-exist", dockerClient)
				h.AssertNil(t, err)

				h.AssertNil(t, img.Delete())
			})
		})

		when("the image does exist", func() {
			var (
				origImg  imgutil.Image
				origID   string
				repoName = newTestImageName()
			)

			it.Before(func() {
				existingImage, err := local.NewImage(repoName, dockerClient)
				h.AssertNil(t, err)
				h.AssertNil(t, existingImage.SetLabel("some", "label"))
				h.AssertNil(t, existingImage.Save())

				origImg, err = local.NewImage(repoName, dockerClient, local.FromBaseImage(repoName))
				h.AssertNil(t, err)

				origID = h.ImageID(t, repoName)
			})

			it("should delete the image", func() {
				h.AssertEq(t, origImg.Found(), true)

				h.AssertNil(t, origImg.Delete())

				img, err := local.NewImage(origID, dockerClient)
				h.AssertNil(t, err)

				h.AssertEq(t, img.Found(), false)
			})

			when("the image has been re-tagged", func() {
				const newTag = "different-tag"

				it.Before(func() {
					h.AssertNil(t, dockerClient.ImageTag(context.TODO(), origImg.Name(), newTag))

					_, err := dockerClient.ImageRemove(context.TODO(), origImg.Name(), image.RemoveOptions{})
					h.AssertNil(t, err)
				})

				it("should delete the image", func() {
					h.AssertEq(t, origImg.Found(), true)

					h.AssertNil(t, origImg.Delete())

					origImg, err := local.NewImage(newTag, dockerClient)
					h.AssertNil(t, err)

					h.AssertEq(t, origImg.Found(), false)
				})
			})
		})
	})
}
