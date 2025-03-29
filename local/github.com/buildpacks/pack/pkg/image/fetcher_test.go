package image_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/local"
	"github.com/buildpacks/imgutil/remote"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

var docker client.CommonAPIClient
var registryConfig *h.TestRegistryConfig

func TestFetcher(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	h.RequireDocker(t)

	registryConfig = h.RunRegistry(t)
	defer registryConfig.StopRegistry(t)

	// TODO: is there a better solution to the auth problem?
	os.Setenv("DOCKER_CONFIG", registryConfig.DockerConfigDir)

	var err error
	docker, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
	h.AssertNil(t, err)
	spec.Run(t, "Fetcher", testFetcher, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testFetcher(t *testing.T, when spec.G, it spec.S) {
	var (
		imageFetcher *image.Fetcher
		repoName     string
		repo         string
		outBuf       bytes.Buffer
		osType       string
	)

	it.Before(func() {
		repo = "some-org/" + h.RandString(10)
		repoName = registryConfig.RepoName(repo)
		imageFetcher = image.NewFetcher(logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose()), docker)

		info, err := docker.Info(context.TODO())
		h.AssertNil(t, err)
		osType = info.OSType
	})

	when("#Fetch", func() {
		when("daemon is false", func() {
			when("PullAlways", func() {
				when("there is a remote image", func() {
					when("default platform", func() {
						// default is linux/runtime.GOARCH
						it.Before(func() {
							img, err := remote.NewImage(repoName, authn.DefaultKeychain)
							h.AssertNil(t, err)

							h.AssertNil(t, img.Save())
						})

						it("returns the remote image", func() {
							_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: false, PullPolicy: image.PullAlways})
							h.AssertNil(t, err)
						})
					})

					when("platform with variant and version", func() {
						var target dist.Target

						// default is linux/runtime.GOARCH
						it.Before(func() {
							img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.WithDefaultPlatform(imgutil.Platform{
								OS:           runtime.GOOS,
								Architecture: runtime.GOARCH,
								Variant:      "v1",
								OSVersion:    "my-version",
							}))
							h.AssertNil(t, err)
							h.AssertNil(t, img.Save())
						})

						it("returns the remote image", func() {
							target = dist.Target{
								OS:          runtime.GOOS,
								Arch:        runtime.GOARCH,
								ArchVariant: "v1",
								Distributions: []dist.Distribution{
									{Name: "some-name", Version: "my-version"},
								},
							}

							img, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: false, PullPolicy: image.PullAlways, Target: &target})
							h.AssertNil(t, err)
							variant, err := img.Variant()
							h.AssertNil(t, err)
							h.AssertEq(t, variant, "v1")

							osVersion, err := img.OSVersion()
							h.AssertNil(t, err)
							h.AssertEq(t, osVersion, "my-version")
						})
					})
				})

				when("there is no remote image", func() {
					it("returns an error", func() {
						_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: false, PullPolicy: image.PullAlways})
						h.AssertError(t, err, fmt.Sprintf("image '%s' does not exist in registry", repoName))
					})
				})
			})

			when("PullIfNotPresent", func() {
				when("there is a remote image", func() {
					it.Before(func() {
						img, err := remote.NewImage(repoName, authn.DefaultKeychain)
						h.AssertNil(t, err)

						h.AssertNil(t, img.Save())
					})

					it("returns the remote image", func() {
						_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: false, PullPolicy: image.PullIfNotPresent})
						h.AssertNil(t, err)
					})
				})

				when("there is no remote image", func() {
					it("returns an error", func() {
						_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: false, PullPolicy: image.PullIfNotPresent})
						h.AssertError(t, err, fmt.Sprintf("image '%s' does not exist in registry", repoName))
					})
				})
			})
		})

		when("daemon is true", func() {
			when("PullNever", func() {
				when("there is a local image", func() {
					it.Before(func() {
						// Make sure the repoName is not a valid remote repo.
						// This is to verify that no remote check is made
						// when there's a valid local image.
						repoName = "invalidhost" + repoName

						img, err := local.NewImage(repoName, docker)
						h.AssertNil(t, err)

						h.AssertNil(t, img.Save())
					})

					it.After(func() {
						h.DockerRmi(docker, repoName)
					})

					it("returns the local image", func() {
						_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullNever})
						h.AssertNil(t, err)
					})
				})

				when("there is no local image", func() {
					it("returns an error", func() {
						_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullNever})
						h.AssertError(t, err, fmt.Sprintf("image '%s' does not exist on the daemon", repoName))
					})
				})
			})

			when("PullAlways", func() {
				when("there is a remote image", func() {
					var (
						logger *logging.LogWithWriters
						output func() string
					)

					it.Before(func() {
						// Instantiate a pull-able local image
						// as opposed to a remote image so that the image
						// is created with the OS of the docker daemon
						img, err := local.NewImage(repoName, docker)
						h.AssertNil(t, err)
						defer h.DockerRmi(docker, repoName)

						h.AssertNil(t, img.Save())

						h.AssertNil(t, h.PushImage(docker, img.Name(), registryConfig))

						var outCons *color.Console
						outCons, output = h.MockWriterAndOutput()
						logger = logging.NewLogWithWriters(outCons, outCons)
						imageFetcher = image.NewFetcher(logger, docker)
					})

					it.After(func() {
						h.DockerRmi(docker, repoName)
					})

					it("pull the image and return the local copy", func() {
						_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways})
						h.AssertNil(t, err)
						h.AssertNotEq(t, output(), "")
					})

					it("doesn't log anything in quiet mode", func() {
						logger.WantQuiet(true)
						_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways})
						h.AssertNil(t, err)
						h.AssertEq(t, output(), "")
					})
				})

				when("there is no remote image", func() {
					when("there is a local image", func() {
						it.Before(func() {
							img, err := local.NewImage(repoName, docker)
							h.AssertNil(t, err)

							h.AssertNil(t, img.Save())
						})

						it.After(func() {
							h.DockerRmi(docker, repoName)
						})

						it("returns the local image", func() {
							_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways})
							h.AssertNil(t, err)
						})
					})

					when("there is no local image", func() {
						it("returns an error", func() {
							_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways})
							h.AssertError(t, err, fmt.Sprintf("image '%s' does not exist on the daemon", repoName))
						})
					})
				})

				when("image platform is specified", func() {
					it("passes the platform argument to the daemon", func() {
						_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways, Target: &dist.Target{OS: "some-unsupported-platform"}})
						h.AssertError(t, err, "unknown operating system or architecture")
					})

					when("remote platform does not match", func() {
						it.Before(func() {
							img, err := remote.NewImage(repoName, authn.DefaultKeychain, remote.WithDefaultPlatform(imgutil.Platform{OS: osType, Architecture: ""}))
							h.AssertNil(t, err)
							h.AssertNil(t, img.Save())
						})

						it("retries without setting platform", func() {
							_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullAlways, Target: &dist.Target{OS: osType, Arch: runtime.GOARCH}})
							h.AssertNil(t, err)
						})
					})
				})
			})

			when("PullIfNotPresent", func() {
				when("there is a remote image", func() {
					var (
						label          = "label"
						remoteImgLabel string
					)

					it.Before(func() {
						// Instantiate a pull-able local image
						// as opposed to a remote image so that the image
						// is created with the OS of the docker daemon
						remoteImg, err := local.NewImage(repoName, docker)
						h.AssertNil(t, err)
						defer h.DockerRmi(docker, repoName)

						h.AssertNil(t, remoteImg.SetLabel(label, "1"))
						h.AssertNil(t, remoteImg.Save())

						h.AssertNil(t, h.PushImage(docker, remoteImg.Name(), registryConfig))

						remoteImgLabel, err = remoteImg.Label(label)
						h.AssertNil(t, err)
					})

					it.After(func() {
						h.DockerRmi(docker, repoName)
					})

					when("there is a local image", func() {
						var localImgLabel string

						it.Before(func() {
							localImg, err := local.NewImage(repoName, docker)
							h.AssertNil(t, localImg.SetLabel(label, "2"))
							h.AssertNil(t, err)

							h.AssertNil(t, localImg.Save())

							localImgLabel, err = localImg.Label(label)
							h.AssertNil(t, err)
						})

						it.After(func() {
							h.DockerRmi(docker, repoName)
						})

						it("returns the local image", func() {
							fetchedImg, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullIfNotPresent})
							h.AssertNil(t, err)
							h.AssertNotContains(t, outBuf.String(), "Pulling image")

							fetchedImgLabel, err := fetchedImg.Label(label)
							h.AssertNil(t, err)

							h.AssertEq(t, fetchedImgLabel, localImgLabel)
							h.AssertNotEq(t, fetchedImgLabel, remoteImgLabel)
						})
					})

					when("there is no local image", func() {
						it("returns the remote image", func() {
							fetchedImg, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullIfNotPresent})
							h.AssertNil(t, err)

							fetchedImgLabel, err := fetchedImg.Label(label)
							h.AssertNil(t, err)
							h.AssertEq(t, fetchedImgLabel, remoteImgLabel)
						})
					})
				})

				when("there is no remote image", func() {
					when("there is a local image", func() {
						it.Before(func() {
							img, err := local.NewImage(repoName, docker)
							h.AssertNil(t, err)

							h.AssertNil(t, img.Save())
						})

						it.After(func() {
							h.DockerRmi(docker, repoName)
						})

						it("returns the local image", func() {
							_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullIfNotPresent})
							h.AssertNil(t, err)
						})
					})

					when("there is no local image", func() {
						it("returns an error", func() {
							_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullIfNotPresent})
							h.AssertError(t, err, fmt.Sprintf("image '%s' does not exist on the daemon", repoName))
						})
					})
				})

				when("image platform is specified", func() {
					it("passes the platform argument to the daemon", func() {
						_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{Daemon: true, PullPolicy: image.PullIfNotPresent, Target: &dist.Target{OS: "some-unsupported-platform"}})
						h.AssertError(t, err, "unknown operating system or architecture")
					})
				})
			})
		})

		when("layout option is provided", func() {
			var (
				layoutOption image.LayoutOption
				imagePath    string
				tmpDir       string
				err          error
			)

			it.Before(func() {
				// set up local layout repo
				tmpDir, err = os.MkdirTemp("", "pack.fetcher.test")
				h.AssertNil(t, err)

				// dummy layer to validate sparse behavior
				tarDir := filepath.Join(tmpDir, "layer")
				err = os.MkdirAll(tarDir, os.ModePerm)
				h.AssertNil(t, err)
				layerPath := h.CreateTAR(t, tarDir, ".", -1)

				// set up the remote image to be used
				img, err := remote.NewImage(repoName, authn.DefaultKeychain)
				img.AddLayer(layerPath)
				h.AssertNil(t, err)
				h.AssertNil(t, img.Save())

				// set up layout options for the tests
				imagePath = filepath.Join(tmpDir, repo)
				layoutOption = image.LayoutOption{
					Path:   imagePath,
					Sparse: false,
				}
			})

			it.After(func() {
				err = os.RemoveAll(tmpDir)
				h.AssertNil(t, err)
			})

			when("sparse is false", func() {
				it("returns and layout image on disk", func() {
					_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{LayoutOption: layoutOption})
					h.AssertNil(t, err)

					// all layers were written
					h.AssertBlobsLen(t, imagePath, 3)
				})
			})

			when("sparse is true", func() {
				it("returns and layout image on disk", func() {
					layoutOption.Sparse = true
					_, err := imageFetcher.Fetch(context.TODO(), repoName, image.FetchOptions{LayoutOption: layoutOption})
					h.AssertNil(t, err)

					// only manifest and config was written
					h.AssertBlobsLen(t, imagePath, 2)
				})
			})
		})
	})

	when("#CheckReadAccess", func() {
		var daemon bool

		when("Daemon is true", func() {
			it.Before(func() {
				daemon = true
			})

			when("an error is thrown by the daemon", func() {
				it.Before(func() {
					mockController := gomock.NewController(t)
					mockDockerClient := testmocks.NewMockCommonAPIClient(mockController)
					mockDockerClient.EXPECT().ServerVersion(gomock.Any()).Return(types.Version{}, errors.New("something wrong happened"))
					imageFetcher = image.NewFetcher(logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose()), mockDockerClient)
				})
				when("PullNever", func() {
					it("read access must be false", func() {
						h.AssertFalse(t, imageFetcher.CheckReadAccess("pack.test/dummy", image.FetchOptions{Daemon: daemon, PullPolicy: image.PullNever}))
						h.AssertContains(t, outBuf.String(), "failed reading image 'pack.test/dummy' from the daemon")
					})
				})

				when("PullIfNotPresent", func() {
					it("read access must be false", func() {
						h.AssertFalse(t, imageFetcher.CheckReadAccess("pack.test/dummy", image.FetchOptions{Daemon: daemon, PullPolicy: image.PullIfNotPresent}))
						h.AssertContains(t, outBuf.String(), "failed reading image 'pack.test/dummy' from the daemon")
					})
				})
			})

			when("image exists only in the daemon", func() {
				it.Before(func() {
					img, err := local.NewImage("pack.test/dummy", docker)
					h.AssertNil(t, err)
					h.AssertNil(t, img.Save())
				})
				when("PullAlways", func() {
					it("read access must be false", func() {
						h.AssertFalse(t, imageFetcher.CheckReadAccess("pack.test/dummy", image.FetchOptions{Daemon: daemon, PullPolicy: image.PullAlways}))
					})
				})

				when("PullNever", func() {
					it("read access must be true", func() {
						h.AssertTrue(t, imageFetcher.CheckReadAccess("pack.test/dummy", image.FetchOptions{Daemon: daemon, PullPolicy: image.PullNever}))
					})
				})

				when("PullIfNotPresent", func() {
					it("read access must be true", func() {
						h.AssertTrue(t, imageFetcher.CheckReadAccess("pack.test/dummy", image.FetchOptions{Daemon: daemon, PullPolicy: image.PullIfNotPresent}))
					})
				})
			})

			when("image doesn't exist in the daemon but in remote", func() {
				it.Before(func() {
					img, err := remote.NewImage(repoName, authn.DefaultKeychain)
					h.AssertNil(t, err)
					h.AssertNil(t, img.Save())
				})
				when("PullAlways", func() {
					it("read access must be true", func() {
						h.AssertTrue(t, imageFetcher.CheckReadAccess(repoName, image.FetchOptions{Daemon: daemon, PullPolicy: image.PullAlways}))
					})
				})

				when("PullNever", func() {
					it("read access must be false", func() {
						h.AssertFalse(t, imageFetcher.CheckReadAccess(repoName, image.FetchOptions{Daemon: daemon, PullPolicy: image.PullNever}))
					})
				})

				when("PullIfNotPresent", func() {
					it("read access must be true", func() {
						h.AssertTrue(t, imageFetcher.CheckReadAccess(repoName, image.FetchOptions{Daemon: daemon, PullPolicy: image.PullIfNotPresent}))
					})
				})
			})
		})

		when("Daemon is false", func() {
			it.Before(func() {
				daemon = false
			})

			when("remote image doesn't exists", func() {
				it("fails when checking dummy image", func() {
					h.AssertFalse(t, imageFetcher.CheckReadAccess("pack.test/dummy", image.FetchOptions{Daemon: daemon}))
					h.AssertContains(t, outBuf.String(), "CheckReadAccess failed for the run image pack.test/dummy")
				})
			})

			when("remote image exists", func() {
				it.Before(func() {
					img, err := remote.NewImage(repoName, authn.DefaultKeychain)
					h.AssertNil(t, err)
					h.AssertNil(t, img.Save())
				})

				it("read access is valid", func() {
					h.AssertTrue(t, imageFetcher.CheckReadAccess(repoName, image.FetchOptions{Daemon: daemon}))
					h.AssertContains(t, outBuf.String(), fmt.Sprintf("CheckReadAccess succeeded for the run image %s", repoName))
				})
			})
		})
	})
}
