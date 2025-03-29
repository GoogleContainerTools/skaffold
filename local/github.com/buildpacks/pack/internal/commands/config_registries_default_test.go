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

func TestConfigRegistriesDefault(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "ConfigRegistriesDefaultCommand", testConfigRegistriesDefaultCommand, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testConfigRegistriesDefaultCommand(t *testing.T, when spec.G, it spec.S) {
	when("#ConfigRegistriesDefault", func() {
		var (
			tmpDir     string
			configFile string
			outBuf     bytes.Buffer
			logger     = logging.NewLogWithWriters(&outBuf, &outBuf)
			assert     = h.NewAssertionManager(t)
		)

		it.Before(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "pack-home-*")
			assert.Nil(err)

			configFile = filepath.Join(tmpDir, "config.toml")
		})

		it.After(func() {
			_ = os.RemoveAll(tmpDir)
		})

		when("list", func() {
			it("returns official if none is set", func() {
				command := commands.ConfigRegistriesDefault(logger, config.Config{}, configFile)
				command.SetArgs([]string{})
				assert.Succeeds(command.Execute())

				assert.Contains(outBuf.String(), "official")
			})

			it("returns the default registry if one is set", func() {
				command := commands.ConfigRegistriesDefault(logger, config.Config{DefaultRegistryName: "some-registry"}, configFile)
				command.SetArgs([]string{})
				assert.Succeeds(command.Execute())

				assert.Contains(outBuf.String(), "some-registry")
			})
		})

		when("set default", func() {
			it("should set the default registry", func() {
				cfg := config.Config{
					Registries: []config.Registry{
						{
							Name: "myregistry",
							URL:  "https://github.com/buildpacks/registry-index",
							Type: "github",
						},
					},
				}
				command := commands.ConfigRegistriesDefault(logger, cfg, configFile)
				command.SetArgs([]string{"myregistry"})
				assert.Succeeds(command.Execute())

				cfg, err := config.Read(configFile)
				assert.Nil(err)

				assert.Equal(cfg.DefaultRegistryName, "myregistry")
			})

			it("should fail if no corresponding registry exists", func() {
				command := commands.ConfigRegistriesDefault(logger, config.Config{}, configFile)
				command.SetArgs([]string{"myregistry"})
				assert.Error(command.Execute())

				output := outBuf.String()
				assert.Contains(output, "no registry with the name 'myregistry' exists")
			})

			it("returns clear error if fails to write", func() {
				assert.Nil(os.WriteFile(configFile, []byte("something"), 0001))
				cfg := config.Config{
					Registries: []config.Registry{
						{
							Name: "myregistry",
							URL:  "https://github.com/buildpacks/registry-index",
							Type: "github",
						},
					},
				}
				command := commands.ConfigRegistriesDefault(logger, cfg, configFile)
				command.SetArgs([]string{"myregistry"})
				assert.ErrorContains(command.Execute(), "writing config to")
			})
		})

		when("--unset", func() {
			it("should unset the default registry, if set", func() {
				command := commands.ConfigRegistriesDefault(logger, config.Config{DefaultRegistryName: "some-registry"}, configFile)
				command.SetArgs([]string{"--unset"})
				assert.Nil(command.Execute())

				cfg, err := config.Read(configFile)
				assert.Nil(err)
				assert.Equal(cfg.DefaultRegistryName, "")
				assert.Contains(outBuf.String(), fmt.Sprintf("Successfully unset default registry %s", style.Symbol("some-registry")))
			})

			it("should return an error if official registry is being unset", func() {
				command := commands.ConfigRegistriesDefault(logger, config.Config{DefaultRegistryName: config.OfficialRegistryName}, configFile)
				command.SetArgs([]string{"--unset"})
				assert.ErrorContains(command.Execute(), fmt.Sprintf("Registry %s is a protected registry", style.Symbol(config.OfficialRegistryName)))
			})

			it("should return a clear message if no registry is set", func() {
				command := commands.ConfigRegistriesDefault(logger, config.Config{}, configFile)
				command.SetArgs([]string{"--unset"})
				assert.ErrorContains(command.Execute(), fmt.Sprintf("Registry %s is a protected registry", style.Symbol(config.OfficialRegistryName)))
			})

			it("returns clear error if fails to write", func() {
				assert.Nil(os.WriteFile(configFile, []byte("something"), 0001))
				command := commands.ConfigRegistriesDefault(logger, config.Config{DefaultRegistryName: "some-registry"}, configFile)
				command.SetArgs([]string{"--unset"})
				assert.ErrorContains(command.Execute(), "writing config to")
			})
		})
	})
}
