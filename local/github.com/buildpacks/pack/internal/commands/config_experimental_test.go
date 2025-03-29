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

func TestConfigExperimental(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ConfigExperimentalCommand", testConfigExperimental, spec.Random(), spec.Report(report.Terminal{}))
}

func testConfigExperimental(t *testing.T, when spec.G, it spec.S) {
	var (
		cmd          *cobra.Command
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

		cmd = commands.ConfigExperimental(logger, config.Config{}, configPath)
		cmd.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(tempPackHome))
	})

	when("#ConfigExperimental", func() {
		when("list values", func() {
			it("prints a clear message if false", func() {
				cmd.SetArgs([]string{})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertContains(t, output, "Experimental features aren't currently enabled")
			})

			it("prints a clear message if true", func() {
				cmd = commands.ConfigExperimental(logger, config.Config{Experimental: true}, configPath)
				cmd.SetArgs([]string{})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertContains(t, output, "Experimental features are enabled!")
			})
		})

		when("set", func() {
			it("sets true if provided", func() {
				cmd.SetArgs([]string{"true"})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertContains(t, output, "Experimental features enabled")
				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.Experimental, true)

				// oci layout repo is configured
				layoutDir := filepath.Join(filepath.Dir(configPath), "layout-repo")
				h.AssertEq(t, cfg.LayoutRepositoryDir, layoutDir)
			})

			it("sets false if provided", func() {
				cmd.SetArgs([]string{"false"})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertContains(t, output, "Experimental features disabled")
				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.Experimental, false)

				// oci layout repo is cleaned
				h.AssertEq(t, cfg.LayoutRepositoryDir, "")
			})

			it("returns error if invalid value provided", func() {
				cmd.SetArgs([]string{"disable-me"})
				h.AssertError(t, cmd.Execute(), fmt.Sprintf("invalid value %s provided", style.Symbol("disable-me")))
				// output := outBuf.String()
				// h.AssertContains(t, output, "Experimental features disabled.")
				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.Experimental, false)
			})
		})
	})
}
