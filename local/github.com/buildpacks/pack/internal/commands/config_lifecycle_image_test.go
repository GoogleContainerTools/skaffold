package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestConfigLifecyleImage(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ConfigLifecycleImage", testConfigLifecycleImageCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testConfigLifecycleImageCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command      *cobra.Command
		logger       logging.Logger
		outBuf       bytes.Buffer
		tempPackHome string
		configFile   string
		assert       = h.NewAssertionManager(t)
		cfg          = config.Config{}
	)

	it.Before(func() {
		var err error
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		tempPackHome, err = os.MkdirTemp("", "pack-home")
		h.AssertNil(t, err)
		configFile = filepath.Join(tempPackHome, "config.toml")

		command = commands.ConfigLifecycleImage(logger, cfg, configFile)
		command.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(tempPackHome))
	})

	when("#ConfigLifecycleImage", func() {
		when("list", func() {
			when("no custom lifecycle image was set", func() {
				it("lists the default", func() {
					command.SetArgs([]string{})

					h.AssertNil(t, command.Execute())

					assert.Contains(outBuf.String(), config.DefaultLifecycleImageRepo)
				})
			})
			when("custom lifecycle-image was set", func() {
				it("lists the custom image", func() {
					cfg.LifecycleImage = "custom-lifecycle/image:v1"
					command = commands.ConfigLifecycleImage(logger, cfg, configFile)
					command.SetArgs([]string{})

					h.AssertNil(t, command.Execute())

					assert.Contains(outBuf.String(), "custom-lifecycle/image:v1")
				})
			})
		})
		when("set", func() {
			when("custom lifecycle-image provided is the same as configured lifecycle-image", func() {
				it("provides a helpful message", func() {
					cfg.LifecycleImage = "custom-lifecycle/image:v1"
					command = commands.ConfigLifecycleImage(logger, cfg, configFile)
					command.SetArgs([]string{"custom-lifecycle/image:v1"})

					h.AssertNil(t, command.Execute())

					output := outBuf.String()
					h.AssertEq(t, strings.TrimSpace(output), `Custom lifecycle image is already set to 'custom-lifecycle/image:v1'`)
				})
				it("it does not change the configured", func() {
					command = commands.ConfigLifecycleImage(logger, cfg, configFile)
					command.SetArgs([]string{"custom-lifecycle/image:v1"})
					assert.Succeeds(command.Execute())

					readCfg, err := config.Read(configFile)
					assert.Nil(err)
					assert.Equal(readCfg.LifecycleImage, "custom-lifecycle/image:v1")

					command = commands.ConfigLifecycleImage(logger, readCfg, configFile)
					command.SetArgs([]string{"custom-lifecycle/image:v1"})
					assert.Succeeds(command.Execute())

					readCfg, err = config.Read(configFile)
					assert.Nil(err)
					assert.Equal(readCfg.LifecycleImage, "custom-lifecycle/image:v1")
				})
			})

			when("valid lifecycle-image is specified", func() {
				it("sets the lifecycle-image in config", func() {
					command.SetArgs([]string{"custom-lifecycle/image:v1"})
					assert.Succeeds(command.Execute())

					readCfg, err := config.Read(configFile)
					assert.Nil(err)
					assert.Equal(readCfg.LifecycleImage, "custom-lifecycle/image:v1")
				})
				it("returns clear error if fails to write", func() {
					assert.Nil(os.WriteFile(configFile, []byte("something"), 0001))
					command := commands.ConfigLifecycleImage(logger, cfg, configFile)
					command.SetArgs([]string{"custom-lifecycle/image:v1"})
					assert.ErrorContains(command.Execute(), "failed to write to config at")
				})
			})
			when("invalid lifecycle-image is specified", func() {
				it("returns an error", func() {
					command.SetArgs([]string{"custom$1#-lifecycle/image-repo"})
					err := command.Execute()
					h.AssertError(t, err, `Invalid image name`)
				})
				it("returns clear error if fails to write", func() {
					assert.Nil(os.WriteFile(configFile, []byte("something"), 0001))
					command := commands.ConfigLifecycleImage(logger, cfg, configFile)
					command.SetArgs([]string{"custom-lifecycle/image:v1"})
					assert.ErrorContains(command.Execute(), "failed to write to config at")
				})
			})
		})
		when("unset", func() {
			when("the custom lifecycle image is set", func() {
				it("removes set lifecycle image and resets to default lifecycle image", func() {
					command = commands.ConfigLifecycleImage(logger, cfg, configFile)
					command.SetArgs([]string{"custom-lifecycle/image:v1"})
					assert.Succeeds(command.Execute())

					readCfg, err := config.Read(configFile)
					assert.Nil(err)
					assert.Equal(readCfg.LifecycleImage, "custom-lifecycle/image:v1")

					command = commands.ConfigLifecycleImage(logger, readCfg, configFile)
					command.SetArgs([]string{"--unset"})
					assert.Succeeds(command.Execute())

					readCfg, err = config.Read(configFile)
					assert.Nil(err)
					assert.Equal(readCfg.LifecycleImage, "")
				})
				it("returns clear error if fails to write", func() {
					assert.Nil(os.WriteFile(configFile, []byte("something"), 0001))
					command := commands.ConfigLifecycleImage(logger, config.Config{LifecycleImage: "custom-lifecycle/image:v1"}, configFile)
					command.SetArgs([]string{"--unset"})
					assert.ErrorContains(command.Execute(), "failed to write to config at")
				})
			})
			when("the custom lifecycle image is not set", func() {
				it("returns clear message that no custom lifecycle image is set", func() {
					command.SetArgs([]string{"--unset"})
					assert.Succeeds(command.Execute())
					output := outBuf.String()
					h.AssertEq(t, strings.TrimSpace(output), `No custom lifecycle image was set.`)
				})
			})
		})
		when("--unset and lifecycle image to set is provided", func() {
			it("errors", func() {
				command.SetArgs([]string{
					"custom-lifecycle/image:v1",
					"--unset",
				})
				err := command.Execute()
				h.AssertError(t, err, `lifecycle image and --unset cannot be specified simultaneously`)
			})
		})
	})
}
