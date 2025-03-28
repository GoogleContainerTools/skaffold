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

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestUntrustBuilderCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Commands", testUntrustBuilderCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testUntrustBuilderCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		logger        logging.Logger
		outBuf        bytes.Buffer
		tempPackHome  string
		configPath    string
		configManager configManager
	)

	it.Before(func() {
		var err error

		logger = logging.NewLogWithWriters(&outBuf, &outBuf)

		tempPackHome, err = os.MkdirTemp("", "pack-home")
		h.AssertNil(t, err)
		configPath = filepath.Join(tempPackHome, "config.toml")
		configManager = newConfigManager(t, configPath)
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(tempPackHome))
	})

	when("#UntrustBuilder", func() {
		when("no builder is provided", func() {
			it("prints usage", func() {
				cfg := configManager.configWithTrustedBuilders()
				command := commands.UntrustBuilder(logger, cfg, configPath)
				command.SetArgs([]string{})
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
				command := commands.UntrustBuilder(logger, cfg, configPath)
				command.SetArgs([]string{builderName})

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
				command := commands.UntrustBuilder(logger, cfg, configPath)
				command.SetArgs([]string{untrustBuilder})

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
				command := commands.UntrustBuilder(logger, cfg, configPath)
				command.SetArgs([]string{neverTrustedBuilder})

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
				command := commands.UntrustBuilder(logger, config.Config{}, configPath)
				command.SetArgs([]string{builder})

				err := command.Execute()
				h.AssertError(t, err, fmt.Sprintf("Builder %s is a known trusted builder. Currently pack doesn't support making these builders untrusted", style.Symbol(builder)))
			})
		})
	})
}
