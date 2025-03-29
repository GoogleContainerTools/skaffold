package commands_test

import (
	"bytes"
	"errors"
	"regexp"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestInspectBuilderCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "InspectBuilderCommand", testInspectBuilderCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testInspectBuilderCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		logger logging.Logger
		outBuf bytes.Buffer
		cfg    config.Config
	)

	it.Before(func() {
		cfg = config.Config{
			DefaultBuilder: "default/builder",
			RunImages:      expectedLocalRunImages,
		}
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
	})

	when("InspectBuilder", func() {
		var (
			assert = h.NewAssertionManager(t)
		)

		it("passes output of local and remote builders to correct writer", func() {
			builderInspector := newDefaultBuilderInspector()
			builderWriter := newDefaultBuilderWriter()
			builderWriterFactory := newWriterFactory(returnsForWriter(builderWriter))

			command := commands.InspectBuilder(logger, cfg, builderInspector, builderWriterFactory)
			command.SetArgs([]string{})
			err := command.Execute()
			assert.Nil(err)

			assert.Equal(builderWriter.ReceivedInfoForLocal, expectedLocalInfo)
			assert.Equal(builderWriter.ReceivedInfoForRemote, expectedRemoteInfo)
			assert.Equal(builderWriter.ReceivedBuilderInfo, expectedBuilderInfo)
			assert.Equal(builderWriter.ReceivedLocalRunImages, expectedLocalRunImages)
			assert.Equal(builderWriterFactory.ReceivedForKind, "human-readable")
			assert.Equal(builderInspector.ReceivedForLocalName, "default/builder")
			assert.Equal(builderInspector.ReceivedForRemoteName, "default/builder")
			assert.ContainsF(outBuf.String(), "LOCAL:\n%s", expectedLocalDisplay)
			assert.ContainsF(outBuf.String(), "REMOTE:\n%s", expectedRemoteDisplay)
		})

		when("image name is provided as first arg", func() {
			it("passes that image name to the inspector", func() {
				builderInspector := newDefaultBuilderInspector()
				writer := newDefaultBuilderWriter()
				command := commands.InspectBuilder(logger, cfg, builderInspector, newWriterFactory(returnsForWriter(writer)))
				command.SetArgs([]string{"some/image"})

				err := command.Execute()
				assert.Nil(err)

				assert.Equal(builderInspector.ReceivedForLocalName, "some/image")
				assert.Equal(builderInspector.ReceivedForRemoteName, "some/image")
				assert.Equal(writer.ReceivedBuilderInfo.IsDefault, false)
			})
		})

		when("depth flag is provided", func() {
			it("passes a modifier to the builder inspector", func() {
				builderInspector := newDefaultBuilderInspector()
				command := commands.InspectBuilder(logger, cfg, builderInspector, newDefaultWriterFactory())
				command.SetArgs([]string{"--depth", "5"})

				err := command.Execute()
				assert.Nil(err)

				assert.Equal(builderInspector.CalculatedConfigForLocal.OrderDetectionDepth, 5)
				assert.Equal(builderInspector.CalculatedConfigForRemote.OrderDetectionDepth, 5)
			})
		})

		when("output type is set to json", func() {
			it("passes json to the writer factory", func() {
				writerFactory := newDefaultWriterFactory()
				command := commands.InspectBuilder(logger, cfg, newDefaultBuilderInspector(), writerFactory)
				command.SetArgs([]string{"--output", "json"})

				err := command.Execute()
				assert.Nil(err)

				assert.Equal(writerFactory.ReceivedForKind, "json")
			})
		})

		when("output type is set to toml using the shorthand flag", func() {
			it("passes toml to the writer factory", func() {
				writerFactory := newDefaultWriterFactory()
				command := commands.InspectBuilder(logger, cfg, newDefaultBuilderInspector(), writerFactory)
				command.SetArgs([]string{"-o", "toml"})

				err := command.Execute()
				assert.Nil(err)

				assert.Equal(writerFactory.ReceivedForKind, "toml")
			})
		})

		when("builder inspector returns an error for local builder", func() {
			it("passes that error to the writer to handle appropriately", func() {
				baseError := errors.New("couldn't inspect local")

				builderInspector := newBuilderInspector(errorsForLocal(baseError))
				builderWriter := newDefaultBuilderWriter()
				builderWriterFactory := newWriterFactory(returnsForWriter(builderWriter))

				command := commands.InspectBuilder(logger, cfg, builderInspector, builderWriterFactory)
				command.SetArgs([]string{})
				err := command.Execute()
				assert.Nil(err)

				assert.ErrorWithMessage(builderWriter.ReceivedErrorForLocal, "couldn't inspect local")
			})
		})

		when("builder inspector returns an error remote builder", func() {
			it("passes that error to the writer to handle appropriately", func() {
				baseError := errors.New("couldn't inspect remote")

				builderInspector := newBuilderInspector(errorsForRemote(baseError))
				builderWriter := newDefaultBuilderWriter()
				builderWriterFactory := newWriterFactory(returnsForWriter(builderWriter))

				command := commands.InspectBuilder(logger, cfg, builderInspector, builderWriterFactory)
				command.SetArgs([]string{})
				err := command.Execute()
				assert.Nil(err)

				assert.ErrorWithMessage(builderWriter.ReceivedErrorForRemote, "couldn't inspect remote")
			})
		})

		when("image is trusted", func() {
			it("passes builder info with trusted true to the writer's `Print` method", func() {
				cfg.TrustedBuilders = []config.TrustedBuilder{
					{Name: "trusted/builder"},
				}
				writer := newDefaultBuilderWriter()

				command := commands.InspectBuilder(
					logger,
					cfg,
					newDefaultBuilderInspector(),
					newWriterFactory(returnsForWriter(writer)),
				)
				command.SetArgs([]string{"trusted/builder"})

				err := command.Execute()
				assert.Nil(err)

				assert.Equal(writer.ReceivedBuilderInfo.Trusted, true)
			})
		})

		when("default builder is configured and is the same as specified by the command", func() {
			it("passes builder info with isDefault true to the writer's `Print` method", func() {
				cfg.DefaultBuilder = "the/default-builder"
				writer := newDefaultBuilderWriter()

				command := commands.InspectBuilder(
					logger,
					cfg,
					newDefaultBuilderInspector(),
					newWriterFactory(returnsForWriter(writer)),
				)
				command.SetArgs([]string{"the/default-builder"})

				err := command.Execute()
				assert.Nil(err)

				assert.Equal(writer.ReceivedBuilderInfo.IsDefault, true)
			})
		})

		when("default builder is empty and no builder is specified in command args", func() {
			it("suggests builders and returns a soft error", func() {
				cfg.DefaultBuilder = ""

				command := commands.InspectBuilder(logger, cfg, newDefaultBuilderInspector(), newDefaultWriterFactory())
				command.SetArgs([]string{})

				err := command.Execute()
				assert.Error(err)
				if !errors.Is(err, client.SoftError{}) {
					t.Fatalf("expect a client.SoftError, got: %s", err)
				}

				assert.Contains(outBuf.String(), `Please select a default builder with:

	pack config default-builder <builder-image>`)

				assert.Matches(outBuf.String(), regexp.MustCompile(`Paketo Buildpacks:\s+'paketobuildpacks/builder-jammy-base'`))
				assert.Matches(outBuf.String(), regexp.MustCompile(`Paketo Buildpacks:\s+'paketobuildpacks/builder-jammy-full'`))
				assert.Matches(outBuf.String(), regexp.MustCompile(`Heroku:\s+'heroku/builder:24'`))
			})
		})

		when("print returns an error", func() {
			it("returns that error", func() {
				baseError := errors.New("couldn't write builder")

				builderWriter := newBuilderWriter(errorsForPrint(baseError))
				command := commands.InspectBuilder(
					logger,
					cfg,
					newDefaultBuilderInspector(),
					newWriterFactory(returnsForWriter(builderWriter)),
				)
				command.SetArgs([]string{})

				err := command.Execute()
				assert.ErrorWithMessage(err, "couldn't write builder")
			})
		})

		when("writer factory returns an error", func() {
			it("returns that error", func() {
				baseError := errors.New("invalid output format")

				writerFactory := newWriterFactory(errorsForWriter(baseError))
				command := commands.InspectBuilder(logger, cfg, newDefaultBuilderInspector(), writerFactory)
				command.SetArgs([]string{})

				err := command.Execute()
				assert.ErrorWithMessage(err, "invalid output format")
			})
		})
	})
}
