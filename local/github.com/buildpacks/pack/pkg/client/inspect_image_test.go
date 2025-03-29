package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestInspectImage(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "InspectImage", testInspectImage, spec.Parallel(), spec.Report(report.Terminal{}))
}

// PlatformAPI should be ignored because it is not set in the metadata label
var ignorePlatformAPI = []cmp.Option{
	cmpopts.IgnoreFields(launch.Process{}, "PlatformAPI"),
	cmpopts.IgnoreFields(launch.RawCommand{}, "PlatformAPI"),
}

func testInspectImage(t *testing.T, when spec.G, it spec.S) {
	var (
		subject                        *Client
		mockImageFetcher               *testmocks.MockImageFetcher
		mockDockerClient               *testmocks.MockCommonAPIClient
		mockController                 *gomock.Controller
		mockImage                      *testmocks.MockImage
		mockImageNoRebasable           *testmocks.MockImage
		mockImageRebasableWithoutLabel *testmocks.MockImage
		mockImageWithExtension         *testmocks.MockImage
		out                            bytes.Buffer
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockImageFetcher = testmocks.NewMockImageFetcher(mockController)
		mockDockerClient = testmocks.NewMockCommonAPIClient(mockController)

		var err error
		subject, err = NewClient(WithLogger(logging.NewLogWithWriters(&out, &out)), WithFetcher(mockImageFetcher), WithDockerClient(mockDockerClient))
		h.AssertNil(t, err)

		mockImage = testmocks.NewImage("some/image", "", nil)
		h.AssertNil(t, mockImage.SetWorkingDir("/test-workdir"))
		h.AssertNil(t, mockImage.SetLabel("io.buildpacks.stack.id", "test.stack.id"))
		h.AssertNil(t, mockImage.SetLabel("io.buildpacks.rebasable", "true"))
		h.AssertNil(t, mockImage.SetLabel(
			"io.buildpacks.lifecycle.metadata",
			`{
  "stack": {
    "runImage": {
      "image": "some-run-image",
      "mirrors": [
        "some-mirror",
        "other-mirror"
      ]
    }
  },
  "runImage": {
    "topLayer": "some-top-layer",
    "reference": "some-run-image-reference"
  }
}`,
		))
		h.AssertNil(t, mockImage.SetLabel(
			"io.buildpacks.build.metadata",
			`{
  "bom": [
    {
      "name": "some-bom-element"
    }
  ],
  "buildpacks": [
    {
      "id": "some-buildpack",
      "version": "some-version"
    },
    {
      "id": "other-buildpack",
      "version": "other-version"
    }
  ],
  "processes": [
    {
      "type": "other-process",
      "command": "/other/process",
      "args": ["opt", "1"],
      "direct": true
    },
    {
      "type": "web",
      "command": "/start/web-process",
      "args": ["-p", "1234"],
      "direct": false
    }
  ],
  "launcher": {
    "version": "0.5.0"
  }
}`,
		))

		mockImageNoRebasable = testmocks.NewImage("some/imageNoRebasable", "", nil)
		h.AssertNil(t, mockImageNoRebasable.SetWorkingDir("/test-workdir"))
		h.AssertNil(t, mockImageNoRebasable.SetLabel("io.buildpacks.stack.id", "test.stack.id"))
		h.AssertNil(t, mockImageNoRebasable.SetLabel("io.buildpacks.rebasable", "false"))
		h.AssertNil(t, mockImageNoRebasable.SetLabel(
			"io.buildpacks.lifecycle.metadata",
			`{
  "stack": {
    "runImage": {
      "image": "some-run-image-no-rebasable",
      "mirrors": [
        "some-mirror",
        "other-mirror"
      ]
    }
  },
  "runImage": {
    "topLayer": "some-top-layer",
    "reference": "some-run-image-reference"
  }
}`,
		))
		h.AssertNil(t, mockImageNoRebasable.SetLabel(
			"io.buildpacks.build.metadata",
			`{
  "bom": [
    {
      "name": "some-bom-element"
    }
  ],
  "buildpacks": [
    {
      "id": "some-buildpack",
      "version": "some-version"
    },
    {
      "id": "other-buildpack",
      "version": "other-version"
    }
  ],
  "processes": [
    {
      "type": "other-process",
      "command": "/other/process",
      "args": ["opt", "1"],
      "direct": true
    },
    {
      "type": "web",
      "command": "/start/web-process",
      "args": ["-p", "1234"],
      "direct": false
    }
  ],
  "launcher": {
    "version": "0.5.0"
  }
}`,
		))

		mockImageRebasableWithoutLabel = testmocks.NewImage("some/imageRebasableWithoutLabel", "", nil)
		h.AssertNil(t, mockImageNoRebasable.SetWorkingDir("/test-workdir"))
		h.AssertNil(t, mockImageNoRebasable.SetLabel("io.buildpacks.stack.id", "test.stack.id"))
		h.AssertNil(t, mockImageNoRebasable.SetLabel(
			"io.buildpacks.lifecycle.metadata",
			`{
  "stack": {
    "runImage": {
      "image": "some-run-image-no-rebasable",
      "mirrors": [
        "some-mirror",
        "other-mirror"
      ]
    }
  },
  "runImage": {
    "topLayer": "some-top-layer",
    "reference": "some-run-image-reference"
  }
}`,
		))
		h.AssertNil(t, mockImageNoRebasable.SetLabel(
			"io.buildpacks.build.metadata",
			`{
  "bom": [
    {
      "name": "some-bom-element"
    }
  ],
  "buildpacks": [
    {
      "id": "some-buildpack",
      "version": "some-version"
    },
    {
      "id": "other-buildpack",
      "version": "other-version"
    }
  ],
  "processes": [
    {
      "type": "other-process",
      "command": "/other/process",
      "args": ["opt", "1"],
      "direct": true
    },
    {
      "type": "web",
      "command": "/start/web-process",
      "args": ["-p", "1234"],
      "direct": false
    }
  ],
  "launcher": {
    "version": "0.5.0"
  }
}`,
		))

		mockImageWithExtension = testmocks.NewImage("some/imageWithExtension", "", nil)
		h.AssertNil(t, mockImageWithExtension.SetWorkingDir("/test-workdir"))
		h.AssertNil(t, mockImageWithExtension.SetLabel("io.buildpacks.stack.id", "test.stack.id"))
		h.AssertNil(t, mockImageWithExtension.SetLabel("io.buildpacks.rebasable", "true"))
		h.AssertNil(t, mockImageWithExtension.SetLabel(
			"io.buildpacks.lifecycle.metadata",
			`{
  "stack": {
    "runImage": {
      "image": "some-run-image",
      "mirrors": [
        "some-mirror",
        "other-mirror"
      ]
    }
  },
  "runImage": {
    "topLayer": "some-top-layer",
    "reference": "some-run-image-reference"
  }
}`,
		))
		h.AssertNil(t, mockImageWithExtension.SetLabel(
			"io.buildpacks.build.metadata",
			`{
  "bom": [
    {
      "name": "some-bom-element"
    }
  ],
  "buildpacks": [
    {
      "id": "some-buildpack",
      "version": "some-version"
    },
    {
      "id": "other-buildpack",
      "version": "other-version"
    }
  ],
    "extensions": [
    {
      "id": "some-extension",
      "version": "some-version"
    },
    {
      "id": "other-extension",
      "version": "other-version"
    }
  ],
  "processes": [
    {
      "type": "other-process",
      "command": "/other/process",
      "args": ["opt", "1"],
      "direct": true
    },
    {
      "type": "web",
      "command": "/start/web-process",
      "args": ["-p", "1234"],
      "direct": false
    }
  ],
  "launcher": {
    "version": "0.5.0"
  }
}`,
		))
	})

	it.After(func() {
		mockController.Finish()
	})

	when("the image exists", func() {
		for _, useDaemon := range []bool{true, false} {
			useDaemon := useDaemon
			when(fmt.Sprintf("daemon is %t", useDaemon), func() {
				it.Before(func() {
					if useDaemon {
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/image", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(mockImage, nil).AnyTimes()
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/imageNoRebasable", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(mockImageNoRebasable, nil).AnyTimes()
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/imageRebasableWithoutLabel", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(mockImageRebasableWithoutLabel, nil).AnyTimes()
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/imageWithExtension", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(mockImageWithExtension, nil).AnyTimes()
					} else {
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/image", image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(mockImage, nil).AnyTimes()
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/imageNoRebasable", image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(mockImageNoRebasable, nil).AnyTimes()
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/imageRebasableWithoutLabel", image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(mockImageRebasableWithoutLabel, nil).AnyTimes()
						mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/imageWithExtension", image.FetchOptions{Daemon: false, PullPolicy: image.PullNever}).Return(mockImageWithExtension, nil).AnyTimes()
					}
				})

				it("returns the stack ID", func() {
					info, err := subject.InspectImage("some/image", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, info.StackID, "test.stack.id")
				})

				it("returns the stack ID with extension", func() {
					infoWithExtension, err := subject.InspectImage("some/imageWithExtension", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, infoWithExtension.StackID, "test.stack.id")
				})

				it("returns the stack from runImage.Image if set", func() {
					h.AssertNil(t, mockImage.SetLabel(
						"io.buildpacks.lifecycle.metadata",
						`{
  "runImage": {
    "topLayer": "some-top-layer",
    "reference": "some-run-image-reference",
    "image":  "is everything"
  }
}`,
					))
					info, err := subject.InspectImage("some/image", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, info.Stack,
						files.Stack{RunImage: files.RunImageForExport{Image: "is everything"}})
				})

				it("returns the stack", func() {
					info, err := subject.InspectImage("some/image", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, info.Stack,
						files.Stack{
							RunImage: files.RunImageForExport{
								Image: "some-run-image",
								Mirrors: []string{
									"some-mirror",
									"other-mirror",
								},
							},
						},
					)
				})

				it("returns the stack with extension", func() {
					infoWithExtension, err := subject.InspectImage("some/imageWithExtension", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, infoWithExtension.Stack,
						files.Stack{
							RunImage: files.RunImageForExport{
								Image: "some-run-image",
								Mirrors: []string{
									"some-mirror",
									"other-mirror",
								},
							},
						},
					)
				})

				it("returns the base image", func() {
					info, err := subject.InspectImage("some/image", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, info.Base,
						files.RunImageForRebase{
							TopLayer:  "some-top-layer",
							Reference: "some-run-image-reference",
						},
					)
				})

				it("returns the base image with extension", func() {
					infoWithExtension, err := subject.InspectImage("some/imageWithExtension", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, infoWithExtension.Base,
						files.RunImageForRebase{
							TopLayer:  "some-top-layer",
							Reference: "some-run-image-reference",
						},
					)
				})

				it("returns the rebasable image", func() {
					info, err := subject.InspectImage("some/image", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, info.Rebasable, true)
				})

				it("returns the rebasable image true if the label has not been set", func() {
					info, err := subject.InspectImage("some/imageRebasableWithoutLabel", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, info.Rebasable, true)
				})

				it("returns the no rebasable image", func() {
					info, err := subject.InspectImage("some/imageNoRebasable", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, info.Rebasable, false)
				})

				it("returns the rebasable image with Extension", func() {
					infoRebasableWithExtension, err := subject.InspectImage("some/imageWithExtension", useDaemon)
					h.AssertNil(t, err)
					h.AssertEq(t, infoRebasableWithExtension.Rebasable, true)
				})

				it("returns the BOM", func() {
					info, err := subject.InspectImage("some/image", useDaemon)
					h.AssertNil(t, err)

					rawBOM, err := json.Marshal(info.BOM)
					h.AssertNil(t, err)
					h.AssertContains(t, string(rawBOM), `[{"name":"some-bom-element"`)
				})

				it("returns the BOM", func() {
					infoWithExtension, err := subject.InspectImage("some/imageWithExtension", useDaemon)
					h.AssertNil(t, err)

					rawBOM, err := json.Marshal(infoWithExtension.BOM)
					h.AssertNil(t, err)
					h.AssertContains(t, string(rawBOM), `[{"name":"some-bom-element"`)
				})

				it("returns the buildpacks", func() {
					info, err := subject.InspectImage("some/image", useDaemon)
					h.AssertNil(t, err)

					h.AssertEq(t, len(info.Buildpacks), 2)
					h.AssertEq(t, info.Buildpacks[0].ID, "some-buildpack")
					h.AssertEq(t, info.Buildpacks[0].Version, "some-version")
					h.AssertEq(t, info.Buildpacks[1].ID, "other-buildpack")
					h.AssertEq(t, info.Buildpacks[1].Version, "other-version")
				})

				it("returns the buildpacks with extension", func() {
					infoWithExtension, err := subject.InspectImage("some/imageWithExtension", useDaemon)
					h.AssertNil(t, err)

					h.AssertEq(t, len(infoWithExtension.Buildpacks), 2)
					h.AssertEq(t, infoWithExtension.Buildpacks[0].ID, "some-buildpack")
					h.AssertEq(t, infoWithExtension.Buildpacks[0].Version, "some-version")
					h.AssertEq(t, infoWithExtension.Buildpacks[1].ID, "other-buildpack")
					h.AssertEq(t, infoWithExtension.Buildpacks[1].Version, "other-version")
				})

				it("returns the extensions", func() {
					infoWithExtension, err := subject.InspectImage("some/imageWithExtension", useDaemon)
					h.AssertNil(t, err)

					h.AssertEq(t, len(infoWithExtension.Extensions), 2)
					h.AssertEq(t, infoWithExtension.Extensions[0].ID, "some-extension")
					h.AssertEq(t, infoWithExtension.Extensions[0].Version, "some-version")
					h.AssertEq(t, infoWithExtension.Extensions[1].ID, "other-extension")
					h.AssertEq(t, infoWithExtension.Extensions[1].Version, "other-version")
				})

				it("returns the processes setting the web process as default", func() {
					info, err := subject.InspectImage("some/image", useDaemon)
					h.AssertNil(t, err)

					h.AssertEq(t, info.Processes,
						ProcessDetails{
							DefaultProcess: &launch.Process{
								Type:             "web",
								Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
								Args:             []string{"-p", "1234"},
								Direct:           false,
								WorkingDirectory: "/test-workdir",
							},
							OtherProcesses: []launch.Process{
								{
									Type:             "other-process",
									Command:          launch.RawCommand{Entries: []string{"/other/process"}},
									Args:             []string{"opt", "1"},
									Direct:           true,
									WorkingDirectory: "/test-workdir",
								},
							},
						},
						ignorePlatformAPI...)
				})

				when("Platform API < 0.4", func() {
					when("CNB_PROCESS_TYPE is set", func() {
						it.Before(func() {
							h.AssertNil(t, mockImage.SetEnv("CNB_PROCESS_TYPE", "other-process"))
						})

						it("returns processes setting the correct default process", func() {
							info, err := subject.InspectImage("some/image", useDaemon)
							h.AssertNil(t, err)

							h.AssertEq(t, info.Processes,
								ProcessDetails{
									DefaultProcess: &launch.Process{
										Type:             "other-process",
										Command:          launch.RawCommand{Entries: []string{"/other/process"}},
										Args:             []string{"opt", "1"},
										Direct:           true,
										WorkingDirectory: "/test-workdir",
									},
									OtherProcesses: []launch.Process{
										{
											Type:             "web",
											Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
											Args:             []string{"-p", "1234"},
											Direct:           false,
											WorkingDirectory: "/test-workdir",
										},
									},
								},
								ignorePlatformAPI...)
						})
					})

					when("CNB_PROCESS_TYPE is set, but doesn't match an existing process", func() {
						it.Before(func() {
							h.AssertNil(t, mockImage.SetEnv("CNB_PROCESS_TYPE", "missing-process"))
						})

						it("returns a nil default process", func() {
							info, err := subject.InspectImage("some/image", useDaemon)
							h.AssertNil(t, err)

							h.AssertEq(t, info.Processes,
								ProcessDetails{
									DefaultProcess: nil,
									OtherProcesses: []launch.Process{
										{
											Type:             "other-process",
											Command:          launch.RawCommand{Entries: []string{"/other/process"}},
											Args:             []string{"opt", "1"},
											Direct:           true,
											WorkingDirectory: "/test-workdir",
										},
										{
											Type:             "web",
											Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
											Args:             []string{"-p", "1234"},
											Direct:           false,
											WorkingDirectory: "/test-workdir",
										},
									},
								},
								ignorePlatformAPI...)
						})
					})

					it("returns a nil default process when CNB_PROCESS_TYPE is not set and there is no web process", func() {
						h.AssertNil(t, mockImage.SetLabel(
							"io.buildpacks.build.metadata",
							`{
  "processes": [
    {
      "type": "other-process",
      "command": "/other/process",
      "args": ["opt", "1"],
      "direct": true
    }
  ]
}`,
						))

						info, err := subject.InspectImage("some/image", useDaemon)
						h.AssertNil(t, err)

						h.AssertEq(t, info.Processes,
							ProcessDetails{
								DefaultProcess: nil,
								OtherProcesses: []launch.Process{
									{
										Type:             "other-process",
										Command:          launch.RawCommand{Entries: []string{"/other/process"}},
										Args:             []string{"opt", "1"},
										Direct:           true,
										WorkingDirectory: "/test-workdir",
									},
								},
							},
							ignorePlatformAPI...)
					})
				})

				when("Platform API >= 0.4 and <= 0.8", func() {
					it.Before(func() {
						h.AssertNil(t, mockImage.SetEnv("CNB_PLATFORM_API", "0.4"))
					})

					when("CNB_PLATFORM_API set to bad value", func() {
						it("errors", func() {
							h.AssertNil(t, mockImage.SetEnv("CNB_PLATFORM_API", "not-semver"))
							_, err := subject.InspectImage("some/image", useDaemon)
							h.AssertError(t, err, "parsing platform api version")
						})
					})

					when("Can't inspect Image entrypoint", func() {
						it("errors", func() {
							mockImage.EntrypointCall.Returns.Error = errors.New("some-error")

							_, err := subject.InspectImage("some/image", useDaemon)
							h.AssertError(t, err, "reading entrypoint")
						})
					})

					when("ENTRYPOINT is empty", func() {
						it("sets nil default process", func() {
							info, err := subject.InspectImage("some/image", useDaemon)
							h.AssertNil(t, err)

							h.AssertEq(t, info.Processes,
								ProcessDetails{
									DefaultProcess: nil,
									OtherProcesses: []launch.Process{
										{
											Type:             "other-process",
											Command:          launch.RawCommand{Entries: []string{"/other/process"}},
											Args:             []string{"opt", "1"},
											Direct:           true,
											WorkingDirectory: "/test-workdir",
										},
										{
											Type:             "web",
											Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
											Args:             []string{"-p", "1234"},
											Direct:           false,
											WorkingDirectory: "/test-workdir",
										},
									},
								},
								ignorePlatformAPI...)
						})
					})

					when("CNB_PROCESS_TYPE is set", func() {
						it.Before(func() {
							h.AssertNil(t, mockImage.SetEnv("CNB_PROCESS_TYPE", "other-process"))

							mockImage.EntrypointCall.Returns.StringArr = []string{"/cnb/process/web"}
						})

						it("ignores it and sets the correct default process", func() {
							info, err := subject.InspectImage("some/image", useDaemon)
							h.AssertNil(t, err)

							h.AssertEq(t, info.Processes,
								ProcessDetails{
									DefaultProcess: &launch.Process{
										Type:             "web",
										Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
										Args:             []string{"-p", "1234"},
										Direct:           false,
										WorkingDirectory: "/test-workdir",
									},
									OtherProcesses: []launch.Process{
										{
											Type:             "other-process",
											Command:          launch.RawCommand{Entries: []string{"/other/process"}},
											Args:             []string{"opt", "1"},
											Direct:           true,
											WorkingDirectory: "/test-workdir",
										},
									},
								},
								ignorePlatformAPI...)
						})
					})

					when("ENTRYPOINT is set, but doesn't match an existing process", func() {
						it.Before(func() {
							mockImage.EntrypointCall.Returns.StringArr = []string{"/cnb/process/unknown-process"}
						})

						it("returns nil default default process", func() {
							info, err := subject.InspectImage("some/image", useDaemon)
							h.AssertNil(t, err)

							h.AssertEq(t, info.Processes,
								ProcessDetails{
									DefaultProcess: nil,
									OtherProcesses: []launch.Process{
										{
											Type:             "other-process",
											Command:          launch.RawCommand{Entries: []string{"/other/process"}},
											Args:             []string{"opt", "1"},
											Direct:           true,
											WorkingDirectory: "/test-workdir",
										},
										{
											Type:             "web",
											Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
											Args:             []string{"-p", "1234"},
											Direct:           false,
											WorkingDirectory: "/test-workdir",
										},
									},
								},
								ignorePlatformAPI...)
						})
					})

					when("ENTRYPOINT set to /cnb/lifecycle/launcher", func() {
						it("returns a nil default process", func() {
							mockImage.EntrypointCall.Returns.StringArr = []string{"/cnb/lifecycle/launcher"}

							h.AssertNil(t, mockImage.SetLabel(
								"io.buildpacks.build.metadata",
								`{
					 "processes": [
					   {
					     "type": "other-process",
					     "command": "/other/process",
					     "args": ["opt", "1"],
					     "direct": true
					   }
					 ]
					}`,
							))

							info, err := subject.InspectImage("some/image", useDaemon)
							h.AssertNil(t, err)

							h.AssertEq(t, info.Processes,
								ProcessDetails{
									DefaultProcess: nil,
									OtherProcesses: []launch.Process{
										{
											Type:             "other-process",
											Command:          launch.RawCommand{Entries: []string{"/other/process"}},
											Args:             []string{"opt", "1"},
											Direct:           true,
											WorkingDirectory: "/test-workdir",
										},
									},
								},
								ignorePlatformAPI...)
						})
					})

					when("Inspecting Windows images", func() {
						when(`ENTRYPOINT set to c:\cnb\lifecycle\launcher.exe`, func() {
							it("sets default process to nil", func() {
								mockImage.EntrypointCall.Returns.StringArr = []string{`c:\cnb\lifecycle\launcher.exe`}

								info, err := subject.InspectImage("some/image", useDaemon)
								h.AssertNil(t, err)

								h.AssertEq(t, info.Processes,
									ProcessDetails{
										DefaultProcess: nil,
										OtherProcesses: []launch.Process{
											{
												Type:             "other-process",
												Command:          launch.RawCommand{Entries: []string{"/other/process"}},
												Args:             []string{"opt", "1"},
												Direct:           true,
												WorkingDirectory: "/test-workdir",
											},
											{
												Type:             "web",
												Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
												Args:             []string{"-p", "1234"},
												Direct:           false,
												WorkingDirectory: "/test-workdir",
											},
										},
									},
									ignorePlatformAPI...)
							})
						})

						when("ENTRYPOINT is set, but doesn't match an existing process", func() {
							it("sets default process to nil", func() {
								mockImage.EntrypointCall.Returns.StringArr = []string{`c:\cnb\process\unknown-process.exe`}

								info, err := subject.InspectImage("some/image", useDaemon)
								h.AssertNil(t, err)

								h.AssertEq(t, info.Processes,
									ProcessDetails{
										DefaultProcess: nil,
										OtherProcesses: []launch.Process{
											{
												Type:             "other-process",
												Command:          launch.RawCommand{Entries: []string{"/other/process"}},
												Args:             []string{"opt", "1"},
												Direct:           true,
												WorkingDirectory: "/test-workdir",
											},
											{
												Type:             "web",
												Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
												Args:             []string{"-p", "1234"},
												Direct:           false,
												WorkingDirectory: "/test-workdir",
											},
										},
									},
									ignorePlatformAPI...)
							})
						})

						when("ENTRYPOINT is set, and matches an existing process", func() {
							it("sets default process to defined process", func() {
								mockImage.EntrypointCall.Returns.StringArr = []string{`c:\cnb\process\other-process.exe`}

								info, err := subject.InspectImage("some/image", useDaemon)
								h.AssertNil(t, err)

								h.AssertEq(t, info.Processes,
									ProcessDetails{
										DefaultProcess: &launch.Process{
											Type:             "other-process",
											Command:          launch.RawCommand{Entries: []string{"/other/process"}},
											Args:             []string{"opt", "1"},
											Direct:           true,
											WorkingDirectory: "/test-workdir",
										},
										OtherProcesses: []launch.Process{
											{
												Type:             "web",
												Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
												Args:             []string{"-p", "1234"},
												Direct:           false,
												WorkingDirectory: "/test-workdir",
											},
										},
									},
									ignorePlatformAPI...)
							})
						})
					})
				})

				when("Platform API > 0.8", func() {
					when("working-dir is set", func() {
						it("returns process with working directory if available", func() {
							h.AssertNil(t, mockImage.SetLabel(
								"io.buildpacks.build.metadata",
								`{
					 "processes": [
					   {
					     "type": "other-process",
					     "command": "/other/process",
					     "args": ["opt", "1"],
					     "direct": true,
						 "working-dir": "/other-workdir"
					   }
					 ]
					}`,
							))

							info, err := subject.InspectImage("some/image", useDaemon)
							h.AssertNil(t, err)
							fmt.Print(info)

							h.AssertEq(t, info.Processes,
								ProcessDetails{
									DefaultProcess: nil,
									OtherProcesses: []launch.Process{
										{
											Type:             "other-process",
											Command:          launch.RawCommand{Entries: []string{"/other/process"}},
											Args:             []string{"opt", "1"},
											Direct:           true,
											WorkingDirectory: "/other-workdir",
										},
									},
								},
								ignorePlatformAPI...)
						})
					})

					when("working-dir is not set", func() {
						it("returns process with working directory from image", func() {
							info, err := subject.InspectImage("some/image", useDaemon)
							h.AssertNil(t, err)

							h.AssertEq(t, info.Processes,
								ProcessDetails{
									DefaultProcess: &launch.Process{
										Type:             "web",
										Command:          launch.RawCommand{Entries: []string{"/start/web-process"}},
										Args:             []string{"-p", "1234"},
										Direct:           false,
										WorkingDirectory: "/test-workdir",
									},
									OtherProcesses: []launch.Process{
										{
											Type:             "other-process",
											Command:          launch.RawCommand{Entries: []string{"/other/process"}},
											Args:             []string{"opt", "1"},
											Direct:           true,
											WorkingDirectory: "/test-workdir",
										},
									},
								},
								ignorePlatformAPI...)
						})
					})
				})
			})
		}
	})

	when("the image doesn't exist", func() {
		it("returns nil", func() {
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "not/some-image", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(nil, image.ErrNotFound)

			info, err := subject.InspectImage("not/some-image", true)
			h.AssertNil(t, err)
			h.AssertNil(t, info)
		})
	})

	when("there is an error fetching the image", func() {
		it("returns the error", func() {
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "not/some-image", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(nil, errors.New("some-error"))

			_, err := subject.InspectImage("not/some-image", true)
			h.AssertError(t, err, "some-error")
		})
	})

	when("the image is missing labels", func() {
		it("returns empty data", func() {
			mockImageFetcher.EXPECT().
				Fetch(gomock.Any(), "missing/labels", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).
				Return(fakes.NewImage("missing/labels", "", nil), nil)
			info, err := subject.InspectImage("missing/labels", true)
			h.AssertNil(t, err)
			h.AssertEq(t, info, &ImageInfo{Rebasable: true}, ignorePlatformAPI...)
		})
	})

	when("the image has malformed labels", func() {
		var badImage *fakes.Image

		it.Before(func() {
			badImage = fakes.NewImage("bad/image", "", nil)
			mockImageFetcher.EXPECT().
				Fetch(gomock.Any(), "bad/image", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).
				Return(badImage, nil)
		})

		it("returns an error when layers md cannot parse", func() {
			h.AssertNil(t, badImage.SetLabel("io.buildpacks.lifecycle.metadata", "not   ----  json"))
			_, err := subject.InspectImage("bad/image", true)
			h.AssertError(t, err, "unmarshalling label 'io.buildpacks.lifecycle.metadata'")
		})

		it("returns an error when build md cannot parse", func() {
			h.AssertNil(t, badImage.SetLabel("io.buildpacks.build.metadata", "not   ----  json"))
			_, err := subject.InspectImage("bad/image", true)
			h.AssertError(t, err, "unmarshalling label 'io.buildpacks.build.metadata'")
		})
	})

	when("lifecycle version is 0.4.x or earlier", func() {
		it("includes an empty base image reference", func() {
			oldImage := fakes.NewImage("old/image", "", nil)
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "old/image", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(oldImage, nil)

			h.AssertNil(t, oldImage.SetLabel(
				"io.buildpacks.lifecycle.metadata",
				`{
  "runImage": {
    "topLayer": "some-top-layer",
    "reference": "some-run-image-reference"
  }
}`,
			))
			h.AssertNil(t, oldImage.SetLabel(
				"io.buildpacks.build.metadata",
				`{
  "launcher": {
    "version": "0.4.0"
  }
}`,
			))

			info, err := subject.InspectImage("old/image", true)
			h.AssertNil(t, err)
			h.AssertEq(t, info.Base,
				files.RunImageForRebase{
					TopLayer:  "some-top-layer",
					Reference: "",
				},
			)
		})
	})
}
