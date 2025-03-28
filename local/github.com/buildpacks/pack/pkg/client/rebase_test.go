package client

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/buildpacks/lifecycle/auth"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	ifakes "github.com/buildpacks/pack/internal/fakes"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestRebase(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "rebase_factory", testRebase, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testRebase(t *testing.T, when spec.G, it spec.S) {
	when("#Rebase", func() {
		var (
			fakeImageFetcher   *ifakes.FakeImageFetcher
			subject            *Client
			fakeAppImage       *fakes.Image
			fakeRunImage       *fakes.Image
			fakeRunImageMirror *fakes.Image
			out                bytes.Buffer
		)

		it.Before(func() {
			fakeImageFetcher = ifakes.NewFakeImageFetcher()

			fakeAppImage = fakes.NewImage("some/app", "", &fakeIdentifier{name: "app-image"})
			h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.lifecycle.metadata",
				`{"stack":{"runImage":{"image":"some/run", "mirrors":["example.com/some/run"]}}}`))
			h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy"))
			fakeImageFetcher.LocalImages["some/app"] = fakeAppImage

			fakeRunImage = fakes.NewImage("some/run", "run-image-top-layer-sha", &fakeIdentifier{name: "run-image-digest"})
			h.AssertNil(t, fakeRunImage.SetLabel("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy"))
			fakeImageFetcher.LocalImages["some/run"] = fakeRunImage

			fakeRunImageMirror = fakes.NewImage("example.com/some/run", "mirror-top-layer-sha", &fakeIdentifier{name: "mirror-digest"})
			h.AssertNil(t, fakeRunImageMirror.SetLabel("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy"))
			fakeImageFetcher.LocalImages["example.com/some/run"] = fakeRunImageMirror

			keychain, err := auth.DefaultKeychain("pack-test/dummy")
			h.AssertNil(t, err)

			fakeLogger := logging.NewLogWithWriters(&out, &out)
			subject = &Client{
				logger:       fakeLogger,
				imageFetcher: fakeImageFetcher,
				keychain:     keychain,
			}
		})

		it.After(func() {
			h.AssertNilE(t, fakeAppImage.Cleanup())
			h.AssertNilE(t, fakeRunImage.Cleanup())
			h.AssertNilE(t, fakeRunImageMirror.Cleanup())
		})

		when("#Rebase", func() {
			when("run image is provided by the user", func() {
				when("the image has a label with a run image specified", func() {
					var fakeCustomRunImage *fakes.Image

					it.Before(func() {
						fakeCustomRunImage = fakes.NewImage("custom/run", "custom-base-top-layer-sha", &fakeIdentifier{name: "custom-base-digest"})
						h.AssertNil(t, fakeCustomRunImage.SetLabel("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy"))
						fakeImageFetcher.LocalImages["custom/run"] = fakeCustomRunImage
					})

					it.After(func() {
						h.AssertNilE(t, fakeCustomRunImage.Cleanup())
					})

					when("--force", func() {
						it("uses the run image provided by the user", func() {
							h.AssertNil(t, subject.Rebase(context.TODO(),
								RebaseOptions{
									RunImage: "custom/run",
									RepoName: "some/app",
									Force:    true,
								}))
							h.AssertEq(t, fakeAppImage.Base(), "custom/run")
							lbl, _ := fakeAppImage.Label("io.buildpacks.lifecycle.metadata")
							h.AssertContains(t, lbl, `"runImage":{"topLayer":"custom-base-top-layer-sha","reference":"custom-base-digest"`)
						})
					})

					it("errors", func() {
						h.AssertError(t, subject.Rebase(context.TODO(),
							RebaseOptions{
								RunImage: "custom/run",
								RepoName: "some/app",
							}), "new base image 'custom/run' not found in existing run image metadata")
					})
				})
			})

			when("run image is NOT provided by the user", func() {
				when("the image has a label with a run image specified", func() {
					it("uses the run image provided in the App image label", func() {
						h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
							RepoName: "some/app",
						}))
						h.AssertEq(t, fakeAppImage.Base(), "some/run")
						lbl, _ := fakeAppImage.Label("io.buildpacks.lifecycle.metadata")
						h.AssertContains(t, lbl, `"runImage":{"topLayer":"run-image-top-layer-sha","reference":"run-image-digest"`)
					})
				})

				when("the image has a label with a run image mirrors specified", func() {
					when("there are no user provided mirrors", func() {
						it.Before(func() {
							fakeImageFetcher.LocalImages["example.com/some/app"] = fakeAppImage
						})

						it("chooses a matching mirror from the app image label", func() {
							h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
								RepoName: "example.com/some/app",
							}))
							h.AssertEq(t, fakeAppImage.Base(), "example.com/some/run")
							lbl, _ := fakeAppImage.Label("io.buildpacks.lifecycle.metadata")
							h.AssertContains(t, lbl, `"runImage":{"topLayer":"mirror-top-layer-sha","reference":"mirror-digest"`)
						})
					})

					when("there are user provided mirrors", func() {
						var (
							fakeLocalMirror *fakes.Image
						)
						it.Before(func() {
							fakeImageFetcher.LocalImages["example.com/some/app"] = fakeAppImage
							fakeLocalMirror = fakes.NewImage("example.com/some/local-run", "local-mirror-top-layer-sha", &fakeIdentifier{name: "local-mirror-digest"})
							h.AssertNil(t, fakeLocalMirror.SetLabel("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy"))
							fakeImageFetcher.LocalImages["example.com/some/local-run"] = fakeLocalMirror
						})

						it.After(func() {
							h.AssertNilE(t, fakeLocalMirror.Cleanup())
						})
						when("--force", func() {
							it("chooses a matching local mirror first", func() {
								h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
									RepoName: "example.com/some/app",
									AdditionalMirrors: map[string][]string{
										"some/run": {"example.com/some/local-run"},
									},
									Force: true,
								}))
								h.AssertEq(t, fakeAppImage.Base(), "example.com/some/local-run")
								lbl, _ := fakeAppImage.Label("io.buildpacks.lifecycle.metadata")
								h.AssertContains(t, lbl, `"runImage":{"topLayer":"local-mirror-top-layer-sha","reference":"local-mirror-digest"`)
							})
						})
					})
					when("there is a label and it has a run image and no stack", func() {
						it("reads the run image from the label", func() {
							h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.lifecycle.metadata",
								`{"runImage":{"image":"some/run", "mirrors":["example.com/some/run"]}}`))
							h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
								RepoName: "some/app",
							}))
							h.AssertEq(t, fakeAppImage.Base(), "some/run")
						})
					})
					when("there is neither runImage nor stack", func() {
						it("fails gracefully", func() {
							h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.lifecycle.metadata", `{}`))
							h.AssertError(t, subject.Rebase(context.TODO(), RebaseOptions{RepoName: "some/app"}),
								"run image must be specified")
						})
					})
				})

				when("the image does not have a label with a run image specified", func() {
					it("returns an error", func() {
						h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.lifecycle.metadata", "{}"))
						err := subject.Rebase(context.TODO(), RebaseOptions{
							RepoName: "some/app",
						})
						h.AssertError(t, err, "run image must be specified")
					})
				})
			})

			when("publish", func() {
				var (
					fakeRemoteRunImage *fakes.Image
				)

				it.Before(func() {
					fakeRemoteRunImage = fakes.NewImage("some/run", "remote-top-layer-sha", &fakeIdentifier{name: "remote-digest"})
					h.AssertNil(t, fakeRemoteRunImage.SetLabel("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy"))
					fakeImageFetcher.RemoteImages["some/run"] = fakeRemoteRunImage
				})

				it.After(func() {
					h.AssertNilE(t, fakeRemoteRunImage.Cleanup())
				})

				when("is false", func() {
					when("pull policy is always", func() {
						it("updates the local image", func() {
							h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
								RepoName:   "some/app",
								PullPolicy: image.PullAlways,
							}))
							h.AssertEq(t, fakeAppImage.Base(), "some/run")
							lbl, _ := fakeAppImage.Label("io.buildpacks.lifecycle.metadata")
							h.AssertContains(t, lbl, `"runImage":{"topLayer":"remote-top-layer-sha","reference":"remote-digest"`)
						})
					})

					when("pull policy is never", func() {
						it("uses local image", func() {
							h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
								RepoName:   "some/app",
								PullPolicy: image.PullNever,
							}))
							h.AssertEq(t, fakeAppImage.Base(), "some/run")
							lbl, _ := fakeAppImage.Label("io.buildpacks.lifecycle.metadata")
							h.AssertContains(t, lbl, `"runImage":{"topLayer":"run-image-top-layer-sha","reference":"run-image-digest"`)
						})
					})
				})

				when("report directory is set", func() {
					it("writes the report", func() {
						tmpdir := t.TempDir()
						h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
							RepoName:             "some/app",
							ReportDestinationDir: tmpdir,
						}))
						_, err := os.Stat(filepath.Join(tmpdir, "report.toml"))
						h.AssertNil(t, err)
					})
				})

				when("is true", func() {
					it.Before(func() {
						fakeImageFetcher.RemoteImages["some/app"] = fakeAppImage
					})

					when("skip pull is anything", func() {
						it("uses remote image", func() {
							h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
								RepoName: "some/app",
								Publish:  true,
							}))
							h.AssertEq(t, fakeAppImage.Base(), "some/run")
							lbl, _ := fakeAppImage.Label("io.buildpacks.lifecycle.metadata")
							h.AssertContains(t, lbl, `"runImage":{"topLayer":"remote-top-layer-sha","reference":"remote-digest"`)
							args := fakeImageFetcher.FetchCalls["some/run"]
							h.AssertEq(t, args.Target.ValuesAsPlatform(), "linux/amd64")
						})
					})
				})
			})
			when("previous image is provided", func() {
				it("fetches the image using the previous image name", func() {
					h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
						RepoName:      "new/app",
						PreviousImage: "some/app",
					}))
					args := fakeImageFetcher.FetchCalls["some/app"]
					h.AssertNotNil(t, args)
					h.AssertEq(t, args.Daemon, true)
				})
			})

			when("previous image is set to new image name", func() {
				it("returns error if Fetch function fails", func() {
					err := subject.Rebase(context.TODO(), RebaseOptions{
						RepoName:      "some/app",
						PreviousImage: "new/app",
					})
					h.AssertError(t, err, "image 'new/app' does not exist on the daemon: not found")
				})
			})

			when("previous image is not provided", func() {
				it("fetches the image using the repo name", func() {
					h.AssertNil(t, subject.Rebase(context.TODO(), RebaseOptions{
						RepoName: "some/app",
					}))
					args := fakeImageFetcher.FetchCalls["some/app"]
					h.AssertNotNil(t, args)
					h.AssertEq(t, args.Daemon, true)
				})
			})
		})
	})
}

type fakeIdentifier struct {
	name string
}

func (f *fakeIdentifier) String() string {
	return f.name
}
