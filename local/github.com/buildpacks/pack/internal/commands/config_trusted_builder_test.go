package commands_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestTrustedBuilderCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "TrustedBuilderCommands", testTrustedBuilderCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testTrustedBuilderCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command      *cobra.Command
		logger       logging.Logger
		outBuf       bytes.Buffer
		tempPackHome string
		configPath   string
	)

	it.Before(func() {
		var err error

		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		tempPackHome, err = os.MkdirTemp("", "pack-home")
		h.AssertNil(t, err)
		configPath = filepath.Join(tempPackHome, "config.toml")

		command = commands.ConfigTrustedBuilder(logger, config.Config{}, configPath)
		command.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(tempPackHome))
	})

	when("no args", func() {
		it("prints list of trusted builders", func() {
			command.SetArgs([]string{})
			h.AssertNil(t, command.Execute())
			h.AssertContainsAllInOrder(t,
				outBuf,
				"gcr.io/buildpacks/builder:google-22",
				"heroku/builder:20",
				"heroku/builder:22",
				"heroku/builder:24",
				"paketobuildpacks/builder-jammy-base",
				"paketobuildpacks/builder-jammy-full",
				"paketobuildpacks/builder-jammy-tiny",
			)
		})

		it("works with alias of trusted-builders", func() {
			command.SetArgs([]string{})
			h.AssertNil(t, command.Execute())
			h.AssertContainsAllInOrder(t,
				outBuf,
				"gcr.io/buildpacks/builder:google-22",
				"heroku/builder:20",
				"heroku/builder:22",
				"heroku/builder:24",
				"paketobuildpacks/builder-jammy-base",
				"paketobuildpacks/builder-jammy-full",
				"paketobuildpacks/builder-jammy-tiny",
			)
		})
	})

	when("list", func() {
		var args = []string{"list"}

		it("shows suggested builders and locally trusted builder in alphabetical order", func() {
			builderName := "great-builder-" + h.RandString(8)

			command.SetArgs(args)
			h.AssertNil(t, command.Execute())
			h.AssertNotContains(t, outBuf.String(), builderName)
			h.AssertContainsAllInOrder(t,
				outBuf,
				"gcr.io/buildpacks/builder:google-22",
				"heroku/builder:20",
				"heroku/builder:22",
				"heroku/builder:24",
				"paketobuildpacks/builder-jammy-base",
				"paketobuildpacks/builder-jammy-full",
				"paketobuildpacks/builder-jammy-tiny",
			)
			outBuf.Reset()

			configManager := newConfigManager(t, configPath)
			command = commands.ConfigTrustedBuilder(logger, configManager.configWithTrustedBuilders(builderName), configPath)
			command.SetArgs(args)
			h.AssertNil(t, command.Execute())

			h.AssertContainsAllInOrder(t,
				outBuf,
				"gcr.io/buildpacks/builder:google-22",
				builderName,
				"heroku/builder:20",
				"heroku/builder:22",
				"heroku/builder:24",
				"paketobuildpacks/builder-jammy-base",
				"paketobuildpacks/builder-jammy-full",
				"paketobuildpacks/builder-jammy-tiny",
			)
		})
	})

	when("add", func() {
		var args = []string{"add"}
		when("no builder is provided", func() {
			it("prints usage", func() {
				command.SetArgs(args)
				h.AssertError(t, command.Execute(), "accepts 1 arg(s)")
			})
		})

		when("can't write to config path", func() {
			it("fails", func() {
				tempPath := filepath.Join(tempPackHome, "non-existent-file.toml")
				h.AssertNil(t, os.WriteFile(tempPath, []byte("something"), 0111))
				command = commands.ConfigTrustedBuilder(logger, config.Config{}, tempPath)
				command.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))
				command.SetArgs(append(args, "some-builder"))
				h.AssertError(t, command.Execute(), "writing config")
			})
		})

		when("builder is provided", func() {
			when("builder is not already trusted", func() {
				it("updates the config", func() {
					command.SetArgs(append(args, "some-builder"))
					h.AssertNil(t, command.Execute())

					b, err := os.ReadFile(configPath)
					h.AssertNil(t, err)
					h.AssertContains(t, string(b), `[[trusted-builders]]
  name = "some-builder"`)
				})
			})

			when("builder is already trusted", func() {
				it("does nothing", func() {
					command.SetArgs(append(args, "some-already-trusted-builder"))
					h.AssertNil(t, command.Execute())
					oldContents, err := os.ReadFile(configPath)
					h.AssertNil(t, err)

					command.SetArgs(append(args, "some-already-trusted-builder"))
					h.AssertNil(t, command.Execute())

					newContents, err := os.ReadFile(configPath)
					h.AssertNil(t, err)
					h.AssertEq(t, newContents, oldContents)
				})
			})

			when("builder is a suggested builder", func() {
				it("does nothing", func() {
					h.AssertNil(t, os.WriteFile(configPath, []byte(""), os.ModePerm))

					command.SetArgs(append(args, "paketobuildpacks/builder-jammy-base"))
					h.AssertNil(t, command.Execute())
					oldContents, err := os.ReadFile(configPath)
					h.AssertNil(t, err)
					h.AssertEq(t, string(oldContents), "")
				})
			})
		})
	})

	when("remove", func() {
		var (
			args          = []string{"remove"}
			configManager configManager
		)

		it.Before(func() {
			configManager = newConfigManager(t, configPath)
		})

		when("no builder is provided", func() {
			it("prints usage", func() {
				cfg := configManager.configWithTrustedBuilders()
				command := commands.ConfigTrustedBuilder(logger, cfg, configPath)
				command.SetArgs(args)
				command.SetOut(&outBuf)

				err := command.Execute()
				h.AssertError(t, err, "accepts 1 arg(s), received 0")
				h.AssertContains(t, outBuf.String(), "Usage:")
			})
		})

		when("builder is already trusted", func() {
			it("removes builder from the config", func() {
				builderName := "some-builder"

				cfg := configManager.configWithTrustedBuilders(builderName)
				command := commands.ConfigTrustedBuilder(logger, cfg, configPath)
				command.SetArgs(append(args, builderName))

				h.AssertNil(t, command.Execute())

				b, err := os.ReadFile(configPath)
				h.AssertNil(t, err)
				h.AssertNotContains(t, string(b), builderName)

				h.AssertContains(t,
					outBuf.String(),
					fmt.Sprintf("Builder %s is no longer trusted", style.Symbol(builderName)),
				)
			})

			it("removes only the named builder when multiple builders are trusted", func() {
				untrustBuilder := "stop/trusting:me"
				stillTrustedBuilder := "very/safe/builder"

				cfg := configManager.configWithTrustedBuilders(untrustBuilder, stillTrustedBuilder)
				command := commands.ConfigTrustedBuilder(logger, cfg, configPath)
				command.SetArgs(append(args, untrustBuilder))

				h.AssertNil(t, command.Execute())

				b, err := os.ReadFile(configPath)
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), stillTrustedBuilder)
				h.AssertNotContains(t, string(b), untrustBuilder)
			})
		})

		when("builder wasn't already trusted", func() {
			it("does nothing and reports builder wasn't trusted", func() {
				neverTrustedBuilder := "never/trusted-builder"
				stillTrustedBuilder := "very/safe/builder"

				cfg := configManager.configWithTrustedBuilders(stillTrustedBuilder)
				command := commands.ConfigTrustedBuilder(logger, cfg, configPath)
				command.SetArgs(append(args, neverTrustedBuilder))

				h.AssertNil(t, command.Execute())

				b, err := os.ReadFile(configPath)
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), stillTrustedBuilder)
				h.AssertNotContains(t, string(b), neverTrustedBuilder)

				h.AssertContains(t,
					outBuf.String(),
					fmt.Sprintf("Builder %s wasn't trusted", style.Symbol(neverTrustedBuilder)),
				)
			})
		})

		when("builder is a suggested builder", func() {
			it("does nothing and reports that ", func() {
				builder := "paketobuildpacks/builder-jammy-base"
				command := commands.ConfigTrustedBuilder(logger, config.Config{}, configPath)
				command.SetArgs(append(args, builder))

				err := command.Execute()
				h.AssertError(t, err, fmt.Sprintf("Builder %s is a known trusted builder. Currently pack doesn't support making these builders untrusted", style.Symbol(builder)))
			})
		})
	})
}
