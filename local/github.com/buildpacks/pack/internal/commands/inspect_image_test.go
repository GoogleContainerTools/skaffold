package commands_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/fakes"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/inspectimage"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

var (
	expectedLocalImageDisplay  = "Sample output for local image"
	expectedRemoteImageDisplay = "Sample output for remote image"

	expectedSharedInfo = inspectimage.GeneralInfo{
		Name: "some/image",
	}

	expectedLocalImageInfo = &client.ImageInfo{
		StackID:    "local.image.stack",
		Buildpacks: nil,
		Base:       files.RunImageForRebase{},
		BOM:        nil,
		Stack:      files.Stack{},
		Processes:  client.ProcessDetails{},
	}

	expectedRemoteImageInfo = &client.ImageInfo{
		StackID:    "remote.image.stack",
		Buildpacks: nil,
		Base:       files.RunImageForRebase{},
		BOM:        nil,
		Stack:      files.Stack{},
		Processes:  client.ProcessDetails{},
	}

	expectedLocalImageWithExtensionInfo = &client.ImageInfo{
		StackID:    "local.image.stack",
		Buildpacks: nil,
		Extensions: nil,
		Base:       files.RunImageForRebase{},
		BOM:        nil,
		Stack:      files.Stack{},
		Processes:  client.ProcessDetails{},
	}

	expectedRemoteImageWithExtensionInfo = &client.ImageInfo{
		StackID:    "remote.image.stack",
		Buildpacks: nil,
		Extensions: nil,
		Base:       files.RunImageForRebase{},
		BOM:        nil,
		Stack:      files.Stack{},
		Processes:  client.ProcessDetails{},
	}
)

func TestInspectImageCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Commands", testInspectImageCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testInspectImageCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		logger         logging.Logger
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
		cfg            config.Config
	)

	it.Before(func() {
		cfg = config.Config{}
		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
	})

	it.After(func() {
		mockController.Finish()
	})
	when("#InspectImage", func() {
		var (
			assert = h.NewAssertionManager(t)
		)
		it("passes output of local and remote builders to correct writer", func() {
			inspectImageWriter := newDefaultInspectImageWriter()
			inspectImageWriterFactory := newImageWriterFactory(inspectImageWriter)

			mockClient.EXPECT().InspectImage("some/image", true).Return(expectedLocalImageInfo, nil)
			mockClient.EXPECT().InspectImage("some/image", false).Return(expectedRemoteImageInfo, nil)
			command := commands.InspectImage(logger, inspectImageWriterFactory, cfg, mockClient)
			command.SetArgs([]string{"some/image"})
			err := command.Execute()
			assert.Nil(err)

			assert.Equal(inspectImageWriter.ReceivedInfoForLocal, expectedLocalImageInfo)
			assert.Equal(inspectImageWriter.ReceivedInfoForRemote, expectedRemoteImageInfo)
			assert.Equal(inspectImageWriter.RecievedGeneralInfo, expectedSharedInfo)
			assert.Equal(inspectImageWriter.ReceivedErrorForLocal, nil)
			assert.Equal(inspectImageWriter.ReceivedErrorForRemote, nil)
			assert.Equal(inspectImageWriterFactory.ReceivedForKind, "human-readable")

			assert.ContainsF(outBuf.String(), "LOCAL:\n%s", expectedLocalImageDisplay)
			assert.ContainsF(outBuf.String(), "REMOTE:\n%s", expectedRemoteImageDisplay)
		})

		it("passes output of local and remote builders to correct writer for extension", func() {
			inspectImageWriter := newDefaultInspectImageWriter()
			inspectImageWriterFactory := newImageWriterFactory(inspectImageWriter)

			mockClient.EXPECT().InspectImage("some/image", true).Return(expectedLocalImageWithExtensionInfo, nil)
			mockClient.EXPECT().InspectImage("some/image", false).Return(expectedRemoteImageWithExtensionInfo, nil)
			command := commands.InspectImage(logger, inspectImageWriterFactory, cfg, mockClient)
			command.SetArgs([]string{"some/image"})
			err := command.Execute()
			assert.Nil(err)

			assert.Equal(inspectImageWriter.ReceivedInfoForLocal, expectedLocalImageWithExtensionInfo)
			assert.Equal(inspectImageWriter.ReceivedInfoForRemote, expectedRemoteImageWithExtensionInfo)
			assert.Equal(inspectImageWriter.RecievedGeneralInfo, expectedSharedInfo)
			assert.Equal(inspectImageWriter.ReceivedErrorForLocal, nil)
			assert.Equal(inspectImageWriter.ReceivedErrorForRemote, nil)
			assert.Equal(inspectImageWriterFactory.ReceivedForKind, "human-readable")

			assert.ContainsF(outBuf.String(), "LOCAL:\n%s", expectedLocalImageDisplay)
			assert.ContainsF(outBuf.String(), "REMOTE:\n%s", expectedRemoteImageDisplay)
		})

		it("passes configured run image mirrors to the writer", func() {
			cfg = config.Config{
				RunImages: []config.RunImage{{
					Image:   "image-name",
					Mirrors: []string{"first-mirror", "second-mirror2"},
				},
					{
						Image:   "image-name2",
						Mirrors: []string{"other-mirror"},
					},
				},
				TrustedBuilders: nil,
				Registries:      nil,
			}

			inspectImageWriter := newDefaultInspectImageWriter()
			inspectImageWriterFactory := newImageWriterFactory(inspectImageWriter)

			mockClient.EXPECT().InspectImage("some/image", true).Return(expectedLocalImageInfo, nil)
			mockClient.EXPECT().InspectImage("some/image", false).Return(expectedRemoteImageInfo, nil)

			command := commands.InspectImage(logger, inspectImageWriterFactory, cfg, mockClient)
			command.SetArgs([]string{"some/image"})
			err := command.Execute()
			assert.Nil(err)

			assert.Equal(inspectImageWriter.RecievedGeneralInfo.RunImageMirrors, cfg.RunImages)
		})

		it("passes configured run image mirrors to the writer", func() {
			cfg = config.Config{
				RunImages: []config.RunImage{{
					Image:   "image-name",
					Mirrors: []string{"first-mirror", "second-mirror2"},
				},
					{
						Image:   "image-name2",
						Mirrors: []string{"other-mirror"},
					},
				},
				TrustedBuilders: nil,
				Registries:      nil,
			}

			inspectImageWriter := newDefaultInspectImageWriter()
			inspectImageWriterFactory := newImageWriterFactory(inspectImageWriter)

			mockClient.EXPECT().InspectImage("some/image", true).Return(expectedLocalImageWithExtensionInfo, nil)
			mockClient.EXPECT().InspectImage("some/image", false).Return(expectedRemoteImageWithExtensionInfo, nil)

			command := commands.InspectImage(logger, inspectImageWriterFactory, cfg, mockClient)
			command.SetArgs([]string{"some/image"})
			err := command.Execute()
			assert.Nil(err)

			assert.Equal(inspectImageWriter.RecievedGeneralInfo.RunImageMirrors, cfg.RunImages)
		})

		when("error cases", func() {
			when("client returns an error when inspecting", func() {
				it("passes errors to the Writer", func() {
					inspectImageWriter := newDefaultInspectImageWriter()
					inspectImageWriterFactory := newImageWriterFactory(inspectImageWriter)

					localErr := errors.New("local inspection error")
					mockClient.EXPECT().InspectImage("some/image", true).Return(nil, localErr)

					remoteErr := errors.New("remote inspection error")
					mockClient.EXPECT().InspectImage("some/image", false).Return(nil, remoteErr)

					command := commands.InspectImage(logger, inspectImageWriterFactory, cfg, mockClient)
					command.SetArgs([]string{"some/image"})
					err := command.Execute()
					assert.Nil(err)

					assert.ErrorWithMessage(inspectImageWriter.ReceivedErrorForLocal, "local inspection error")
					assert.ErrorWithMessage(inspectImageWriter.ReceivedErrorForRemote, "remote inspection error")
				})
			})

			when("writerFactory fails to create a writer", func() {
				it("returns the error", func() {
					writerFactoryErr := errors.New("unable to create writer factory")

					erroniousInspectImageWriterFactory := &fakes.FakeInspectImageWriterFactory{
						ReturnForWriter: nil,
						ErrorForWriter:  writerFactoryErr,
					}

					command := commands.InspectImage(logger, erroniousInspectImageWriterFactory, cfg, mockClient)
					command.SetArgs([]string{"some/image"})
					err := command.Execute()
					assert.ErrorWithMessage(err, "unable to create writer factory")
				})
			})
			when("Print returns fails", func() {
				it("returns the error", func() {
					printError := errors.New("unable to print")
					inspectImageWriter := &fakes.FakeInspectImageWriter{
						ErrorForPrint: printError,
					}
					inspectImageWriterFactory := newImageWriterFactory(inspectImageWriter)

					mockClient.EXPECT().InspectImage("some/image", true).Return(expectedLocalImageInfo, nil)
					mockClient.EXPECT().InspectImage("some/image", false).Return(expectedRemoteImageInfo, nil)

					command := commands.InspectImage(logger, inspectImageWriterFactory, cfg, mockClient)
					command.SetArgs([]string{"some/image"})
					err := command.Execute()
					assert.ErrorWithMessage(err, "unable to print")
				})
			})

			when("Print returns fails for extension", func() {
				it("returns the error", func() {
					printError := errors.New("unable to print")
					inspectImageWriter := &fakes.FakeInspectImageWriter{
						ErrorForPrint: printError,
					}
					inspectImageWriterFactory := newImageWriterFactory(inspectImageWriter)

					mockClient.EXPECT().InspectImage("some/image", true).Return(expectedLocalImageWithExtensionInfo, nil)
					mockClient.EXPECT().InspectImage("some/image", false).Return(expectedRemoteImageWithExtensionInfo, nil)

					command := commands.InspectImage(logger, inspectImageWriterFactory, cfg, mockClient)
					command.SetArgs([]string{"some/image"})
					err := command.Execute()
					assert.ErrorWithMessage(err, "unable to print")
				})
			})
		})
	})
}

func newDefaultInspectImageWriter() *fakes.FakeInspectImageWriter {
	return &fakes.FakeInspectImageWriter{
		PrintForLocal:  expectedLocalImageDisplay,
		PrintForRemote: expectedRemoteImageDisplay,
	}
}

func newImageWriterFactory(writer *fakes.FakeInspectImageWriter) *fakes.FakeInspectImageWriterFactory {
	return &fakes.FakeInspectImageWriterFactory{
		ReturnForWriter: writer,
	}
}
