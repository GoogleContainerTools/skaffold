package commands_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

const extensionOutputSection = `Extension:
  ID                           NAME        VERSION        HOMEPAGE
  some/single-extension        some        0.0.1          single-extension-homepage`

const inspectExtensionOutputTemplate = `Inspecting extension: '%s'

%s

%s
`

func TestExtensionInspectCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ExtensionInspectCommand", testExtensionInspectCommand, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testExtensionInspectCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command        *cobra.Command
		logger         logging.Logger
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
		cfg            config.Config
		info           *client.ExtensionInfo
		assert         = h.NewAssertionManager(t)
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)

		info = &client.ExtensionInfo{
			Extension: dist.ModuleInfo{
				ID:       "some/single-extension",
				Version:  "0.0.1",
				Name:     "some",
				Homepage: "single-extension-homepage",
			},
		}

		command = commands.ExtensionInspect(logger, cfg, mockClient)
	})

	when("ExtensionInspect", func() {
		when("inspecting an image", func() {
			when("both remote and local image are present", func() {
				it.Before(func() {
					info.Location = buildpack.PackageLocator

					mockClient.EXPECT().InspectExtension(client.InspectExtensionOptions{
						ExtensionName: "test/extension",
						Daemon:        true,
					}).Return(info, nil)

					mockClient.EXPECT().InspectExtension(client.InspectExtensionOptions{
						ExtensionName: "test/extension",
						Daemon:        false,
					}).Return(info, nil)
				})

				it("succeeds", func() {
					command.SetArgs([]string{"test/extension"})
					assert.Nil(command.Execute())

					localOutputSection := fmt.Sprintf(inspectExtensionOutputTemplate,
						"test/extension",
						"LOCAL IMAGE:",
						extensionOutputSection)

					remoteOutputSection := fmt.Sprintf("%s\n\n%s",
						"REMOTE IMAGE:",
						extensionOutputSection)

					assert.AssertTrimmedContains(outBuf.String(), localOutputSection)
					assert.AssertTrimmedContains(outBuf.String(), remoteOutputSection)
				})
			})

			when("only a local image is present", func() {
				it.Before(func() {
					info.Location = buildpack.PackageLocator

					mockClient.EXPECT().InspectExtension(client.InspectExtensionOptions{
						ExtensionName: "only-local-test/extension",
						Daemon:        true,
					}).Return(info, nil)

					mockClient.EXPECT().InspectExtension(client.InspectExtensionOptions{
						ExtensionName: "only-local-test/extension",
						Daemon:        false,
					}).Return(nil, errors.Wrap(image.ErrNotFound, "remote image not found!"))
				})

				it("displays output for local image", func() {
					command.SetArgs([]string{"only-local-test/extension"})
					assert.Nil(command.Execute())

					expectedOutput := fmt.Sprintf(inspectExtensionOutputTemplate,
						"only-local-test/extension",
						"LOCAL IMAGE:",
						extensionOutputSection)

					assert.AssertTrimmedContains(outBuf.String(), expectedOutput)
				})
			})

			when("only a remote image is present", func() {
				it.Before(func() {
					info.Location = buildpack.PackageLocator

					mockClient.EXPECT().InspectExtension(client.InspectExtensionOptions{
						ExtensionName: "only-remote-test/extension",
						Daemon:        false,
					}).Return(info, nil)

					mockClient.EXPECT().InspectExtension(client.InspectExtensionOptions{
						ExtensionName: "only-remote-test/extension",
						Daemon:        true,
					}).Return(nil, errors.Wrap(image.ErrNotFound, "local image not found!"))
				})

				it("displays output for remote image", func() {
					command.SetArgs([]string{"only-remote-test/extension"})
					assert.Nil(command.Execute())

					expectedOutput := fmt.Sprintf(inspectExtensionOutputTemplate,
						"only-remote-test/extension",
						"REMOTE IMAGE:",
						extensionOutputSection)

					assert.AssertTrimmedContains(outBuf.String(), expectedOutput)
				})
			})
		})
	})

	when("failure cases", func() {
		when("unable to inspect extension image", func() {
			it.Before(func() {
				mockClient.EXPECT().InspectExtension(client.InspectExtensionOptions{
					ExtensionName: "failure-case/extension",
					Daemon:        true,
				}).Return(&client.ExtensionInfo{}, errors.Wrap(image.ErrNotFound, "unable to inspect local failure-case/extension"))

				mockClient.EXPECT().InspectExtension(client.InspectExtensionOptions{
					ExtensionName: "failure-case/extension",
					Daemon:        false,
				}).Return(&client.ExtensionInfo{}, errors.Wrap(image.ErrNotFound, "unable to inspect remote failure-case/extension"))
			})

			it("errors", func() {
				command.SetArgs([]string{"failure-case/extension"})
				err := command.Execute()
				assert.Error(err)
			})
		})
	})
}
