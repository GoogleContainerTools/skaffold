package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"

	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestBuildpackNewCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "BuildpackNewCommand", testBuildpackNewCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testBuildpackNewCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command        *cobra.Command
		logger         *logging.LogWithWriters
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
		tmpDir         string
	)
	targets := []dist.Target{{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}}

	it.Before(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "build-test")
		h.AssertNil(t, err)

		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)

		command = commands.BuildpackNew(logger, mockClient)
	})

	it.After(func() {
		os.RemoveAll(tmpDir)
	})

	when("BuildpackNew#Execute", func() {
		it("uses the args to generate artifacts", func() {
			mockClient.EXPECT().NewBuildpack(gomock.Any(), client.NewBuildpackOptions{
				API:     "0.8",
				ID:      "example/some-cnb",
				Path:    filepath.Join(tmpDir, "some-cnb"),
				Version: "1.0.0",
				Targets: targets,
			}).Return(nil).MaxTimes(1)

			path := filepath.Join(tmpDir, "some-cnb")
			command.SetArgs([]string{"--path", path, "example/some-cnb"})

			err := command.Execute()
			h.AssertNil(t, err)
		})

		it("stops if the directory already exists", func() {
			err := os.MkdirAll(tmpDir, 0600)
			h.AssertNil(t, err)

			command.SetArgs([]string{"--path", tmpDir, "example/some-cnb"})
			err = command.Execute()
			h.AssertNotNil(t, err)
			h.AssertContains(t, outBuf.String(), "ERROR: directory")
		})

		when("target flag is specified, ", func() {
			it("it uses target to generate artifacts", func() {
				mockClient.EXPECT().NewBuildpack(gomock.Any(), client.NewBuildpackOptions{
					API:     "0.8",
					ID:      "example/targets",
					Path:    filepath.Join(tmpDir, "targets"),
					Version: "1.0.0",
					Targets: []dist.Target{{
						OS:          "linux",
						Arch:        "arm",
						ArchVariant: "v6",
						Distributions: []dist.Distribution{{
							Name:    "ubuntu",
							Version: "14.04",
						}},
					}},
				}).Return(nil).MaxTimes(1)

				path := filepath.Join(tmpDir, "targets")
				command.SetArgs([]string{"--path", path, "example/targets", "--targets", "linux/arm/v6:ubuntu@14.04"})

				err := command.Execute()
				h.AssertNil(t, err)
			})
			it("it should show error when invalid [os]/[arch] passed", func() {
				mockClient.EXPECT().NewBuildpack(gomock.Any(), client.NewBuildpackOptions{
					API:     "0.8",
					ID:      "example/targets",
					Path:    filepath.Join(tmpDir, "targets"),
					Version: "1.0.0",
					Targets: []dist.Target{{
						OS:          "os",
						Arch:        "arm",
						ArchVariant: "v6",
						Distributions: []dist.Distribution{{
							Name:    "ubuntu",
							Version: "14.04",
						}, {
							Name:    "ubuntu",
							Version: "16.04",
						}},
					}},
				}).Return(nil).MaxTimes(1)

				path := filepath.Join(tmpDir, "targets")
				command.SetArgs([]string{"--path", path, "example/targets", "--targets", "os/arm/v6:ubuntu@14.04@16.04"})

				err := command.Execute()
				h.AssertNotNil(t, err)
			})
			when("it should", func() {
				it("support format [os][/arch][/variant]:[name@version];[some-name@version]", func() {
					mockClient.EXPECT().NewBuildpack(gomock.Any(), client.NewBuildpackOptions{
						API:     "0.8",
						ID:      "example/targets",
						Path:    filepath.Join(tmpDir, "targets"),
						Version: "1.0.0",
						Targets: []dist.Target{
							{
								OS:          "linux",
								Arch:        "arm",
								ArchVariant: "v6",
								Distributions: []dist.Distribution{
									{
										Name:    "ubuntu",
										Version: "14.04",
									},
									{
										Name:    "debian",
										Version: "8.10",
									},
								},
							},
							{
								OS:   "windows",
								Arch: "amd64",
								Distributions: []dist.Distribution{
									{
										Name:    "windows-nano",
										Version: "10.0.19041.1415",
									},
								},
							},
						},
					}).Return(nil).MaxTimes(1)

					path := filepath.Join(tmpDir, "targets")
					command.SetArgs([]string{"--path", path, "example/targets", "--targets", "linux/arm/v6:ubuntu@14.04;debian@8.10", "-t", "windows/amd64:windows-nano@10.0.19041.1415"})

					err := command.Execute()
					h.AssertNil(t, err)
				})
			})
			when("stacks ", func() {
				it("flag should show deprecated message when used", func() {
					mockClient.EXPECT().NewBuildpack(gomock.Any(), client.NewBuildpackOptions{
						API:     "0.8",
						ID:      "example/stacks",
						Path:    filepath.Join(tmpDir, "stacks"),
						Version: "1.0.0",
						Stacks: []dist.Stack{{
							ID:     "io.buildpacks.stacks.jammy",
							Mixins: []string{},
						}},
					}).Return(nil).MaxTimes(1)

					path := filepath.Join(tmpDir, "stacks")
					output := new(bytes.Buffer)
					command.SetOut(output)
					command.SetErr(output)
					command.SetArgs([]string{"--path", path, "example/stacks", "--stacks", "io.buildpacks.stacks.jammy"})

					err := command.Execute()
					h.AssertNil(t, err)
					h.AssertContains(t, output.String(), "Flag --stacks has been deprecated,")
				})
			})
		})
	})
}
