package commands_test

import (
	"bytes"
	"errors"
	"regexp"
	"testing"

	pubbldr "github.com/buildpacks/pack/builder"

	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/builder/writer"
	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/fakes"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

var (
	minimalLifecycleDescriptor = builder.LifecycleDescriptor{
		Info: builder.LifecycleInfo{Version: builder.VersionMustParse("3.4")},
		API: builder.LifecycleAPI{
			BuildpackVersion: api.MustParse("1.2"),
			PlatformVersion:  api.MustParse("2.3"),
		},
	}

	expectedLocalRunImages = []config.RunImage{
		{Image: "some/run-image", Mirrors: []string{"first/local", "second/local"}},
	}
	expectedLocalInfo = &client.BuilderInfo{
		Description: "test-local-builder",
		Stack:       "local-stack",
		RunImages:   []pubbldr.RunImageConfig{{Image: "local/image"}},
		Lifecycle:   minimalLifecycleDescriptor,
	}
	expectedRemoteInfo = &client.BuilderInfo{
		Description: "test-remote-builder",
		Stack:       "remote-stack",
		RunImages:   []pubbldr.RunImageConfig{{Image: "remote/image"}},
		Lifecycle:   minimalLifecycleDescriptor,
	}
	expectedLocalDisplay  = "Sample output for local builder"
	expectedRemoteDisplay = "Sample output for remote builder"
	expectedBuilderInfo   = writer.SharedBuilderInfo{
		Name:      "default/builder",
		Trusted:   false,
		IsDefault: true,
	}
)

func TestBuilderInspectCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "BuilderInspectCommand", testBuilderInspectCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testBuilderInspectCommand(t *testing.T, when spec.G, it spec.S) {
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

	when("BuilderInspect", func() {
		var (
			assert = h.NewAssertionManager(t)
		)

		it("passes output of local and remote builders to correct writer", func() {
			builderInspector := newDefaultBuilderInspector()
			builderWriter := newDefaultBuilderWriter()
			builderWriterFactory := newWriterFactory(returnsForWriter(builderWriter))

			command := commands.BuilderInspect(logger, cfg, builderInspector, builderWriterFactory)
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
				command := commands.BuilderInspect(logger, cfg, builderInspector, newWriterFactory(returnsForWriter(writer)))
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
				command := commands.BuilderInspect(logger, cfg, builderInspector, newDefaultWriterFactory())
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
				command := commands.BuilderInspect(logger, cfg, newDefaultBuilderInspector(), writerFactory)
				command.SetArgs([]string{"--output", "json"})

				err := command.Execute()
				assert.Nil(err)

				assert.Equal(writerFactory.ReceivedForKind, "json")
			})
		})

		when("output type is set to toml using the shorthand flag", func() {
			it("passes toml to the writer factory", func() {
				writerFactory := newDefaultWriterFactory()
				command := commands.BuilderInspect(logger, cfg, newDefaultBuilderInspector(), writerFactory)
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

				command := commands.BuilderInspect(logger, cfg, builderInspector, builderWriterFactory)
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

				command := commands.BuilderInspect(logger, cfg, builderInspector, builderWriterFactory)
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

				command := commands.BuilderInspect(
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

				command := commands.BuilderInspect(
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

				command := commands.BuilderInspect(logger, cfg, newDefaultBuilderInspector(), newDefaultWriterFactory())
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
				command := commands.BuilderInspect(
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
				command := commands.BuilderInspect(logger, cfg, newDefaultBuilderInspector(), writerFactory)
				command.SetArgs([]string{})

				err := command.Execute()
				assert.ErrorWithMessage(err, "invalid output format")
			})
		})
	})
}

func newDefaultBuilderInspector() *fakes.FakeBuilderInspector {
	return &fakes.FakeBuilderInspector{
		InfoForLocal:  expectedLocalInfo,
		InfoForRemote: expectedRemoteInfo,
	}
}

func newDefaultBuilderWriter() *fakes.FakeBuilderWriter {
	return &fakes.FakeBuilderWriter{
		PrintForLocal:  expectedLocalDisplay,
		PrintForRemote: expectedRemoteDisplay,
	}
}

func newDefaultWriterFactory() *fakes.FakeBuilderWriterFactory {
	return &fakes.FakeBuilderWriterFactory{
		ReturnForWriter: newDefaultBuilderWriter(),
	}
}

type BuilderWriterModifier func(w *fakes.FakeBuilderWriter)

func errorsForPrint(err error) BuilderWriterModifier {
	return func(w *fakes.FakeBuilderWriter) {
		w.ErrorForPrint = err
	}
}

func newBuilderWriter(modifiers ...BuilderWriterModifier) *fakes.FakeBuilderWriter {
	w := newDefaultBuilderWriter()

	for _, mod := range modifiers {
		mod(w)
	}

	return w
}

type WriterFactoryModifier func(f *fakes.FakeBuilderWriterFactory)

func returnsForWriter(writer writer.BuilderWriter) WriterFactoryModifier {
	return func(f *fakes.FakeBuilderWriterFactory) {
		f.ReturnForWriter = writer
	}
}

func errorsForWriter(err error) WriterFactoryModifier {
	return func(f *fakes.FakeBuilderWriterFactory) {
		f.ErrorForWriter = err
	}
}

func newWriterFactory(modifiers ...WriterFactoryModifier) *fakes.FakeBuilderWriterFactory {
	f := newDefaultWriterFactory()

	for _, mod := range modifiers {
		mod(f)
	}

	return f
}

type BuilderInspectorModifier func(i *fakes.FakeBuilderInspector)

func errorsForLocal(err error) BuilderInspectorModifier {
	return func(i *fakes.FakeBuilderInspector) {
		i.ErrorForLocal = err
	}
}

func errorsForRemote(err error) BuilderInspectorModifier {
	return func(i *fakes.FakeBuilderInspector) {
		i.ErrorForRemote = err
	}
}

func newBuilderInspector(modifiers ...BuilderInspectorModifier) *fakes.FakeBuilderInspector {
	i := newDefaultBuilderInspector()

	for _, mod := range modifiers {
		mod(i)
	}

	return i
}
