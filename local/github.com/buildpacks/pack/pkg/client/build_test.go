package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/fakes"
	"github.com/buildpacks/imgutil/local"
	"github.com/buildpacks/imgutil/remote"
	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/platform/files"
	dockerclient "github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/heroku/color"
	"github.com/onsi/gomega/ghttp"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/builder"
	cfg "github.com/buildpacks/pack/internal/config"
	ifakes "github.com/buildpacks/pack/internal/fakes"
	rg "github.com/buildpacks/pack/internal/registry"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	projectTypes "github.com/buildpacks/pack/pkg/project/types"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestBuild(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "build", testBuild, spec.Report(report.Terminal{}))
}

func testBuild(t *testing.T, when spec.G, it spec.S) {
	var (
		subject                      *Client
		fakeImageFetcher             *ifakes.FakeImageFetcher
		fakeLifecycle                *ifakes.FakeLifecycle
		defaultBuilderStackID        = "some.stack.id"
		defaultWindowsBuilderStackID = "some.windows.stack.id"
		defaultBuilderImage          *fakes.Image
		defaultWindowsBuilderImage   *fakes.Image
		defaultBuilderName           = "example.com/default/builder:tag"
		defaultWindowsBuilderName    = "example.com/windows-default/builder:tag"
		defaultRunImageName          = "default/run"
		defaultWindowsRunImageName   = "default/win-run"
		fakeDefaultRunImage          *fakes.Image
		fakeDefaultWindowsRunImage   *fakes.Image
		fakeMirror1                  *fakes.Image
		fakeMirror2                  *fakes.Image
		tmpDir                       string
		outBuf                       bytes.Buffer
		logger                       *logging.LogWithWriters
		fakeLifecycleImage           *fakes.Image

		withExtensionsLabel bool
	)

	it.Before(func() {
		var err error

		fakeImageFetcher = ifakes.NewFakeImageFetcher()
		fakeLifecycle = &ifakes.FakeLifecycle{}

		tmpDir, err = os.MkdirTemp("", "build-test")
		h.AssertNil(t, err)

		defaultBuilderImage = newFakeBuilderImage(t, tmpDir, defaultBuilderName, defaultBuilderStackID, defaultRunImageName, builder.DefaultLifecycleVersion, newLinuxImage)
		h.AssertNil(t, defaultBuilderImage.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "build:mixinB", "mixinX", "build:mixinY"]`))
		fakeImageFetcher.LocalImages[defaultBuilderImage.Name()] = defaultBuilderImage
		if withExtensionsLabel {
			h.AssertNil(t, defaultBuilderImage.SetLabel("io.buildpacks.buildpack.order-extensions", `[{"group":[{"id":"some-extension-id","version":"some-extension-version"}]}]`))
		}

		defaultWindowsBuilderImage = newFakeBuilderImage(t, tmpDir, defaultWindowsBuilderName, defaultWindowsBuilderStackID, defaultWindowsRunImageName, builder.DefaultLifecycleVersion, newWindowsImage)
		h.AssertNil(t, defaultWindowsBuilderImage.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "build:mixinB", "mixinX", "build:mixinY"]`))
		fakeImageFetcher.LocalImages[defaultWindowsBuilderImage.Name()] = defaultWindowsBuilderImage
		if withExtensionsLabel {
			h.AssertNil(t, defaultWindowsBuilderImage.SetLabel("io.buildpacks.buildpack.order-extensions", `[{"group":[{"id":"some-extension-id","version":"some-extension-version"}]}]`))
		}

		fakeDefaultWindowsRunImage = newWindowsImage("default/win-run", "", nil)
		h.AssertNil(t, fakeDefaultWindowsRunImage.SetLabel("io.buildpacks.stack.id", defaultWindowsBuilderStackID))
		h.AssertNil(t, fakeDefaultWindowsRunImage.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "run:mixinC", "mixinX", "run:mixinZ"]`))
		fakeImageFetcher.LocalImages[fakeDefaultWindowsRunImage.Name()] = fakeDefaultWindowsRunImage

		fakeDefaultRunImage = newLinuxImage("default/run", "", nil)
		h.AssertNil(t, fakeDefaultRunImage.SetLabel("io.buildpacks.stack.id", defaultBuilderStackID))
		h.AssertNil(t, fakeDefaultRunImage.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "run:mixinC", "mixinX", "run:mixinZ"]`))
		fakeImageFetcher.LocalImages[fakeDefaultRunImage.Name()] = fakeDefaultRunImage

		fakeMirror1 = newLinuxImage("registry1.example.com/run/mirror", "", nil)
		h.AssertNil(t, fakeMirror1.SetLabel("io.buildpacks.stack.id", defaultBuilderStackID))
		h.AssertNil(t, fakeMirror1.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "mixinX", "run:mixinZ"]`))
		fakeImageFetcher.LocalImages[fakeMirror1.Name()] = fakeMirror1

		fakeMirror2 = newLinuxImage("registry2.example.com/run/mirror", "", nil)
		h.AssertNil(t, fakeMirror2.SetLabel("io.buildpacks.stack.id", defaultBuilderStackID))
		h.AssertNil(t, fakeMirror2.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "mixinX", "run:mixinZ"]`))
		fakeImageFetcher.LocalImages[fakeMirror2.Name()] = fakeMirror2

		fakeLifecycleImage = newLinuxImage(fmt.Sprintf("%s:%s", cfg.DefaultLifecycleImageRepo, builder.DefaultLifecycleVersion), "", nil)
		fakeImageFetcher.LocalImages[fakeLifecycleImage.Name()] = fakeLifecycleImage

		docker, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithVersion("1.38"))
		h.AssertNil(t, err)

		logger = logging.NewLogWithWriters(&outBuf, &outBuf)

		dlCacheDir, err := os.MkdirTemp(tmpDir, "dl-cache")
		h.AssertNil(t, err)

		blobDownloader := blob.NewDownloader(logger, dlCacheDir)
		buildpackDownloader := buildpack.NewDownloader(logger, fakeImageFetcher, blobDownloader, &registryResolver{logger: logger})
		subject = &Client{
			logger:              logger,
			imageFetcher:        fakeImageFetcher,
			downloader:          blobDownloader,
			lifecycleExecutor:   fakeLifecycle,
			docker:              docker,
			buildpackDownloader: buildpackDownloader,
		}
	})

	it.After(func() {
		h.AssertNilE(t, defaultBuilderImage.Cleanup())
		h.AssertNilE(t, fakeDefaultRunImage.Cleanup())
		h.AssertNilE(t, fakeMirror1.Cleanup())
		h.AssertNilE(t, fakeMirror2.Cleanup())
		os.RemoveAll(tmpDir)
		h.AssertNilE(t, fakeLifecycleImage.Cleanup())
	})

	when("#Build", func() {
		when("ephemeral builder is not needed", func() {
			it("does not create one", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Builder: defaultBuilderName,
					Image:   "example.com/some/repo:tag",
				}))
				h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderName)
				bldr := fakeLifecycle.Opts.Builder.(*builder.Builder)
				h.AssertNotNil(t, bldr.Save(logger, builder.CreatorMetadata{})) // it shouldn't be possible to save this builder, as that would overwrite the original builder
			})
		})

		when("Workspace option", func() {
			it("uses the specified dir", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Workspace: "app",
					Builder:   defaultBuilderName,
					Image:     "example.com/some/repo:tag",
				}))
				h.AssertEq(t, fakeLifecycle.Opts.Workspace, "app")
			})
		})

		when("Image option", func() {
			it("is required", func() {
				h.AssertError(t, subject.Build(context.TODO(), BuildOptions{
					Image:   "",
					Builder: defaultBuilderName,
				}),
					"invalid image name ''",
				)
			})

			it("must be a valid image reference", func() {
				h.AssertError(t, subject.Build(context.TODO(), BuildOptions{
					Image:   "not@valid",
					Builder: defaultBuilderName,
				}),
					"invalid image name 'not@valid'",
				)
			})

			it("must be a valid tag reference", func() {
				h.AssertError(t, subject.Build(context.TODO(), BuildOptions{
					Image:   "registry.com/my/image@sha256:954e1f01e80ce09d0887ff6ea10b13a812cb01932a0781d6b0cc23f743a874fd",
					Builder: defaultBuilderName,
				}),
					"invalid image name 'registry.com/my/image@sha256:954e1f01e80ce09d0887ff6ea10b13a812cb01932a0781d6b0cc23f743a874fd'",
				)
			})

			it("lifecycle receives resolved reference", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Builder: defaultBuilderName,
					Image:   "example.com/some/repo:tag",
				}))
				h.AssertEq(t, fakeLifecycle.Opts.Image.Context().RegistryStr(), "example.com")
				h.AssertEq(t, fakeLifecycle.Opts.Image.Context().RepositoryStr(), "some/repo")
				h.AssertEq(t, fakeLifecycle.Opts.Image.Identifier(), "tag")
			})
		})

		when("Quiet mode", func() {
			var builtImage *fakes.Image

			it.After(func() {
				logger.WantQuiet(false)
			})

			when("publish", func() {
				var remoteRunImage, builderWithoutLifecycleImageOrCreator *fakes.Image

				it.Before(func() {
					remoteRunImage = fakes.NewImage("default/run", "", nil)
					h.AssertNil(t, remoteRunImage.SetLabel("io.buildpacks.stack.id", defaultBuilderStackID))
					h.AssertNil(t, remoteRunImage.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "mixinX", "run:mixinZ"]`))
					fakeImageFetcher.RemoteImages[remoteRunImage.Name()] = remoteRunImage

					builderWithoutLifecycleImageOrCreator = newFakeBuilderImage(
						t,
						tmpDir,
						"example.com/supportscreator/builder:tag",
						"some.stack.id",
						defaultRunImageName,
						"0.3.0",
						newLinuxImage,
					)
					h.AssertNil(t, builderWithoutLifecycleImageOrCreator.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "build:mixinB", "mixinX", "build:mixinY"]`))
					fakeImageFetcher.LocalImages[builderWithoutLifecycleImageOrCreator.Name()] = builderWithoutLifecycleImageOrCreator

					digest, err := name.NewDigest("example.io/some/app@sha256:363c754893f0efe22480b4359a5956cf3bd3ce22742fc576973c61348308c2e4", name.WeakValidation)
					h.AssertNil(t, err)
					builtImage = fakes.NewImage("example.io/some/app:latest", "", remote.DigestIdentifier{Digest: digest})
					fakeImageFetcher.RemoteImages[builtImage.Name()] = builtImage
				})

				it.After(func() {
					h.AssertNilE(t, remoteRunImage.Cleanup())
					h.AssertNilE(t, builderWithoutLifecycleImageOrCreator.Cleanup())
					h.AssertNilE(t, builtImage.Cleanup())
				})

				it("only prints app name and sha", func() {
					logger.WantQuiet(true)

					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "example.io/some/app",
						Builder: defaultBuilderName,
						AppPath: filepath.Join("testdata", "some-app"),
						Publish: true,
					}))

					h.AssertEq(t, strings.TrimSpace(outBuf.String()), "example.io/some/app@sha256:363c754893f0efe22480b4359a5956cf3bd3ce22742fc576973c61348308c2e4")
				})
			})

			when("local", func() {
				it.Before(func() {
					builtImage = fakes.NewImage("index.docker.io/some/app:latest", "", local.IDIdentifier{
						ImageID: "363c754893f0efe22480b4359a5956cf3bd3ce22742fc576973c61348308c2e4",
					})
					fakeImageFetcher.LocalImages[builtImage.Name()] = builtImage
				})

				it.After(func() {
					h.AssertNilE(t, builtImage.Cleanup())
				})

				it("only prints app name and sha", func() {
					logger.WantQuiet(true)

					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						AppPath: filepath.Join("testdata", "some-app"),
					}))

					h.AssertEq(t, strings.TrimSpace(outBuf.String()), "some/app@sha256:363c754893f0efe22480b4359a5956cf3bd3ce22742fc576973c61348308c2e4")
				})
			})
		})

		when("AppDir option", func() {
			it("defaults to the current working directory", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:   "some/app",
					Builder: defaultBuilderName,
				}))

				wd, err := os.Getwd()
				h.AssertNil(t, err)
				resolvedWd, err := filepath.EvalSymlinks(wd)
				h.AssertNil(t, err)
				h.AssertEq(t, fakeLifecycle.Opts.AppPath, resolvedWd)
			})
			for fileDesc, appPath := range map[string]string{
				"zip": filepath.Join("testdata", "zip-file.zip"),
				"jar": filepath.Join("testdata", "jar-file.jar"),
			} {
				fileDesc := fileDesc
				appPath := appPath

				it(fmt.Sprintf("supports %s files", fileDesc), func() {
					err := subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						AppPath: appPath,
					})
					h.AssertNil(t, err)
				})
			}

			for fileDesc, testData := range map[string][]string{
				"non-existent": {"not/exist/path", "does not exist"},
				"empty":        {filepath.Join("testdata", "empty-file"), "app path must be a directory or zip"},
				"non-zip":      {filepath.Join("testdata", "non-zip-file"), "app path must be a directory or zip"},
			} {
				fileDesc := fileDesc
				appPath := testData[0]
				errMessage := testData[0]

				it(fmt.Sprintf("does NOT support %s files", fileDesc), func() {
					err := subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						AppPath: appPath,
					})

					h.AssertError(t, err, errMessage)
				})
			}

			it("resolves the absolute path", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:   "some/app",
					Builder: defaultBuilderName,
					AppPath: filepath.Join("testdata", "some-app"),
				}))
				absPath, err := filepath.Abs(filepath.Join("testdata", "some-app"))
				h.AssertNil(t, err)
				h.AssertEq(t, fakeLifecycle.Opts.AppPath, absPath)
			})

			when("appDir is a symlink", func() {
				var (
					appDirName     = "some-app"
					absoluteAppDir string
					tmpDir         string
					err            error
				)

				it.Before(func() {
					tmpDir, err = os.MkdirTemp("", "build-symlink-test")
					h.AssertNil(t, err)

					appDirPath := filepath.Join(tmpDir, appDirName)
					h.AssertNil(t, os.MkdirAll(filepath.Join(tmpDir, appDirName), 0666))

					absoluteAppDir, err = filepath.Abs(appDirPath)
					h.AssertNil(t, err)

					absoluteAppDir, err = filepath.EvalSymlinks(appDirPath)
					h.AssertNil(t, err)
				})

				it.After(func() {
					os.RemoveAll(tmpDir)
				})

				it("resolves relative symbolic links", func() {
					relLink := filepath.Join(tmpDir, "some-app.link")
					h.AssertNil(t, os.Symlink(filepath.Join(".", appDirName), relLink))

					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						AppPath: relLink,
					}))

					h.AssertEq(t, fakeLifecycle.Opts.AppPath, absoluteAppDir)
				})

				it("resolves absolute symbolic links", func() {
					relLink := filepath.Join(tmpDir, "some-app.link")
					h.AssertNil(t, os.Symlink(absoluteAppDir, relLink))

					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						AppPath: relLink,
					}))

					h.AssertEq(t, fakeLifecycle.Opts.AppPath, absoluteAppDir)
				})

				it("resolves symbolic links recursively", func() {
					linkRef1 := absoluteAppDir
					absoluteLink1 := filepath.Join(tmpDir, "some-app-abs-1.link")

					linkRef2 := "some-app-abs-1.link"
					symbolicLink := filepath.Join(tmpDir, "some-app-rel-2.link")

					h.AssertNil(t, os.Symlink(linkRef1, absoluteLink1))
					h.AssertNil(t, os.Symlink(linkRef2, symbolicLink))

					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						AppPath: symbolicLink,
					}))

					h.AssertEq(t, fakeLifecycle.Opts.AppPath, absoluteAppDir)
				})
			})
		})

		when("Builder option", func() {
			it("builder is required", func() {
				h.AssertError(t, subject.Build(context.TODO(), BuildOptions{
					Image: "some/app",
				}),
					"invalid builder ''",
				)
			})

			when("the builder name is provided", func() {
				var (
					customBuilderImage *fakes.Image
					fakeRunImage       *fakes.Image
				)

				it.Before(func() {
					customBuilderImage = ifakes.NewFakeBuilderImage(t,
						tmpDir,
						defaultBuilderName,
						"some.stack.id",
						"1234",
						"5678",
						builder.Metadata{
							Stack: builder.StackMetadata{
								RunImage: builder.RunImageMetadata{
									Image: "some/run",
								},
							},
							Lifecycle: builder.LifecycleMetadata{
								LifecycleInfo: builder.LifecycleInfo{
									Version: &builder.Version{
										Version: *semver.MustParse(builder.DefaultLifecycleVersion),
									},
								},
								APIs: builder.LifecycleAPIs{
									Buildpack: builder.APIVersions{
										Supported: builder.APISet{api.MustParse("0.2"), api.MustParse("0.3"), api.MustParse("0.4")},
									},
									Platform: builder.APIVersions{
										Supported: builder.APISet{api.MustParse("0.3"), api.MustParse("0.4")},
									},
								},
							},
						},
						nil,
						nil,
						nil,
						nil,
						newLinuxImage,
					)

					fakeImageFetcher.LocalImages[customBuilderImage.Name()] = customBuilderImage

					fakeRunImage = fakes.NewImage("some/run", "", nil)
					h.AssertNil(t, fakeRunImage.SetLabel("io.buildpacks.stack.id", "some.stack.id"))
					fakeImageFetcher.LocalImages[fakeRunImage.Name()] = fakeRunImage
				})

				it.After(func() {
					h.AssertNilE(t, customBuilderImage.Cleanup())
					h.AssertNilE(t, fakeRunImage.Cleanup())
				})

				it("it uses the provided builder", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
					}))
					h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), customBuilderImage.Name())
				})
			})
		})

		when("RunImage option", func() {
			var (
				fakeRunImage *fakes.Image
			)

			it.Before(func() {
				fakeRunImage = fakes.NewImage("custom/run", "", nil)
				h.AssertNil(t, fakeRunImage.SetLabel("io.buildpacks.stack.id", defaultBuilderStackID))
				h.AssertNil(t, fakeRunImage.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "mixinX", "run:mixinZ"]`))
				fakeImageFetcher.LocalImages[fakeRunImage.Name()] = fakeRunImage
			})

			it.After(func() {
				h.AssertNilE(t, fakeRunImage.Cleanup())
			})

			when("run image stack matches the builder stack", func() {
				it("uses the provided image", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:    "some/app",
						Builder:  defaultBuilderName,
						RunImage: "custom/run",
					}))
					h.AssertEq(t, fakeLifecycle.Opts.RunImage, "custom/run")
				})
			})

			when("run image stack does not match the builder stack", func() {
				it.Before(func() {
					h.AssertNil(t, fakeRunImage.SetLabel("io.buildpacks.stack.id", "other.stack"))
				})

				it("warning", func() {
					err := subject.Build(context.TODO(), BuildOptions{
						Image:    "some/app",
						Builder:  defaultBuilderName,
						RunImage: "custom/run",
					})
					h.AssertNil(t, err)
					h.AssertContains(t, outBuf.String(), "Warning: deprecated usage of stack")
				})
			})

			when("run image is not supplied", func() {
				when("there are no locally configured mirrors", func() {
					when("Publish is true", func() {
						it("chooses the run image mirror matching the local image", func() {
							fakeImageFetcher.RemoteImages[fakeDefaultRunImage.Name()] = fakeDefaultRunImage

							h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:   "some/app",
								Builder: defaultBuilderName,
								Publish: true,
							}))
							h.AssertEq(t, fakeLifecycle.Opts.RunImage, "default/run")
						})

						for _, registry := range []string{"registry1.example.com", "registry2.example.com"} {
							testRegistry := registry
							it("chooses the run image mirror matching the built image", func() {
								runImg := testRegistry + "/run/mirror"
								fakeImageFetcher.RemoteImages[runImg] = fakeDefaultRunImage
								h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
									Image:   testRegistry + "/some/app",
									Builder: defaultBuilderName,
									Publish: true,
								}))
								h.AssertEq(t, fakeLifecycle.Opts.RunImage, runImg)
							})
						}
					})

					when("Publish is false", func() {
						for _, img := range []string{"some/app",
							"registry1.example.com/some/app",
							"registry2.example.com/some/app"} {
							testImg := img
							it("chooses a mirror on the builder registry", func() {
								h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
									Image:   testImg,
									Builder: defaultBuilderName,
								}))
								h.AssertEq(t, fakeLifecycle.Opts.RunImage, "default/run")
							})
						}
					})
				})

				when("there are locally configured mirrors", func() {
					var (
						fakeLocalMirror  *fakes.Image
						fakeLocalMirror1 *fakes.Image
						mirrors          = map[string][]string{
							"default/run": {"local/mirror", "registry1.example.com/local/mirror"},
						}
					)

					it.Before(func() {
						fakeLocalMirror = fakes.NewImage("local/mirror", "", nil)
						h.AssertNil(t, fakeLocalMirror.SetLabel("io.buildpacks.stack.id", defaultBuilderStackID))
						h.AssertNil(t, fakeLocalMirror.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "mixinX", "run:mixinZ"]`))

						fakeImageFetcher.LocalImages[fakeLocalMirror.Name()] = fakeLocalMirror

						fakeLocalMirror1 = fakes.NewImage("registry1.example.com/local/mirror", "", nil)
						h.AssertNil(t, fakeLocalMirror1.SetLabel("io.buildpacks.stack.id", defaultBuilderStackID))
						h.AssertNil(t, fakeLocalMirror1.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "mixinX", "run:mixinZ"]`))

						fakeImageFetcher.LocalImages[fakeLocalMirror1.Name()] = fakeLocalMirror1
					})

					it.After(func() {
						h.AssertNilE(t, fakeLocalMirror.Cleanup())
						h.AssertNilE(t, fakeLocalMirror1.Cleanup())
					})

					when("Publish is true", func() {
						for _, registry := range []string{"", "registry1.example.com"} {
							testRegistry := registry
							it("prefers user provided mirrors for registry "+testRegistry, func() {
								if testRegistry != "" {
									testRegistry += "/"
								}
								runImg := testRegistry + "local/mirror"
								fakeImageFetcher.RemoteImages[runImg] = fakeDefaultRunImage

								h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
									Image:             testRegistry + "some/app",
									Builder:           defaultBuilderName,
									AdditionalMirrors: mirrors,
									Publish:           true,
								}))
								h.AssertEq(t, fakeLifecycle.Opts.RunImage, runImg)
							})
						}
					})

					when("Publish is false", func() {
						for _, registry := range []string{"", "registry1.example.com", "registry2.example.com"} {
							testRegistry := registry
							it("prefers user provided mirrors", func() {
								h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
									Image:             testRegistry + "some/app",
									Builder:           defaultBuilderName,
									AdditionalMirrors: mirrors,
								}))
								h.AssertEq(t, fakeLifecycle.Opts.RunImage, "local/mirror")
							})
						}
					})
				})
			})
		})

		when("ClearCache option", func() {
			it("passes it through to lifecycle", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:      "some/app",
					Builder:    defaultBuilderName,
					ClearCache: true,
				}))
				h.AssertEq(t, fakeLifecycle.Opts.ClearCache, true)
			})

			it("defaults to false", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:   "some/app",
					Builder: defaultBuilderName,
				}))
				h.AssertEq(t, fakeLifecycle.Opts.ClearCache, false)
			})
		})

		when("ImageCache option", func() {
			it("passes it through to lifecycle", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:      "some/app",
					Builder:    defaultBuilderName,
					CacheImage: "some-cache-image",
				}))
				h.AssertEq(t, fakeLifecycle.Opts.CacheImage, "some-cache-image")
			})

			it("defaults to false", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:   "some/app",
					Builder: defaultBuilderName,
				}))
				h.AssertEq(t, fakeLifecycle.Opts.CacheImage, "")
			})
		})

		when("Buildpacks option", func() {
			assertOrderEquals := func(content string) {
				t.Helper()

				orderLayer, err := defaultBuilderImage.FindLayerWithPath("/cnb/order.toml")
				h.AssertNil(t, err)
				h.AssertOnTarEntry(t, orderLayer, "/cnb/order.toml", h.ContentEquals(content))
			}

			it("builder order is overwritten", func() {
				additionalBP := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
					WithAPI: api.MustParse("0.3"),
					WithInfo: dist.ModuleInfo{
						ID:      "buildpack.add.1.id",
						Version: "buildpack.add.1.version",
					},
					WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
					WithOrder:  nil,
				})

				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:      "some/app",
					Builder:    defaultBuilderName,
					ClearCache: true,
					Buildpacks: []string{additionalBP},
				}))
				h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())

				assertOrderEquals(`[[order]]

  [[order.group]]
    id = "buildpack.add.1.id"
    version = "buildpack.add.1.version"
`)
			})

			when("id - no version is provided", func() {
				it("resolves version", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{"buildpack.1.id"},
					}))
					h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())

					assertOrderEquals(`[[order]]

  [[order.group]]
    id = "buildpack.1.id"
    version = "buildpack.1.version"
`)
				})
			})

			when("from=builder:id@version", func() {
				it("builder order is prepended", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{
							"from=builder:buildpack.1.id@buildpack.1.version",
						},
					}))
					h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())

					assertOrderEquals(`[[order]]

  [[order.group]]
    id = "buildpack.1.id"
    version = "buildpack.1.version"
`)
				})
			})

			when("from=builder is set first", func() {
				it("builder order is prepended", func() {
					additionalBP1 := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack.add.1.id",
							Version: "buildpack.add.1.version",
						},
						WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
						WithOrder:  nil,
					})

					additionalBP2 := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack.add.2.id",
							Version: "buildpack.add.2.version",
						},
						WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
						WithOrder:  nil,
					})

					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{
							"from=builder",
							additionalBP1,
							additionalBP2,
						},
					}))

					assertOrderEquals(`[[order]]

  [[order.group]]
    id = "buildpack.1.id"
    version = "buildpack.1.version"

  [[order.group]]
    id = "buildpack.add.1.id"
    version = "buildpack.add.1.version"

  [[order.group]]
    id = "buildpack.add.2.id"
    version = "buildpack.add.2.version"

[[order]]

  [[order.group]]
    id = "buildpack.2.id"
    version = "buildpack.2.version"

  [[order.group]]
    id = "buildpack.add.1.id"
    version = "buildpack.add.1.version"

  [[order.group]]
    id = "buildpack.add.2.id"
    version = "buildpack.add.2.version"
`)
				})
			})

			when("from=builder is set in middle", func() {
				it("builder order is appended", func() {
					additionalBP1 := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack.add.1.id",
							Version: "buildpack.add.1.version",
						},
						WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
						WithOrder:  nil,
					})

					additionalBP2 := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack.add.2.id",
							Version: "buildpack.add.2.version",
						},
						WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
						WithOrder:  nil,
					})

					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{
							additionalBP1,
							"from=builder",
							additionalBP2,
						},
					}))
					h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())

					assertOrderEquals(`[[order]]

  [[order.group]]
    id = "buildpack.add.1.id"
    version = "buildpack.add.1.version"

  [[order.group]]
    id = "buildpack.1.id"
    version = "buildpack.1.version"

  [[order.group]]
    id = "buildpack.add.2.id"
    version = "buildpack.add.2.version"

[[order]]

  [[order.group]]
    id = "buildpack.add.1.id"
    version = "buildpack.add.1.version"

  [[order.group]]
    id = "buildpack.2.id"
    version = "buildpack.2.version"

  [[order.group]]
    id = "buildpack.add.2.id"
    version = "buildpack.add.2.version"
`)
				})
			})

			when("from=builder is set last", func() {
				it("builder order is appended", func() {
					additionalBP1 := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack.add.1.id",
							Version: "buildpack.add.1.version",
						},
						WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
						WithOrder:  nil,
					})

					additionalBP2 := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "buildpack.add.2.id",
							Version: "buildpack.add.2.version",
						},
						WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
						WithOrder:  nil,
					})

					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{
							additionalBP1,
							additionalBP2,
							"from=builder",
						},
					}))
					h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())

					assertOrderEquals(`[[order]]

  [[order.group]]
    id = "buildpack.add.1.id"
    version = "buildpack.add.1.version"

  [[order.group]]
    id = "buildpack.add.2.id"
    version = "buildpack.add.2.version"

  [[order.group]]
    id = "buildpack.1.id"
    version = "buildpack.1.version"

[[order]]

  [[order.group]]
    id = "buildpack.add.1.id"
    version = "buildpack.add.1.version"

  [[order.group]]
    id = "buildpack.add.2.id"
    version = "buildpack.add.2.version"

  [[order.group]]
    id = "buildpack.2.id"
    version = "buildpack.2.version"
`)
				})
			})

			when("meta-buildpack is used", func() {
				it("resolves buildpack from builder", func() {
					buildpackTar := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
						WithAPI: api.MustParse("0.3"),
						WithInfo: dist.ModuleInfo{
							ID:      "metabuildpack.id",
							Version: "metabuildpack.version",
						},
						WithStacks: nil,
						WithOrder: dist.Order{{
							Group: []dist.ModuleRef{{
								ModuleInfo: dist.ModuleInfo{
									ID:      "buildpack.1.id",
									Version: "buildpack.1.version",
								},
								Optional: false,
							}, {
								ModuleInfo: dist.ModuleInfo{
									ID:      "buildpack.2.id",
									Version: "buildpack.2.version",
								},
								Optional: false,
							}},
						}},
					})

					err := subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{buildpackTar},
					})

					h.AssertNil(t, err)
				})
			})

			when("meta-buildpack folder is used", func() {
				it("resolves buildpack", func() {
					metaBuildpackFolder := filepath.Join(tmpDir, "meta-buildpack")
					err := os.Mkdir(metaBuildpackFolder, os.ModePerm)
					h.AssertNil(t, err)

					err = os.WriteFile(filepath.Join(metaBuildpackFolder, "buildpack.toml"), []byte(`
api = "0.2"

[buildpack]
  id = "local/meta-bp"
  version = "local-meta-bp-version"
  name = "Local Meta-Buildpack"

[[order]]
[[order.group]]
id = "local/meta-bp-dep"
version = "local-meta-bp-version"
					`), 0644)
					h.AssertNil(t, err)

					err = os.WriteFile(filepath.Join(metaBuildpackFolder, "package.toml"), []byte(`
[buildpack]
uri = "."

[[dependencies]]
uri = "../meta-buildpack-dependency"
					`), 0644)
					h.AssertNil(t, err)

					metaBuildpackDependencyFolder := filepath.Join(tmpDir, "meta-buildpack-dependency")
					err = os.Mkdir(metaBuildpackDependencyFolder, os.ModePerm)
					h.AssertNil(t, err)

					err = os.WriteFile(filepath.Join(metaBuildpackDependencyFolder, "buildpack.toml"), []byte(`
api = "0.2"

[buildpack]
  id = "local/meta-bp-dep"
  version = "local-meta-bp-version"
  name = "Local Meta-Buildpack Dependency"

[[stacks]]
  id = "*"
					`), 0644)
					h.AssertNil(t, err)

					err = subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{metaBuildpackFolder},
					})

					h.AssertNil(t, err)
					h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())

					bldr, err := builder.FromImage(defaultBuilderImage)
					h.AssertNil(t, err)

					buildpack1Info := dist.ModuleInfo{ID: "buildpack.1.id", Version: "buildpack.1.version"}
					buildpack2Info := dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}
					metaBuildpackInfo := dist.ModuleInfo{ID: "local/meta-bp", Version: "local-meta-bp-version", Name: "Local Meta-Buildpack"}
					metaBuildpackDependencyInfo := dist.ModuleInfo{ID: "local/meta-bp-dep", Version: "local-meta-bp-version", Name: "Local Meta-Buildpack Dependency"}
					h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
						buildpack1Info,
						buildpack2Info,
						metaBuildpackInfo,
						metaBuildpackDependencyInfo,
					})
				})

				it("fails if buildpack dependency could not be fetched", func() {
					metaBuildpackFolder := filepath.Join(tmpDir, "meta-buildpack")
					err := os.Mkdir(metaBuildpackFolder, os.ModePerm)
					h.AssertNil(t, err)

					err = os.WriteFile(filepath.Join(metaBuildpackFolder, "buildpack.toml"), []byte(`
api = "0.2"

[buildpack]
  id = "local/meta-bp"
  version = "local-meta-bp-version"
  name = "Local Meta-Buildpack"

[[order]]
[[order.group]]
id = "local/meta-bp-dep"
version = "local-meta-bp-version"
					`), 0644)
					h.AssertNil(t, err)

					err = os.WriteFile(filepath.Join(metaBuildpackFolder, "package.toml"), []byte(`
[buildpack]
uri = "."

[[dependencies]]
uri = "../meta-buildpack-dependency"

[[dependencies]]
uri = "../not-a-valid-dependency"
					`), 0644)
					h.AssertNil(t, err)

					metaBuildpackDependencyFolder := filepath.Join(tmpDir, "meta-buildpack-dependency")
					err = os.Mkdir(metaBuildpackDependencyFolder, os.ModePerm)
					h.AssertNil(t, err)

					err = os.WriteFile(filepath.Join(metaBuildpackDependencyFolder, "buildpack.toml"), []byte(`
api = "0.2"

[buildpack]
  id = "local/meta-bp-dep"
  version = "local-meta-bp-version"
  name = "Local Meta-Buildpack Dependency"

[[stacks]]
  id = "*"
					`), 0644)
					h.AssertNil(t, err)

					err = subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{metaBuildpackFolder},
					})
					h.AssertError(t, err, fmt.Sprintf("fetching package.toml dependencies (path='%s')", filepath.Join(metaBuildpackFolder, "package.toml")))
					h.AssertError(t, err, "fetching dependencies (uri='../not-a-valid-dependency',image='')")
				})
			})

			when("buildpackage image is used", func() {
				var fakePackage *fakes.Image

				it.Before(func() {
					fakePackage = makeFakePackage(t, tmpDir, defaultBuilderStackID)
					fakeImageFetcher.LocalImages[fakePackage.Name()] = fakePackage
				})

				it("all buildpacks are added to ephemeral builder", func() {
					err := subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{
							"example.com/some/package",
						},
					})

					h.AssertNil(t, err)
					h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
					bldr, err := builder.FromImage(defaultBuilderImage)
					h.AssertNil(t, err)
					h.AssertEq(t, bldr.Order(), dist.Order{
						{Group: []dist.ModuleRef{
							{ModuleInfo: dist.ModuleInfo{ID: "meta.buildpack.id", Version: "meta.buildpack.version"}},
						}},
						// Child buildpacks should not be added to order
					})
					h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
						{
							ID:      "buildpack.1.id",
							Version: "buildpack.1.version",
						},
						{
							ID:      "buildpack.2.id",
							Version: "buildpack.2.version",
						},
						{
							ID:      "meta.buildpack.id",
							Version: "meta.buildpack.version",
						},
						{
							ID:      "child.buildpack.id",
							Version: "child.buildpack.version",
						},
					})
					args := fakeImageFetcher.FetchCalls[fakePackage.Name()]
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/amd64")
				})

				it("fails when no metadata label on package", func() {
					h.AssertNil(t, fakePackage.SetLabel("io.buildpacks.buildpackage.metadata", ""))

					err := subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{
							"example.com/some/package",
						},
					})

					h.AssertError(t, err, "extracting buildpacks from 'example.com/some/package': could not find label 'io.buildpacks.buildpackage.metadata'")
				})

				it("fails when no bp layers label is on package", func() {
					h.AssertNil(t, fakePackage.SetLabel("io.buildpacks.buildpack.layers", ""))

					err := subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{
							"example.com/some/package",
						},
					})

					h.AssertError(t, err, "extracting buildpacks from 'example.com/some/package': could not find label 'io.buildpacks.buildpack.layers'")
				})
			})

			it("ensures buildpacks exist on builder", func() {
				h.AssertError(t, subject.Build(context.TODO(), BuildOptions{
					Image:      "some/app",
					Builder:    defaultBuilderName,
					ClearCache: true,
					Buildpacks: []string{"missing.bp@version"},
				}),
					"downloading buildpack: error reading missing.bp@version: invalid locator: InvalidLocator",
				)
			})

			when("from project descriptor", func() {
				when("id - no version is provided", func() {
					it("resolves version", func() {
						h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{Buildpacks: []projectTypes.Buildpack{{ID: "buildpack.1.id"}}},
							},
						}))
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())

						assertOrderEquals(`[[order]]

  [[order.group]]
    id = "buildpack.1.id"
    version = "buildpack.1.version"
`)
					})
				})
			})

			when("buildpacks include URIs", func() {
				var buildpackTgz string

				it.Before(func() {
					buildpackTgz = h.CreateTGZ(t, filepath.Join("testdata", "buildpack2"), "./", 0755)
				})

				it.After(func() {
					h.AssertNilE(t, os.Remove(buildpackTgz))
				})

				it("buildpacks are added to ephemeral builder", func() {
					err := subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Buildpacks: []string{
							"buildpack.1.id@buildpack.1.version",
							"buildpack.2.id@buildpack.2.version",
							filepath.Join("testdata", "buildpack"),
							buildpackTgz,
						},
					})

					h.AssertNil(t, err)
					h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
					bldr, err := builder.FromImage(defaultBuilderImage)
					h.AssertNil(t, err)
					buildpack1Info := dist.ModuleInfo{ID: "buildpack.1.id", Version: "buildpack.1.version"}
					buildpack2Info := dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}
					dirBuildpackInfo := dist.ModuleInfo{ID: "bp.one", Version: "1.2.3", Homepage: "http://one.buildpack"}
					tgzBuildpackInfo := dist.ModuleInfo{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"}
					h.AssertEq(t, bldr.Order(), dist.Order{
						{Group: []dist.ModuleRef{
							{ModuleInfo: buildpack1Info},
							{ModuleInfo: buildpack2Info},
							{ModuleInfo: dirBuildpackInfo},
							{ModuleInfo: tgzBuildpackInfo},
						}},
					})
					h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
						buildpack1Info,
						buildpack2Info,
						dirBuildpackInfo,
						tgzBuildpackInfo,
					})
				})

				when("uri is an http url", func() {
					var server *ghttp.Server

					it.Before(func() {
						server = ghttp.NewServer()
						server.AppendHandlers(func(w http.ResponseWriter, r *http.Request) {
							http.ServeFile(w, r, buildpackTgz)
						})
					})

					it.After(func() {
						server.Close()
					})

					it("adds the buildpack", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							Buildpacks: []string{
								"buildpack.1.id@buildpack.1.version",
								"buildpack.2.id@buildpack.2.version",
								server.URL(),
							},
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
						bldr, err := builder.FromImage(defaultBuilderImage)
						h.AssertNil(t, err)
						h.AssertEq(t, bldr.Order(), dist.Order{
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.1.id", Version: "buildpack.1.version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"}},
							}},
						})
						h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
							{ID: "buildpack.1.id", Version: "buildpack.1.version"},
							{ID: "buildpack.2.id", Version: "buildpack.2.version"},
							{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"},
						})
					})

					it("adds the buildpack from the project descriptor", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Buildpacks: []projectTypes.Buildpack{{
										URI: server.URL(),
									}},
								},
							},
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
						bldr, err := builder.FromImage(defaultBuilderImage)
						h.AssertNil(t, err)
						h.AssertEq(t, bldr.Order(), dist.Order{
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"}},
							}},
						})
						h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
							{ID: "buildpack.1.id", Version: "buildpack.1.version"},
							{ID: "buildpack.2.id", Version: "buildpack.2.version"},
							{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"},
						})
					})

					it("adds the pre buildpack from the project descriptor", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Pre: projectTypes.GroupAddition{
										Buildpacks: []projectTypes.Buildpack{{
											URI: server.URL(),
										}},
									},
								},
							},
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
						bldr, err := builder.FromImage(defaultBuilderImage)
						h.AssertNil(t, err)
						h.AssertEq(t, bldr.Order(), dist.Order{
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.1.id", Version: "buildpack.1.version"}},
							}},
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
							}},
						})
						h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
							{ID: "buildpack.1.id", Version: "buildpack.1.version"},
							{ID: "buildpack.2.id", Version: "buildpack.2.version"},
							{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"},
						})
					})

					it("adds the post buildpack from the project descriptor", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Post: projectTypes.GroupAddition{
										Buildpacks: []projectTypes.Buildpack{{
											URI: server.URL(),
										}},
									},
								},
							},
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
						bldr, err := builder.FromImage(defaultBuilderImage)
						h.AssertNil(t, err)
						h.AssertEq(t, bldr.Order(), dist.Order{
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.1.id", Version: "buildpack.1.version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"}},
							}},
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"}},
							}},
						})
						h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
							{ID: "buildpack.1.id", Version: "buildpack.1.version"},
							{ID: "buildpack.2.id", Version: "buildpack.2.version"},
							{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"},
						})
					})
				})

				when("pre and post buildpacks", func() {
					it("added from the project descriptor", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Pre: projectTypes.GroupAddition{
										Buildpacks: []projectTypes.Buildpack{{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
									},
									Post: projectTypes.GroupAddition{
										Buildpacks: []projectTypes.Buildpack{{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
									},
								},
							},
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
						bldr, err := builder.FromImage(defaultBuilderImage)
						h.AssertNil(t, err)
						h.AssertEq(t, bldr.Order(), dist.Order{
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.1.id", Version: "buildpack.1.version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
							}},
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.2.id", Version: "buildpack.2.version"}},
							}},
						})
						h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
							{ID: "buildpack.1.id", Version: "buildpack.1.version"},
							{ID: "buildpack.2.id", Version: "buildpack.2.version"},
						})
					})

					it("not added from the project descriptor", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							Buildpacks: []string{
								"buildpack.1.id@buildpack.1.version",
							},
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Pre: projectTypes.GroupAddition{
										Buildpacks: []projectTypes.Buildpack{{ID: "some-other-buildpack-id", Version: "some-other-buildpack-version"}},
									},
									Post: projectTypes.GroupAddition{
										Buildpacks: []projectTypes.Buildpack{{ID: "yet-other-buildpack-id", Version: "yet-other-buildpack-version"}},
									},
								},
							},
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
						bldr, err := builder.FromImage(defaultBuilderImage)
						h.AssertNil(t, err)
						h.AssertEq(t, bldr.Order(), dist.Order{
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "buildpack.1.id", Version: "buildpack.1.version"}},
							}},
						})
						h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
							{ID: "buildpack.1.id", Version: "buildpack.1.version"},
							{ID: "buildpack.2.id", Version: "buildpack.2.version"},
						})
					})
				})

				when("added buildpack's mixins are not satisfied", func() {
					it.Before(func() {
						h.AssertNil(t, defaultBuilderImage.SetLabel("io.buildpacks.stack.mixins", `["mixinX", "build:mixinY"]`))
						h.AssertNil(t, fakeDefaultRunImage.SetLabel("io.buildpacks.stack.mixins", `["mixinX", "run:mixinZ"]`))
					})

					it("succeeds", func() {
						h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultBuilderName,
							Buildpacks: []string{
								buildpackTgz, // requires mixinA, build:mixinB, run:mixinC
							},
						}))
					})

					when("platform API < 0.12", func() {
						it.Before(func() {
							setAPIs(t, defaultBuilderImage, []string{"0.8"}, []string{"0.11"})
						})

						it("returns an error", func() {
							err := subject.Build(context.TODO(), BuildOptions{
								Image:   "some/app",
								Builder: defaultBuilderName,
								Buildpacks: []string{
									buildpackTgz, // requires mixinA, build:mixinB, run:mixinC
								},
							})

							h.AssertError(t, err, "validating stack mixins: buildpack 'some-other-buildpack-id@some-other-buildpack-version' requires missing mixin(s): build:mixinB, mixinA, run:mixinC")
						})
					})
				})

				when("buildpack is inline", func() {
					var (
						tmpDir string
					)

					it.Before(func() {
						var err error
						tmpDir, err = os.MkdirTemp("", "project-desc")
						h.AssertNil(t, err)
					})

					it.After(func() {
						err := os.RemoveAll(tmpDir)
						h.AssertNil(t, err)
					})

					it("all buildpacks are added to ephemeral builder", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Buildpacks: []projectTypes.Buildpack{{
										ID: "my/inline",
										Script: projectTypes.Script{
											API:    "0.4",
											Inline: "touch foo.txt",
										},
									}},
								},
							},
							ProjectDescriptorBaseDir: tmpDir,
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
						bldr, err := builder.FromImage(defaultBuilderImage)
						h.AssertNil(t, err)
						h.AssertEq(t, bldr.Order(), dist.Order{
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "my/inline", Version: "0.0.0"}},
							}},
						})
						h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
							{ID: "buildpack.1.id", Version: "buildpack.1.version"},
							{ID: "buildpack.2.id", Version: "buildpack.2.version"},
							{ID: "my/inline", Version: "0.0.0"},
						})
					})

					it("sets version if version is set", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Buildpacks: []projectTypes.Buildpack{{
										ID:      "my/inline",
										Version: "1.0.0-my-version",
										Script: projectTypes.Script{
											API:    "0.4",
											Inline: "touch foo.txt",
										},
									}},
								},
							},
							ProjectDescriptorBaseDir: tmpDir,
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
						bldr, err := builder.FromImage(defaultBuilderImage)
						h.AssertNil(t, err)
						h.AssertEq(t, bldr.Order(), dist.Order{
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "my/inline", Version: "1.0.0-my-version"}},
							}},
						})
						h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
							{ID: "buildpack.1.id", Version: "buildpack.1.version"},
							{ID: "buildpack.2.id", Version: "buildpack.2.version"},
							{ID: "my/inline", Version: "1.0.0-my-version"},
						})
					})

					it("fails if there is no API", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Buildpacks: []projectTypes.Buildpack{{
										ID: "my/inline",
										Script: projectTypes.Script{
											Inline: "touch foo.txt",
										},
									}},
								},
							},
							ProjectDescriptorBaseDir: tmpDir,
						})

						h.AssertEq(t, "Missing API version for inline buildpack", err.Error())
					})

					it("fails if there is no ID", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Buildpacks: []projectTypes.Buildpack{{
										Script: projectTypes.Script{
											API:    "0.4",
											Inline: "touch foo.txt",
										},
									}},
								},
							},
							ProjectDescriptorBaseDir: tmpDir,
						})

						h.AssertEq(t, "Invalid buildpack definition", err.Error())
					})

					it("ignores script if there is a URI", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Build: projectTypes.Build{
									Buildpacks: []projectTypes.Buildpack{{
										ID:      "buildpack.1.id",
										URI:     "some-uri",
										Version: "buildpack.1.version",
										Script: projectTypes.Script{
											Inline: "touch foo.txt",
										},
									}},
								},
							},
							ProjectDescriptorBaseDir: tmpDir,
						})

						h.AssertContains(t, err.Error(), "extracting from registry 'some-uri'")
					})
				})

				when("buildpack is from a registry", func() {
					var (
						fakePackage     *fakes.Image
						tmpDir          string
						registryFixture string
						packHome        string

						configPath string
					)

					it.Before(func() {
						var err error
						tmpDir, err = os.MkdirTemp("", "registry")
						h.AssertNil(t, err)

						packHome = filepath.Join(tmpDir, ".pack")
						err = os.MkdirAll(packHome, 0755)
						h.AssertNil(t, err)
						os.Setenv("PACK_HOME", packHome)

						registryFixture = h.CreateRegistryFixture(t, tmpDir, filepath.Join("testdata", "registry"))

						configPath = filepath.Join(packHome, "config.toml")
						h.AssertNil(t, cfg.Write(cfg.Config{
							Registries: []cfg.Registry{
								{
									Name: "some-registry",
									Type: "github",
									URL:  registryFixture,
								},
							},
						}, configPath))

						_, err = rg.NewRegistryCache(logger, tmpDir, registryFixture)
						h.AssertNil(t, err)

						childBuildpackTar := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
							WithAPI: api.MustParse("0.3"),
							WithInfo: dist.ModuleInfo{
								ID:      "example/foo",
								Version: "1.0.0",
							},
							WithStacks: []dist.Stack{
								{ID: defaultBuilderStackID},
							},
						})

						bpLayers := dist.ModuleLayers{
							"example/foo": {
								"1.0.0": {
									API: api.MustParse("0.3"),
									Stacks: []dist.Stack{
										{ID: defaultBuilderStackID},
									},
									LayerDiffID: diffIDForFile(t, childBuildpackTar),
								},
							},
						}

						md := buildpack.Metadata{
							ModuleInfo: dist.ModuleInfo{
								ID:      "example/foo",
								Version: "1.0.0",
							},
							Stacks: []dist.Stack{
								{ID: defaultBuilderStackID},
							},
						}

						fakePackage = fakes.NewImage("example.com/some/package@sha256:8c27fe111c11b722081701dfed3bd55e039b9ce92865473cf4cdfa918071c566", "", nil)
						h.AssertNil(t, dist.SetLabel(fakePackage, "io.buildpacks.buildpack.layers", bpLayers))
						h.AssertNil(t, dist.SetLabel(fakePackage, "io.buildpacks.buildpackage.metadata", md))

						h.AssertNil(t, fakePackage.AddLayer(childBuildpackTar))

						fakeImageFetcher.LocalImages[fakePackage.Name()] = fakePackage
					})

					it.After(func() {
						os.Unsetenv("PACK_HOME")
						h.AssertNil(t, os.RemoveAll(tmpDir))
					})

					it("all buildpacks are added to ephemeral builder", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							Buildpacks: []string{
								"urn:cnb:registry:example/foo@1.0.0",
							},
							Registry: "some-registry",
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())
						bldr, err := builder.FromImage(defaultBuilderImage)
						h.AssertNil(t, err)
						h.AssertEq(t, bldr.Order(), dist.Order{
							{Group: []dist.ModuleRef{
								{ModuleInfo: dist.ModuleInfo{ID: "example/foo", Version: "1.0.0"}},
							}},
						})
						h.AssertEq(t, bldr.Buildpacks(), []dist.ModuleInfo{
							{ID: "buildpack.1.id", Version: "buildpack.1.version"},
							{ID: "buildpack.2.id", Version: "buildpack.2.version"},
							{ID: "example/foo", Version: "1.0.0"},
						})
					})
				})
			})
		})

		when("Extensions option", func() {
			it.Before(func() {
				subject.experimental = true
				defaultBuilderImage.SetLabel("io.buildpacks.buildpack.order-extensions", `[{"group":[{"id":"extension.1.id","version":"extension.1.version"}]}, {"group":[{"id":"extension.2.id","version":"extension.2.version"}]}]`)
				defaultWindowsBuilderImage.SetLabel("io.buildpacks.buildpack.order-extensions", `[{"group":[{"id":"extension.1.id","version":"extension.1.version"}]}, {"group":[{"id":"extension.2.id","version":"extension.2.version"}]}]`)
			})

			assertOrderEquals := func(content string) {
				t.Helper()

				orderLayer, err := defaultBuilderImage.FindLayerWithPath("/cnb/order.toml")
				h.AssertNil(t, err)
				h.AssertOnTarEntry(t, orderLayer, "/cnb/order.toml", h.ContentEquals(content))
			}

			it("builder order-extensions is overwritten", func() {
				additionalEx := ifakes.CreateExtensionTar(t, tmpDir, dist.ExtensionDescriptor{
					WithAPI: api.MustParse("0.7"),
					WithInfo: dist.ModuleInfo{
						ID:      "extension.add.1.id",
						Version: "extension.add.1.version",
					},
				})

				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:      "some/app",
					Builder:    defaultBuilderName,
					ClearCache: true,
					Extensions: []string{additionalEx},
				}))
				h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())

				assertOrderEquals(`[[order]]

  [[order.group]]
    id = "buildpack.1.id"
    version = "buildpack.1.version"

[[order]]

  [[order.group]]
    id = "buildpack.2.id"
    version = "buildpack.2.version"

[[order-extensions]]

  [[order-extensions.group]]
    id = "extension.add.1.id"
    version = "extension.add.1.version"
`)
			})

			when("id - no version is provided", func() {
				it("resolves version", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						ClearCache: true,
						Extensions: []string{"extension.1.id"},
					}))
					h.AssertEq(t, fakeLifecycle.Opts.Builder.Name(), defaultBuilderImage.Name())

					assertOrderEquals(`[[order]]

  [[order.group]]
    id = "buildpack.1.id"
    version = "buildpack.1.version"

[[order]]

  [[order.group]]
    id = "buildpack.2.id"
    version = "buildpack.2.version"

[[order-extensions]]

  [[order-extensions.group]]
    id = "extension.1.id"
    version = "extension.1.version"
`)
				})
			})
		})

		//TODO: "all buildpacks are added to ephemeral builder" test after extractPackaged() is completed.

		when("ProjectDescriptor", func() {
			when("project metadata", func() {
				when("not experimental", func() {
					it("does not set project source", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Project: projectTypes.Project{
									Version:   "1.2.3",
									SourceURL: "https://example.com",
								},
							},
						})

						h.AssertNil(t, err)
						h.AssertNil(t, fakeLifecycle.Opts.ProjectMetadata.Source)
					})
				})

				when("is experimental", func() {
					it.Before(func() {
						subject.experimental = true
					})

					when("missing information", func() {
						it("does not set project source", func() {
							err := subject.Build(context.TODO(), BuildOptions{
								Image:             "some/app",
								Builder:           defaultBuilderName,
								ClearCache:        true,
								ProjectDescriptor: projectTypes.Descriptor{},
							})

							h.AssertNil(t, err)
							h.AssertNil(t, fakeLifecycle.Opts.ProjectMetadata.Source)
						})
					})

					it("sets project source", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							ClearCache: true,
							ProjectDescriptor: projectTypes.Descriptor{
								Project: projectTypes.Project{
									Version:   "1.2.3",
									SourceURL: "https://example.com",
								},
							},
						})

						h.AssertNil(t, err)
						h.AssertNotNil(t, fakeLifecycle.Opts.ProjectMetadata.Source)
						h.AssertEq(t, fakeLifecycle.Opts.ProjectMetadata.Source, &files.ProjectSource{
							Type:     "project",
							Version:  map[string]interface{}{"declared": "1.2.3"},
							Metadata: map[string]interface{}{"url": "https://example.com"},
						})
					})
				})
			})
		})

		when("Env option", func() {
			it("should set the env on the ephemeral builder", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:   "some/app",
					Builder: defaultBuilderName,
					Env: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				}))
				layerTar, err := defaultBuilderImage.FindLayerWithPath("/platform/env/key1")
				h.AssertNil(t, err)
				h.AssertTarFileContents(t, layerTar, "/platform/env/key1", `value1`)
				h.AssertTarFileContents(t, layerTar, "/platform/env/key2", `value2`)
			})
		})

		when("Publish option", func() {
			var remoteRunImage, builderWithoutLifecycleImageOrCreator *fakes.Image

			it.Before(func() {
				remoteRunImage = fakes.NewImage("default/run", "", nil)
				h.AssertNil(t, remoteRunImage.SetLabel("io.buildpacks.stack.id", defaultBuilderStackID))
				h.AssertNil(t, remoteRunImage.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "mixinX", "run:mixinZ"]`))
				fakeImageFetcher.RemoteImages[remoteRunImage.Name()] = remoteRunImage

				builderWithoutLifecycleImageOrCreator = newFakeBuilderImage(
					t,
					tmpDir,
					"example.com/supportscreator/builder:tag",
					"some.stack.id",
					defaultRunImageName,
					"0.3.0",
					newLinuxImage,
				)
				h.AssertNil(t, builderWithoutLifecycleImageOrCreator.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "build:mixinB", "mixinX", "build:mixinY"]`))
				fakeImageFetcher.LocalImages[builderWithoutLifecycleImageOrCreator.Name()] = builderWithoutLifecycleImageOrCreator
			})

			it.After(func() {
				remoteRunImage.Cleanup()
				builderWithoutLifecycleImageOrCreator.Cleanup()
			})

			when("true", func() {
				it("uses a remote run image", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						Publish: true,
					}))
					h.AssertEq(t, fakeLifecycle.Opts.Publish, true)

					args := fakeImageFetcher.FetchCalls[defaultBuilderName]
					h.AssertEq(t, args.Daemon, true)

					args = fakeImageFetcher.FetchCalls["default/run"]
					h.AssertEq(t, args.Daemon, false)
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/amd64")
				})

				when("builder is untrusted", func() {
					when("lifecycle image is available", func() {
						it("uses the 5 phases with the lifecycle image", func() {
							origLifecyleName := fakeLifecycleImage.Name()

							h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:        "some/app",
								Builder:      defaultBuilderName,
								Publish:      true,
								TrustBuilder: func(string) bool { return false },
							}))
							h.AssertEq(t, fakeLifecycle.Opts.UseCreator, false)
							h.AssertContains(t, fakeLifecycle.Opts.LifecycleImage, "pack.local/lifecycle")
							args := fakeImageFetcher.FetchCalls[origLifecyleName]
							h.AssertNotNil(t, args)
							h.AssertEq(t, args.Daemon, true)
							h.AssertEq(t, args.PullPolicy, image.PullAlways)
							h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/amd64")
						})
						it("parses the versions correctly", func() {
							fakeLifecycleImage.SetLabel("io.buildpacks.lifecycle.apis", "{\"platform\":{\"deprecated\":[\"0.1\",\"0.2\",\"0.3\",\"0.4\",\"0.5\",\"0.6\"],\"supported\":[\"0.7\",\"0.8\",\"0.9\",\"0.10\",\"0.11\",\"0.12\"]}}")

							h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:        "some/app",
								Builder:      defaultBuilderName,
								Publish:      true,
								TrustBuilder: func(string) bool { return false },
							}))
							h.AssertSliceContainsInOrder(t, fakeLifecycle.Opts.LifecycleApis, "0.1", "0.2", "0.3", "0.4", "0.5", "0.6", "0.7", "0.8", "0.9", "0.10", "0.11", "0.12")
						})
					})

					when("lifecycle image is not available", func() {
						it("errors", func() {
							h.AssertNotNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:        "some/app",
								Builder:      builderWithoutLifecycleImageOrCreator.Name(),
								Publish:      true,
								TrustBuilder: func(string) bool { return false },
							}))
						})
					})
				})

				when("builder is trusted", func() {
					when("lifecycle supports creator", func() {
						it("uses the creator with the provided builder", func() {
							h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:        "some/app",
								Builder:      defaultBuilderName,
								Publish:      true,
								TrustBuilder: func(string) bool { return true },
							}))
							h.AssertEq(t, fakeLifecycle.Opts.UseCreator, true)

							args := fakeImageFetcher.FetchCalls[fakeLifecycleImage.Name()]
							h.AssertNil(t, args)
						})

						when("additional buildpacks were added", func() {
							it("uses creator when additional buildpacks are provided and TrustExtraBuildpacks is set", func() {
								additionalBP := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.3"),
									WithInfo: dist.ModuleInfo{
										ID:      "buildpack.add.1.id",
										Version: "buildpack.add.1.version",
									},
									WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
									WithOrder:  nil,
								})

								h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
									Image:                "some/app",
									Builder:              defaultBuilderName,
									Publish:              true,
									TrustBuilder:         func(string) bool { return true },
									TrustExtraBuildpacks: true,
									Buildpacks:           []string{additionalBP},
								}))
								h.AssertEq(t, fakeLifecycle.Opts.UseCreator, true)
							})

							it("uses the 5 phases with the lifecycle image", func() {
								additionalBP := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.3"),
									WithInfo: dist.ModuleInfo{
										ID:      "buildpack.add.1.id",
										Version: "buildpack.add.1.version",
									},
									WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
									WithOrder:  nil,
								})

								h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
									Image:        "some/app",
									Builder:      defaultBuilderName,
									Publish:      true,
									TrustBuilder: func(string) bool { return true },
									Buildpacks:   []string{additionalBP},
								}))
								h.AssertEq(t, fakeLifecycle.Opts.UseCreator, false)
								h.AssertEq(t, fakeLifecycle.Opts.LifecycleImage, fakeLifecycleImage.Name())

								h.AssertContains(t, outBuf.String(), "Builder is trusted but additional modules were added; using the untrusted (5 phases) build flow")
							})

							when("from project descriptor", func() {
								it("uses the 5 phases with the lifecycle image", func() {
									additionalBP := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
										WithAPI: api.MustParse("0.3"),
										WithInfo: dist.ModuleInfo{
											ID:      "buildpack.add.1.id",
											Version: "buildpack.add.1.version",
										},
										WithStacks: []dist.Stack{{ID: defaultBuilderStackID}},
										WithOrder:  nil,
									})

									h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
										Image:        "some/app",
										Builder:      defaultBuilderName,
										Publish:      true,
										TrustBuilder: func(string) bool { return true },
										ProjectDescriptor: projectTypes.Descriptor{Build: projectTypes.Build{
											Buildpacks: []projectTypes.Buildpack{{
												URI: additionalBP,
											}},
										}},
									}))
									h.AssertEq(t, fakeLifecycle.Opts.UseCreator, false)
									h.AssertEq(t, fakeLifecycle.Opts.LifecycleImage, fakeLifecycleImage.Name())

									h.AssertContains(t, outBuf.String(), "Builder is trusted but additional modules were added; using the untrusted (5 phases) build flow")
								})

								when("inline buildpack", func() {
									it("uses the creator with the provided builder", func() {
										h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
											Image:        "some/app",
											Builder:      defaultBuilderName,
											Publish:      true,
											TrustBuilder: func(string) bool { return true },
											ProjectDescriptor: projectTypes.Descriptor{Build: projectTypes.Build{
												Buildpacks: []projectTypes.Buildpack{{
													ID:      "buildpack.add.1.id",
													Version: "buildpack.add.1.version",
													Script: projectTypes.Script{
														API:    "0.10",
														Inline: "echo hello",
													},
												}},
											}},
										}))
										h.AssertEq(t, fakeLifecycle.Opts.UseCreator, true)

										args := fakeImageFetcher.FetchCalls[fakeLifecycleImage.Name()]
										h.AssertNil(t, args)
									})
								})
							})
						})
					})

					when("lifecycle doesn't support creator", func() {
						// the default test builder (example.com/default/builder:tag) has lifecycle version 0.3.0, so creator is not supported
						it("uses the 5 phases with the provided builder", func() {
							h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:        "some/app",
								Builder:      builderWithoutLifecycleImageOrCreator.Name(),
								Publish:      true,
								TrustBuilder: func(string) bool { return true },
							}))
							h.AssertEq(t, fakeLifecycle.Opts.UseCreator, false)
							h.AssertEq(t, fakeLifecycle.Opts.LifecycleImage, builderWithoutLifecycleImageOrCreator.Name())

							args := fakeImageFetcher.FetchCalls[fakeLifecycleImage.Name()]
							h.AssertNil(t, args)
						})
					})
				})
			})

			when("false", func() {
				it("uses a local run image", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						Publish: false,
					}))
					h.AssertEq(t, fakeLifecycle.Opts.Publish, false)

					args := fakeImageFetcher.FetchCalls["default/run"]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)

					args = fakeImageFetcher.FetchCalls[defaultBuilderName]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
				})

				when("builder is untrusted", func() {
					when("lifecycle image is available", func() {
						it("uses the 5 phases with the lifecycle image", func() {
							origLifecyleName := fakeLifecycleImage.Name()
							h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:        "some/app",
								Builder:      defaultBuilderName,
								Publish:      false,
								TrustBuilder: func(string) bool { return false },
							}))
							h.AssertEq(t, fakeLifecycle.Opts.UseCreator, false)
							h.AssertContains(t, fakeLifecycle.Opts.LifecycleImage, "pack.local/lifecycle")
							args := fakeImageFetcher.FetchCalls[origLifecyleName]
							h.AssertNotNil(t, args)
							h.AssertEq(t, args.Daemon, true)
							h.AssertEq(t, args.PullPolicy, image.PullAlways)
							h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/amd64")
						})
					})

					when("lifecycle image is not available", func() {
						it("errors", func() {
							h.AssertNotNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:        "some/app",
								Builder:      builderWithoutLifecycleImageOrCreator.Name(),
								Publish:      false,
								TrustBuilder: func(string) bool { return false },
							}))
						})
					})
				})

				when("builder is trusted", func() {
					when("lifecycle supports creator", func() {
						it("uses the creator with the provided builder", func() {
							h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:        "some/app",
								Builder:      defaultBuilderName,
								Publish:      false,
								TrustBuilder: func(string) bool { return true },
							}))
							h.AssertEq(t, fakeLifecycle.Opts.UseCreator, true)

							args := fakeImageFetcher.FetchCalls[fakeLifecycleImage.Name()]
							h.AssertNil(t, args)
						})
					})

					when("lifecycle doesn't support creator", func() {
						// the default test builder (example.com/default/builder:tag) has lifecycle version 0.3.0, so creator is not supported
						it("uses the 5 phases with the provided builder", func() {
							h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
								Image:        "some/app",
								Builder:      builderWithoutLifecycleImageOrCreator.Name(),
								Publish:      false,
								TrustBuilder: func(string) bool { return true },
							}))
							h.AssertEq(t, fakeLifecycle.Opts.UseCreator, false)
							h.AssertEq(t, fakeLifecycle.Opts.LifecycleImage, builderWithoutLifecycleImageOrCreator.Name())

							args := fakeImageFetcher.FetchCalls[fakeLifecycleImage.Name()]
							h.AssertNil(t, args)
						})
					})
				})
			})
		})

		when("Platform option", func() {
			var fakePackage imgutil.Image

			it.Before(func() {
				fakePackage = makeFakePackage(t, tmpDir, defaultBuilderStackID)
				fakeImageFetcher.LocalImages[fakePackage.Name()] = fakePackage
			})

			when("provided", func() {
				it("uses the provided platform to pull the builder, run image, packages, and lifecycle image", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						Buildpacks: []string{
							"example.com/some/package",
						},
						Platform:   "linux/arm64",
						PullPolicy: image.PullAlways,
					}))

					args := fakeImageFetcher.FetchCalls[defaultBuilderName]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/arm64")

					args = fakeImageFetcher.FetchCalls["default/run"]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/arm64")

					args = fakeImageFetcher.FetchCalls[fakePackage.Name()]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/arm64")

					args = fakeImageFetcher.FetchCalls[fmt.Sprintf("%s:%s", cfg.DefaultLifecycleImageRepo, builder.DefaultLifecycleVersion)]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/arm64")
				})
			})

			when("not provided", func() {
				it("defaults to builder os/arch", func() {
					// defaultBuilderImage has linux/amd64

					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						Buildpacks: []string{
							"example.com/some/package",
						},
						PullPolicy: image.PullAlways,
					}))

					args := fakeImageFetcher.FetchCalls[defaultBuilderName]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
					h.AssertEq(t, args.Target, (*dist.Target)(nil))

					args = fakeImageFetcher.FetchCalls["default/run"]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/amd64")

					args = fakeImageFetcher.FetchCalls[fakePackage.Name()]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/amd64")

					args = fakeImageFetcher.FetchCalls[fmt.Sprintf("%s:%s", cfg.DefaultLifecycleImageRepo, builder.DefaultLifecycleVersion)]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/amd64")
				})
			})
		})

		when("PullPolicy", func() {
			when("never", func() {
				it("uses the local builder and run images without updating", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						PullPolicy: image.PullNever,
					}))

					args := fakeImageFetcher.FetchCalls["default/run"]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullNever)

					args = fakeImageFetcher.FetchCalls[defaultBuilderName]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullNever)

					args = fakeImageFetcher.FetchCalls[fmt.Sprintf("%s:%s", cfg.DefaultLifecycleImageRepo, builder.DefaultLifecycleVersion)]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullNever)
					h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/amd64")
				})
			})

			when("containerized pack", func() {
				it.Before(func() {
					RunningInContainer = func() bool {
						return true
					}
				})

				when("--pull-policy=always", func() {
					it("does not warn", func() {
						h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							PullPolicy: image.PullAlways,
						}))

						h.AssertNotContains(t, outBuf.String(), "failing to pull build inputs from a remote registry is insecure")
					})
				})

				when("not --pull-policy=always", func() {
					it("warns", func() {
						h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							PullPolicy: image.PullNever,
						}))

						h.AssertContains(t, outBuf.String(), "failing to pull build inputs from a remote registry is insecure")
					})
				})
			})

			when("always", func() {
				it("uses pulls the builder and run image before using them", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:      "some/app",
						Builder:    defaultBuilderName,
						PullPolicy: image.PullAlways,
					}))

					args := fakeImageFetcher.FetchCalls["default/run"]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)

					args = fakeImageFetcher.FetchCalls[defaultBuilderName]
					h.AssertEq(t, args.Daemon, true)
					h.AssertEq(t, args.PullPolicy, image.PullAlways)
				})
			})
		})

		when("ProxyConfig option", func() {
			when("ProxyConfig is nil", func() {
				it.Before(func() {
					h.AssertNil(t, os.Setenv("http_proxy", "other-http-proxy"))
					h.AssertNil(t, os.Setenv("https_proxy", "other-https-proxy"))
					h.AssertNil(t, os.Setenv("no_proxy", "other-no-proxy"))
				})

				when("*_PROXY env vars are set", func() {
					it.Before(func() {
						h.AssertNil(t, os.Setenv("HTTP_PROXY", "some-http-proxy"))
						h.AssertNil(t, os.Setenv("HTTPS_PROXY", "some-https-proxy"))
						h.AssertNil(t, os.Setenv("NO_PROXY", "some-no-proxy"))
					})

					it.After(func() {
						h.AssertNilE(t, os.Unsetenv("HTTP_PROXY"))
						h.AssertNilE(t, os.Unsetenv("HTTPS_PROXY"))
						h.AssertNilE(t, os.Unsetenv("NO_PROXY"))
					})

					it("defaults to the *_PROXY environment variables", func() {
						h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultBuilderName,
						}))
						h.AssertEq(t, fakeLifecycle.Opts.HTTPProxy, "some-http-proxy")
						h.AssertEq(t, fakeLifecycle.Opts.HTTPSProxy, "some-https-proxy")
						h.AssertEq(t, fakeLifecycle.Opts.NoProxy, "some-no-proxy")
					})
				})

				it("falls back to the *_proxy environment variables", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
					}))
					h.AssertEq(t, fakeLifecycle.Opts.HTTPProxy, "other-http-proxy")
					h.AssertEq(t, fakeLifecycle.Opts.HTTPSProxy, "other-https-proxy")
					h.AssertEq(t, fakeLifecycle.Opts.NoProxy, "other-no-proxy")
				})
			}, spec.Sequential())

			when("ProxyConfig is not nil", func() {
				it("passes the values through", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
						ProxyConfig: &ProxyConfig{
							HTTPProxy:  "custom-http-proxy",
							HTTPSProxy: "custom-https-proxy",
							NoProxy:    "custom-no-proxy",
						},
					}))
					h.AssertEq(t, fakeLifecycle.Opts.HTTPProxy, "custom-http-proxy")
					h.AssertEq(t, fakeLifecycle.Opts.HTTPSProxy, "custom-https-proxy")
					h.AssertEq(t, fakeLifecycle.Opts.NoProxy, "custom-no-proxy")
				})
			})
		})

		when("Network option", func() {
			it("passes the value through", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Image:   "some/app",
					Builder: defaultBuilderName,
					ContainerConfig: ContainerConfig{
						Network: "some-network",
					},
				}))
				h.AssertEq(t, fakeLifecycle.Opts.Network, "some-network")
			})
		})

		when("Lifecycle option", func() {
			when("Platform API", func() {
				for _, supportedPlatformAPI := range []string{"0.3", "0.4"} {
					var (
						supportedPlatformAPI = supportedPlatformAPI
						compatibleBuilder    *fakes.Image
					)

					when(fmt.Sprintf("lifecycle platform API is compatible (%s)", supportedPlatformAPI), func() {
						it.Before(func() {
							compatibleBuilder = ifakes.NewFakeBuilderImage(t,
								tmpDir,
								"compatible-"+defaultBuilderName,
								defaultBuilderStackID,
								"1234",
								"5678",
								builder.Metadata{
									Stack: builder.StackMetadata{
										RunImage: builder.RunImageMetadata{
											Image: "default/run",
											Mirrors: []string{
												"registry1.example.com/run/mirror",
												"registry2.example.com/run/mirror",
											},
										},
									},
									Lifecycle: builder.LifecycleMetadata{
										LifecycleInfo: builder.LifecycleInfo{
											Version: &builder.Version{
												Version: *semver.MustParse(builder.DefaultLifecycleVersion),
											},
										},
										APIs: builder.LifecycleAPIs{
											Buildpack: builder.APIVersions{
												Supported: builder.APISet{api.MustParse("0.2"), api.MustParse("0.3"), api.MustParse("0.4")},
											},
											Platform: builder.APIVersions{
												Supported: builder.APISet{api.MustParse(supportedPlatformAPI)},
											},
										},
									},
								},
								nil,
								nil,
								nil,
								nil,
								newLinuxImage,
							)

							fakeImageFetcher.LocalImages[compatibleBuilder.Name()] = compatibleBuilder
						})

						it("should succeed", func() {
							err := subject.Build(context.TODO(), BuildOptions{
								Image:   "some/app",
								Builder: compatibleBuilder.Name(),
							})

							h.AssertNil(t, err)
						})
					})
				}

				when("lifecycle Platform API is not compatible", func() {
					var incompatibleBuilderImage *fakes.Image
					it.Before(func() {
						incompatibleBuilderImage = ifakes.NewFakeBuilderImage(t,
							tmpDir,
							"incompatible-"+defaultBuilderName,
							defaultBuilderStackID,
							"1234",
							"5678",
							builder.Metadata{
								Stack: builder.StackMetadata{
									RunImage: builder.RunImageMetadata{
										Image: "default/run",
										Mirrors: []string{
											"registry1.example.com/run/mirror",
											"registry2.example.com/run/mirror",
										},
									},
								},
								Lifecycle: builder.LifecycleMetadata{
									LifecycleInfo: builder.LifecycleInfo{
										Version: &builder.Version{
											Version: *semver.MustParse(builder.DefaultLifecycleVersion),
										},
									},
									API: builder.LifecycleAPI{
										BuildpackVersion: api.MustParse("0.3"),
										PlatformVersion:  api.MustParse("0.1"),
									},
								},
							},
							nil,
							nil,
							nil,
							nil,
							newLinuxImage,
						)

						fakeImageFetcher.LocalImages[incompatibleBuilderImage.Name()] = incompatibleBuilderImage
					})

					it.After(func() {
						incompatibleBuilderImage.Cleanup()
					})

					it("should error", func() {
						builderName := incompatibleBuilderImage.Name()

						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: builderName,
						})

						h.AssertError(t, err, fmt.Sprintf("Builder %s is incompatible with this version of pack", style.Symbol(builderName)))
					})
				})

				when("supported Platform APIs not specified", func() {
					var badBuilderImage *fakes.Image
					it.Before(func() {
						badBuilderImage = ifakes.NewFakeBuilderImage(t,
							tmpDir,
							"incompatible-"+defaultBuilderName,
							defaultBuilderStackID,
							"1234",
							"5678",
							builder.Metadata{
								Stack: builder.StackMetadata{
									RunImage: builder.RunImageMetadata{
										Image: "default/run",
										Mirrors: []string{
											"registry1.example.com/run/mirror",
											"registry2.example.com/run/mirror",
										},
									},
								},
								Lifecycle: builder.LifecycleMetadata{
									LifecycleInfo: builder.LifecycleInfo{
										Version: &builder.Version{
											Version: *semver.MustParse(builder.DefaultLifecycleVersion),
										},
									},
									APIs: builder.LifecycleAPIs{
										Buildpack: builder.APIVersions{Supported: builder.APISet{api.MustParse("0.2")}},
									},
								},
							},
							nil,
							nil,
							nil,
							nil,
							newLinuxImage,
						)

						fakeImageFetcher.LocalImages[badBuilderImage.Name()] = badBuilderImage
					})

					it.After(func() {
						badBuilderImage.Cleanup()
					})

					it("should error", func() {
						builderName := badBuilderImage.Name()

						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: builderName,
						})

						h.AssertError(t, err, "supported Lifecycle Platform APIs not specified")
					})
				})
			})

			when("Buildpack API", func() {
				when("supported Buildpack APIs not specified", func() {
					var badBuilderImage *fakes.Image
					it.Before(func() {
						badBuilderImage = ifakes.NewFakeBuilderImage(t,
							tmpDir,
							"incompatible-"+defaultBuilderName,
							defaultBuilderStackID,
							"1234",
							"5678",
							builder.Metadata{
								Stack: builder.StackMetadata{
									RunImage: builder.RunImageMetadata{
										Image: "default/run",
										Mirrors: []string{
											"registry1.example.com/run/mirror",
											"registry2.example.com/run/mirror",
										},
									},
								},
								Lifecycle: builder.LifecycleMetadata{
									LifecycleInfo: builder.LifecycleInfo{
										Version: &builder.Version{
											Version: *semver.MustParse(builder.DefaultLifecycleVersion),
										},
									},
									APIs: builder.LifecycleAPIs{
										Platform: builder.APIVersions{Supported: builder.APISet{api.MustParse("0.4")}},
									},
								},
							},
							nil,
							nil,
							nil,
							nil,
							newLinuxImage,
						)

						fakeImageFetcher.LocalImages[badBuilderImage.Name()] = badBuilderImage
					})

					it.After(func() {
						badBuilderImage.Cleanup()
					})

					it("should error", func() {
						builderName := badBuilderImage.Name()

						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: builderName,
						})

						h.AssertError(t, err, "supported Lifecycle Buildpack APIs not specified")
					})
				})
			})

			when("use creator with extensions", func() {
				when("lifecycle is old", func() {
					it("false", func() {
						oldLifecycleBuilder := newFakeBuilderImage(t, tmpDir, "example.com/old-lifecycle-builder:tag", defaultBuilderStackID, defaultRunImageName, "0.18.0", newLinuxImage)
						defer oldLifecycleBuilder.Cleanup()
						fakeImageFetcher.LocalImages[oldLifecycleBuilder.Name()] = oldLifecycleBuilder

						h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
							Image:        "some/app",
							Builder:      oldLifecycleBuilder.Name(),
							TrustBuilder: func(string) bool { return true },
						}))

						h.AssertEq(t, fakeLifecycle.Opts.UseCreatorWithExtensions, false)
					})
				})

				when("lifecycle is new", func() {
					it("true", func() {
						newLifecycleBuilder := newFakeBuilderImage(t, tmpDir, "example.com/new-lifecycle-builder:tag", defaultBuilderStackID, defaultRunImageName, "0.19.0", newLinuxImage)
						defer newLifecycleBuilder.Cleanup()
						fakeImageFetcher.LocalImages[newLifecycleBuilder.Name()] = newLifecycleBuilder

						h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
							Image:        "some/app",
							Builder:      newLifecycleBuilder.Name(),
							TrustBuilder: func(string) bool { return true },
						}))

						h.AssertEq(t, fakeLifecycle.Opts.UseCreatorWithExtensions, true)
					})
				})
			})
		})

		when("validating mixins", func() {
			when("stack image mixins disagree", func() {
				it.Before(func() {
					h.AssertNil(t, defaultBuilderImage.SetLabel("io.buildpacks.stack.mixins", `["mixinA"]`))
					h.AssertNil(t, fakeDefaultRunImage.SetLabel("io.buildpacks.stack.mixins", `["mixinB"]`))
				})

				it("succeeds", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
					}))
				})

				when("platform API < 0.12", func() {
					it.Before(func() {
						setAPIs(t, defaultBuilderImage, []string{"0.8"}, []string{"0.11"})
					})

					it("returns an error", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultBuilderName,
						})

						h.AssertError(t, err, "validating stack mixins: 'default/run' missing required mixin(s): mixinA")
					})
				})
			})

			when("builder buildpack mixins are not satisfied", func() {
				it.Before(func() {
					h.AssertNil(t, defaultBuilderImage.SetLabel("io.buildpacks.stack.mixins", ""))
					h.AssertNil(t, fakeDefaultRunImage.SetLabel("io.buildpacks.stack.mixins", ""))
				})

				it("succeeds", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
					}))
				})

				when("platform API < 0.12", func() {
					it.Before(func() {
						setAPIs(t, defaultBuilderImage, []string{"0.8"}, []string{"0.11"})
					})

					it("returns an error", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultBuilderName,
						})

						h.AssertError(t, err, "validating stack mixins: buildpack 'buildpack.1.id@buildpack.1.version' requires missing mixin(s): build:mixinY, mixinX, run:mixinZ")
					})
				})
			})
		})

		when("Volumes option", func() {
			when("on posix", func() {
				it.Before(func() {
					h.SkipIf(t, runtime.GOOS == "windows", "Skipped on windows")
				})

				for _, test := range []struct {
					name        string
					volume      string
					expectation string
				}{
					{"defaults to read-only", "/a:/x", "/a:/x:ro"},
					{"defaults to read-only (nested)", "/a:/some/path/y", "/a:/some/path/y:ro"},
					{"supports rw mode", "/a:/x:rw", "/a:/x:rw"},
				} {
					volume := test.volume
					expectation := test.expectation

					it(test.name, func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultBuilderName,
							ContainerConfig: ContainerConfig{
								Volumes: []string{volume},
							},
						})
						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Volumes, []string{expectation})
					})
				}

				when("volume mode is invalid", func() {
					it("returns an error", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultBuilderName,
							ContainerConfig: ContainerConfig{
								Volumes: []string{"/a:/x:invalid"},
							},
						})
						h.AssertError(t, err, `platform volume "/a:/x:invalid" has invalid format: invalid mode: invalid`)
					})
				})

				when("volume specification is invalid", func() {
					it("returns an error", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultBuilderName,
							ContainerConfig: ContainerConfig{
								Volumes: []string{":::"},
							},
						})
						if runtime.GOOS == "darwin" {
							h.AssertError(t, err, `platform volume ":::" has invalid format: invalid spec: :::: empty section between colons`)
						} else {
							h.AssertError(t, err, `platform volume ":::" has invalid format: invalid volume specification: ':::'`)
						}
					})
				})

				when("mounting onto cnb spec'd dir", func() {
					for _, p := range []string{
						"/cnb/buildpacks",
						"/cnb/buildpacks/nested",
						"/cnb",
						"/cnb/nested",
						"/layers",
						"/layers/nested",
						"/workspace",
						"/workspace/bindings",
					} {
						p := p
						it(fmt.Sprintf("warns when mounting to '%s'", p), func() {
							err := subject.Build(context.TODO(), BuildOptions{
								Image:   "some/app",
								Builder: defaultBuilderName,
								ContainerConfig: ContainerConfig{
									Volumes: []string{fmt.Sprintf("/tmp/path:%s", p)},
								},
							})

							h.AssertNil(t, err)
							h.AssertContains(t, outBuf.String(), fmt.Sprintf("Warning: Mounting to a sensitive directory '%s'", p))
						})
					}
				})
			})

			when("on windows", func() {
				it.Before(func() {
					h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")
				})
				when("linux container", func() {
					it("drive is transformed", func() {
						dir, _ := os.MkdirTemp("", "pack-test-mount")
						volume := fmt.Sprintf("%v:/x", dir)
						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultBuilderName,
							ContainerConfig: ContainerConfig{
								Volumes: []string{volume},
							},
							TrustBuilder: func(string) bool { return true },
						})
						expected := []string{
							fmt.Sprintf("%s:/x:ro", strings.ToLower(dir)),
						}
						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Volumes, expected)
					})

					// May not fail as mode is not used on Windows
					when("volume mode is invalid", func() {
						it("returns an error", func() {
							err := subject.Build(context.TODO(), BuildOptions{
								Image:   "some/app",
								Builder: defaultBuilderName,
								ContainerConfig: ContainerConfig{
									Volumes: []string{"/a:/x:invalid"},
								},
								TrustBuilder: func(string) bool { return true },
							})
							h.AssertError(t, err, `platform volume "/a:/x:invalid" has invalid format: invalid volume specification: '/a:/x:invalid'`)
						})
					})

					when("volume specification is invalid", func() {
						it("returns an error", func() {
							err := subject.Build(context.TODO(), BuildOptions{
								Image:   "some/app",
								Builder: defaultBuilderName,
								ContainerConfig: ContainerConfig{
									Volumes: []string{":::"},
								},
								TrustBuilder: func(string) bool { return true },
							})
							h.AssertError(t, err, `platform volume ":::" has invalid format: invalid volume specification: ':::'`)
						})
					})

					when("mounting onto cnb spec'd dir", func() {
						for _, p := range []string{
							`/cnb`, `/cnb/buildpacks`, `/layers`, `/workspace`,
						} {
							p := p
							it(fmt.Sprintf("warns when mounting to '%s'", p), func() {
								err := subject.Build(context.TODO(), BuildOptions{
									Image:   "some/app",
									Builder: defaultBuilderName,
									ContainerConfig: ContainerConfig{
										Volumes: []string{fmt.Sprintf("c:/Users:%s", p)},
									},
									TrustBuilder: func(string) bool { return true },
								})

								h.AssertNil(t, err)
								h.AssertContains(t, outBuf.String(), fmt.Sprintf("Warning: Mounting to a sensitive directory '%s'", p))
							})
						}
					})
				})
				when("windows container", func() {
					it("drive is mounted", func() {
						dir, _ := os.MkdirTemp("", "pack-test-mount")
						volume := fmt.Sprintf("%v:c:\\x", dir)
						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultWindowsBuilderName,
							ContainerConfig: ContainerConfig{
								Volumes: []string{volume},
							},
							TrustBuilder: func(string) bool { return true },
						})
						expected := []string{
							fmt.Sprintf("%s:c:\\x:ro", strings.ToLower(dir)),
						}
						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.Volumes, expected)
					})

					// May not fail as mode is not used on Windows
					when("volume mode is invalid", func() {
						it("returns an error", func() {
							err := subject.Build(context.TODO(), BuildOptions{
								Image:   "some/app",
								Builder: defaultWindowsBuilderName,
								ContainerConfig: ContainerConfig{
									Volumes: []string{"/a:/x:invalid"},
								},
								TrustBuilder: func(string) bool { return true },
							})
							h.AssertError(t, err, `platform volume "/a:/x:invalid" has invalid format: invalid volume specification: '/a:/x:invalid'`)
						})
					})

					// Should fail even on windows
					when("volume specification is invalid", func() {
						it("returns an error", func() {
							err := subject.Build(context.TODO(), BuildOptions{
								Image:   "some/app",
								Builder: defaultWindowsBuilderName,
								ContainerConfig: ContainerConfig{
									Volumes: []string{":::"},
								},
								TrustBuilder: func(string) bool { return true },
							})
							h.AssertError(t, err, `platform volume ":::" has invalid format: invalid volume specification: ':::'`)
						})
					})

					when("mounting onto cnb spec'd dir", func() {
						for _, p := range []string{
							`c:\cnb`, `c:\cnb\buildpacks`, `c:\layers`, `c:\workspace`,
						} {
							p := p
							it(fmt.Sprintf("warns when mounting to '%s'", p), func() {
								err := subject.Build(context.TODO(), BuildOptions{
									Image:   "some/app",
									Builder: defaultWindowsBuilderName,
									ContainerConfig: ContainerConfig{
										Volumes: []string{fmt.Sprintf("c:/Users:%s", p)},
									},
									TrustBuilder: func(string) bool { return true },
								})

								h.AssertNil(t, err)
								h.AssertContains(t, outBuf.String(), fmt.Sprintf("Warning: Mounting to a sensitive directory '%s'", p))
							})
						}
					})
				})
			})
		})

		when("gid option", func() {
			it("gid is passthroughs to lifecycle", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Workspace: "app",
					Builder:   defaultBuilderName,
					Image:     "example.com/some/repo:tag",
					GroupID:   2,
				}))
				h.AssertEq(t, fakeLifecycle.Opts.GID, 2)
			})
		})

		when("RegistryMirrors option", func() {
			it("translates run image before passing to lifecycle", func() {
				subject.registryMirrors = map[string]string{
					"index.docker.io": "10.0.0.1",
				}

				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Builder: defaultBuilderName,
					Image:   "example.com/some/repo:tag",
				}))
				h.AssertEq(t, fakeLifecycle.Opts.RunImage, "10.0.0.1/default/run:latest")
			})
		})

		when("previous-image option", func() {
			it("previous-image is passed to lifecycle", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Workspace:     "app",
					Builder:       defaultBuilderName,
					Image:         "example.com/some/repo:tag",
					PreviousImage: "example.com/some/new:tag",
				}))
				h.AssertEq(t, fakeLifecycle.Opts.PreviousImage, "example.com/some/new:tag")
			})
		})

		when("interactive option", func() {
			it("passthroughs to lifecycle", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Builder:     defaultBuilderName,
					Image:       "example.com/some/repo:tag",
					Interactive: true,
				}))
				h.AssertEq(t, fakeLifecycle.Opts.Interactive, true)
			})
		})

		when("sbom destination dir option", func() {
			it("passthroughs to lifecycle", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Builder:            defaultBuilderName,
					Image:              "example.com/some/repo:tag",
					SBOMDestinationDir: "some-destination-dir",
				}))
				h.AssertEq(t, fakeLifecycle.Opts.SBOMDestinationDir, "some-destination-dir")
			})
		})

		when("report destination dir option", func() {
			it("passthroughs to lifecycle", func() {
				h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
					Builder:              defaultBuilderName,
					Image:                "example.com/some/repo:tag",
					ReportDestinationDir: "a-destination-dir",
				}))
				h.AssertEq(t, fakeLifecycle.Opts.ReportDestinationDir, "a-destination-dir")
			})
		})

		when("there are extensions", func() {
			withExtensionsLabel = true

			when("default configuration", func() {
				it("succeeds", func() {
					err := subject.Build(context.TODO(), BuildOptions{
						Image:   "some/app",
						Builder: defaultBuilderName,
					})

					h.AssertNil(t, err)
					h.AssertEq(t, fakeLifecycle.Opts.BuilderImage, defaultBuilderName)
				})
			})

			when("os", func() {
				when("windows", func() {
					it.Before(func() {
						h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")
					})

					it("errors", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultWindowsBuilderName,
						})

						h.AssertNotNil(t, err)
					})
				})

				when("linux", func() {
					it("succeeds", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:   "some/app",
							Builder: defaultBuilderName,
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.BuilderImage, defaultBuilderName)
					})
				})
			})

			when("pull policy", func() {
				when("always", func() {
					it("succeeds", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							PullPolicy: image.PullAlways,
						})

						h.AssertNil(t, err)
						h.AssertEq(t, fakeLifecycle.Opts.BuilderImage, defaultBuilderName)
					})
				})

				when("other", func() {
					it("errors", func() {
						err := subject.Build(context.TODO(), BuildOptions{
							Image:      "some/app",
							Builder:    defaultBuilderName,
							PullPolicy: image.PullNever,
						})

						h.AssertNotNil(t, err)
					})
				})
			})
		})

		when("export to OCI layout", func() {
			var (
				inputImageReference, inputPreviousImageReference       InputImageReference
				layoutConfig                                           *LayoutConfig
				hostImagePath, hostPreviousImagePath, hostRunImagePath string
			)

			it.Before(func() {
				h.SkipIf(t, runtime.GOOS == "windows", "skip on windows")

				remoteRunImage := fakes.NewImage("default/run", "", nil)
				h.AssertNil(t, remoteRunImage.SetLabel("io.buildpacks.stack.id", defaultBuilderStackID))
				h.AssertNil(t, remoteRunImage.SetLabel("io.buildpacks.stack.mixins", `["mixinA", "mixinX", "run:mixinZ"]`))
				fakeImageFetcher.RemoteImages[remoteRunImage.Name()] = remoteRunImage

				hostImagePath = filepath.Join(tmpDir, "my-app")
				inputImageReference = ParseInputImageReference(fmt.Sprintf("oci:%s", hostImagePath))
				layoutConfig = &LayoutConfig{
					InputImage:    inputImageReference,
					LayoutRepoDir: filepath.Join(tmpDir, "local-repo"),
				}
			})

			when("previous image is not provided", func() {
				when("sparse is false", func() {
					it("saves run-image locally in oci layout and mount volumes", func() {
						h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
							Image:        inputImageReference.Name(),
							Builder:      defaultBuilderName,
							LayoutConfig: layoutConfig,
						}))

						args := fakeImageFetcher.FetchCalls["default/run"]
						h.AssertEq(t, args.LayoutOption.Sparse, false)
						h.AssertContains(t, args.LayoutOption.Path, layoutConfig.LayoutRepoDir)

						h.AssertEq(t, fakeLifecycle.Opts.Layout, true)
						// verify the host path are mounted as volumes
						h.AssertSliceContainsMatch(t, fakeLifecycle.Opts.Volumes, hostImagePath, hostRunImagePath)
					})
				})

				when("sparse is true", func() {
					it.Before(func() {
						layoutConfig.Sparse = true
					})

					it("saves run-image locally (no layers) in oci layout and mount volumes", func() {
						h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
							Image:        inputImageReference.Name(),
							Builder:      defaultBuilderName,
							LayoutConfig: layoutConfig,
						}))

						args := fakeImageFetcher.FetchCalls["default/run"]
						h.AssertEq(t, args.LayoutOption.Sparse, true)
						h.AssertContains(t, args.LayoutOption.Path, layoutConfig.LayoutRepoDir)

						h.AssertEq(t, fakeLifecycle.Opts.Layout, true)
						// verify the host path are mounted as volumes
						h.AssertSliceContainsMatch(t, fakeLifecycle.Opts.Volumes, hostImagePath, hostRunImagePath)
					})
				})
			})

			when("previous image is provided", func() {
				it.Before(func() {
					hostPreviousImagePath = filepath.Join(tmpDir, "my-previous-app")
					inputPreviousImageReference = ParseInputImageReference(fmt.Sprintf("oci:%s", hostPreviousImagePath))
					layoutConfig.PreviousInputImage = inputPreviousImageReference
				})

				it("mount previous image volume", func() {
					h.AssertNil(t, subject.Build(context.TODO(), BuildOptions{
						Image:         inputImageReference.Name(),
						PreviousImage: inputPreviousImageReference.Name(),
						Builder:       defaultBuilderName,
						LayoutConfig:  layoutConfig,
					}))

					h.AssertEq(t, fakeLifecycle.Opts.Layout, true)
					// verify the host path are mounted as volumes
					h.AssertSliceContainsMatch(t, fakeLifecycle.Opts.Volumes, hostImagePath, hostPreviousImagePath, hostRunImagePath)
				})
			})
		})
	})
}

func makeFakePackage(t *testing.T, tmpDir string, stackID string) *fakes.Image {
	metaBuildpackTar := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
		WithAPI: api.MustParse("0.3"),
		WithInfo: dist.ModuleInfo{
			ID:       "meta.buildpack.id",
			Version:  "meta.buildpack.version",
			Homepage: "http://meta.buildpack",
		},
		WithStacks: nil,
		WithOrder: dist.Order{{
			Group: []dist.ModuleRef{{
				ModuleInfo: dist.ModuleInfo{
					ID:      "child.buildpack.id",
					Version: "child.buildpack.version",
				},
				Optional: false,
			}},
		}},
	})

	childBuildpackTar := ifakes.CreateBuildpackTar(t, tmpDir, dist.BuildpackDescriptor{
		WithAPI: api.MustParse("0.3"),
		WithInfo: dist.ModuleInfo{
			ID:       "child.buildpack.id",
			Version:  "child.buildpack.version",
			Homepage: "http://child.buildpack",
		},
		WithStacks: []dist.Stack{
			{ID: stackID},
		},
	})

	bpLayers := dist.ModuleLayers{
		"meta.buildpack.id": {
			"meta.buildpack.version": {
				API: api.MustParse("0.3"),
				Order: dist.Order{{
					Group: []dist.ModuleRef{{
						ModuleInfo: dist.ModuleInfo{
							ID:      "child.buildpack.id",
							Version: "child.buildpack.version",
						},
						Optional: false,
					}},
				}},
				LayerDiffID: diffIDForFile(t, metaBuildpackTar),
			},
		},
		"child.buildpack.id": {
			"child.buildpack.version": {
				API: api.MustParse("0.3"),
				Stacks: []dist.Stack{
					{ID: stackID},
				},
				LayerDiffID: diffIDForFile(t, childBuildpackTar),
			},
		},
	}

	md := buildpack.Metadata{
		ModuleInfo: dist.ModuleInfo{
			ID:      "meta.buildpack.id",
			Version: "meta.buildpack.version",
		},
		Stacks: []dist.Stack{
			{ID: stackID},
		},
	}

	fakePackage := fakes.NewImage("example.com/some/package", "", nil)
	h.AssertNil(t, dist.SetLabel(fakePackage, "io.buildpacks.buildpack.layers", bpLayers))
	h.AssertNil(t, dist.SetLabel(fakePackage, "io.buildpacks.buildpackage.metadata", md))

	h.AssertNil(t, fakePackage.AddLayer(metaBuildpackTar))
	h.AssertNil(t, fakePackage.AddLayer(childBuildpackTar))

	return fakePackage
}

func diffIDForFile(t *testing.T, path string) string {
	file, err := os.Open(path)
	h.AssertNil(t, err)

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	h.AssertNil(t, err)

	return "sha256:" + hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size())))
}

func newLinuxImage(name, topLayerSha string, identifier imgutil.Identifier) *fakes.Image {
	return fakes.NewImage(name, topLayerSha, identifier)
}

func newWindowsImage(name, topLayerSha string, identifier imgutil.Identifier) *fakes.Image {
	result := fakes.NewImage(name, topLayerSha, identifier)
	arch, _ := result.Architecture()
	osVersion, _ := result.OSVersion()
	result.SetOS("windows")
	result.SetOSVersion(osVersion)
	result.SetArchitecture(arch)
	return result
}

func newFakeBuilderImage(t *testing.T, tmpDir, builderName, defaultBuilderStackID, runImageName, lifecycleVersion string, osImageCreator ifakes.FakeImageCreator) *fakes.Image {
	var supportedBuildpackAPIs builder.APISet
	for _, v := range api.Buildpack.Supported {
		supportedBuildpackAPIs = append(supportedBuildpackAPIs, v)
	}
	var supportedPlatformAPIs builder.APISet
	for _, v := range api.Platform.Supported {
		supportedPlatformAPIs = append(supportedPlatformAPIs, v)
	}
	return ifakes.NewFakeBuilderImage(t,
		tmpDir,
		builderName,
		defaultBuilderStackID,
		"1234",
		"5678",
		builder.Metadata{
			Buildpacks: []dist.ModuleInfo{
				{ID: "buildpack.1.id", Version: "buildpack.1.version"},
				{ID: "buildpack.2.id", Version: "buildpack.2.version"},
			},
			Extensions: []dist.ModuleInfo{
				{ID: "extension.1.id", Version: "extension.1.version"},
				{ID: "extension.2.id", Version: "extension.2.version"},
			},
			Stack: builder.StackMetadata{
				RunImage: builder.RunImageMetadata{
					Image: runImageName,
					Mirrors: []string{
						"registry1.example.com/run/mirror",
						"registry2.example.com/run/mirror",
					},
				},
			},
			Lifecycle: builder.LifecycleMetadata{
				LifecycleInfo: builder.LifecycleInfo{
					Version: &builder.Version{
						Version: *semver.MustParse(lifecycleVersion),
					},
				},
				APIs: builder.LifecycleAPIs{
					Buildpack: builder.APIVersions{
						Supported: supportedBuildpackAPIs,
					},
					Platform: builder.APIVersions{
						Supported: supportedPlatformAPIs,
					},
				},
			},
		},
		dist.ModuleLayers{
			"buildpack.1.id": {
				"buildpack.1.version": {
					API: api.MustParse("0.3"),
					Stacks: []dist.Stack{
						{
							ID:     defaultBuilderStackID,
							Mixins: []string{"mixinX", "build:mixinY", "run:mixinZ"},
						},
					},
				},
			},
			"buildpack.2.id": {
				"buildpack.2.version": {
					API: api.MustParse("0.3"),
					Stacks: []dist.Stack{
						{
							ID:     defaultBuilderStackID,
							Mixins: []string{"mixinX", "build:mixinY"},
						},
					},
				},
			},
		},
		dist.Order{{
			Group: []dist.ModuleRef{{
				ModuleInfo: dist.ModuleInfo{
					ID:      "buildpack.1.id",
					Version: "buildpack.1.version",
				},
			}},
		}, {
			Group: []dist.ModuleRef{{
				ModuleInfo: dist.ModuleInfo{
					ID:      "buildpack.2.id",
					Version: "buildpack.2.version",
				},
			}},
		}},
		dist.ModuleLayers{
			"extension.1.id": {
				"extension.1.version": {
					API: api.MustParse("0.3"),
				},
			},
			"extension.2.id": {
				"extension.2.version": {
					API: api.MustParse("0.3"),
				},
			},
		},
		dist.Order{{
			Group: []dist.ModuleRef{{
				ModuleInfo: dist.ModuleInfo{
					ID:      "extension.1.id",
					Version: "extension.1.version",
				},
			}},
		}, {
			Group: []dist.ModuleRef{{
				ModuleInfo: dist.ModuleInfo{
					ID:      "extension.2.id",
					Version: "extension.2.version",
				},
			}},
		}},
		osImageCreator,
	)
}

func setAPIs(t *testing.T, image *fakes.Image, buildpackAPIs []string, platformAPIs []string) {
	builderMDLabelName := "io.buildpacks.builder.metadata"
	var supportedBuildpackAPIs builder.APISet
	for _, v := range buildpackAPIs {
		supportedBuildpackAPIs = append(supportedBuildpackAPIs, api.MustParse(v))
	}
	var supportedPlatformAPIs builder.APISet
	for _, v := range platformAPIs {
		supportedPlatformAPIs = append(supportedPlatformAPIs, api.MustParse(v))
	}
	builderMDLabel, err := image.Label(builderMDLabelName)
	h.AssertNil(t, err)
	var builderMD builder.Metadata
	h.AssertNil(t, json.Unmarshal([]byte(builderMDLabel), &builderMD))
	builderMD.Lifecycle.APIs = builder.LifecycleAPIs{
		Buildpack: builder.APIVersions{
			Supported: supportedBuildpackAPIs,
		},
		Platform: builder.APIVersions{
			Supported: supportedPlatformAPIs,
		},
	}
	builderMDLabelBytes, err := json.Marshal(&builderMD)
	h.AssertNil(t, err)
	h.AssertNil(t, image.SetLabel(builderMDLabelName, string(builderMDLabelBytes)))
}
