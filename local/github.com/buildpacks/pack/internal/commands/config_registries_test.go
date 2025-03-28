package commands_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
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

func TestConfigRegistries(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ConfigRegistriesCommand", testConfigRegistries, spec.Random(), spec.Report(report.Terminal{}))
}

func testConfigRegistries(t *testing.T, when spec.G, it spec.S) {
	var (
		cmd               *cobra.Command
		logger            logging.Logger
		cfgWithRegistries config.Config
		outBuf            bytes.Buffer
		tempPackHome      string
		configPath        string
		assert            = h.NewAssertionManager(t)
	)

	it.Before(func() {
		var err error

		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		tempPackHome, err = os.MkdirTemp("", "pack-home")
		assert.Nil(err)
		configPath = filepath.Join(tempPackHome, "config.toml")

		cmd = commands.ConfigRegistries(logger, config.Config{}, configPath)
		cmd.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))

		cfgWithRegistries = config.Config{
			DefaultRegistryName: "private registry",
			Registries: []config.Registry{
				{
					Name: "public registry",
					Type: "github",
					URL:  "https://github.com/buildpacks/public-registry",
				},
				{
					Name: "private registry",
					Type: "github",
					URL:  "https://github.com/buildpacks/private-registry",
				},
				{
					Name: "personal registry",
					Type: "github",
					URL:  "https://github.com/buildpacks/personal-registry",
				},
			},
		}
	})

	it.After(func() {
		assert.Nil(os.RemoveAll(tempPackHome))
	})

	when("-h", func() {
		it("prints help text", func() {
			cmd.SetArgs([]string{"-h"})
			assert.Nil(cmd.Execute())
			output := outBuf.String()
			assert.Contains(output, "Usage:")
			for _, command := range []string{"add", "remove", "list", "default"} {
				assert.Contains(output, command)
			}
		})
	})

	when("no args", func() {
		it("calls list", func() {
			logger = logging.NewLogWithWriters(&outBuf, &outBuf)
			cfgWithRegistries = config.Config{
				DefaultRegistryName: "private registry",
				Registries: []config.Registry{
					{
						Name: "public registry",
						Type: "github",
						URL:  "https://github.com/buildpacks/public-registry",
					},
					{
						Name: "private registry",
						Type: "github",
						URL:  "https://github.com/buildpacks/private-registry",
					},
					{
						Name: "personal registry",
						Type: "github",
						URL:  "https://github.com/buildpacks/personal-registry",
					},
				},
			}
			cmd = commands.ConfigRegistries(logger, cfgWithRegistries, configPath)
			cmd.SetArgs([]string{})

			assert.Nil(cmd.Execute())
			assert.Contains(outBuf.String(), "public registry")
			assert.Contains(outBuf.String(), "* private registry")
			assert.Contains(outBuf.String(), "personal registry")
		})
	})

	when("list", func() {
		var args = []string{"list"}
		it.Before(func() {
			logger = logging.NewLogWithWriters(&outBuf, &outBuf)
			cmd = commands.ConfigRegistries(logger, cfgWithRegistries, configPath)
			cmd.SetArgs(args)
		})

		it("should list all registries", func() {
			assert.Nil(cmd.Execute())

			assert.Contains(outBuf.String(), "public registry")
			assert.Contains(outBuf.String(), "* private registry")
			assert.Contains(outBuf.String(), "personal registry")
		})

		it("should list registries in verbose mode", func() {
			logger = logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
			cmd = commands.ConfigRegistries(logger, cfgWithRegistries, configPath)
			cmd.SetArgs(args)
			assert.Nil(cmd.Execute())

			assert.Contains(outBuf.String(), "public registry")
			assert.Contains(outBuf.String(), "https://github.com/buildpacks/public-registry")

			assert.Contains(outBuf.String(), "* private registry")
			assert.Contains(outBuf.String(), "https://github.com/buildpacks/private-registry")

			assert.Contains(outBuf.String(), "personal registry")
			assert.Contains(outBuf.String(), "https://github.com/buildpacks/personal-registry")

			assert.Contains(outBuf.String(), "official")
			assert.Contains(outBuf.String(), "https://github.com/buildpacks/registry-index")
		})

		it("should indicate official as the default registry by default", func() {
			cfgWithRegistries.DefaultRegistryName = ""
			cmd = commands.ConfigRegistries(logger, cfgWithRegistries, configPath)
			cmd.SetArgs(args)

			assert.Nil(cmd.Execute())

			assert.Contains(outBuf.String(), "* official")
			assert.Contains(outBuf.String(), "public registry")
			assert.Contains(outBuf.String(), "private registry")
			assert.Contains(outBuf.String(), "personal registry")
		})

		it("should use official when no registries are defined", func() {
			cmd = commands.ConfigRegistries(logger, config.Config{}, configPath)
			cmd.SetArgs(args)

			assert.Nil(cmd.Execute())

			assert.Contains(outBuf.String(), "* official")
		})
	})

	when("add", func() {
		var (
			args = []string{"add", "bp", "https://github.com/buildpacks/registry-index/"}
		)

		when("add buildpack registry", func() {
			it("adds to registry list", func() {
				cmd.SetArgs(args)
				assert.Succeeds(cmd.Execute())

				cfg, err := config.Read(configPath)
				assert.Nil(err)
				assert.Equal(len(cfg.Registries), 1)
				assert.Equal(cfg.Registries[0].Name, "bp")
				assert.Equal(cfg.Registries[0].Type, "github")
				assert.Equal(cfg.Registries[0].URL, "https://github.com/buildpacks/registry-index/")
				assert.Equal(cfg.DefaultRegistryName, "")
			})
		})

		when("default is true", func() {
			it("sets newly added registry as the default", func() {
				cmd.SetArgs(append(args, "--default"))
				assert.Succeeds(cmd.Execute())

				cfg, err := config.Read(configPath)
				assert.Nil(err)
				assert.Equal(len(cfg.Registries), 1)
				assert.Equal(cfg.Registries[0].Name, "bp")
				assert.Equal(cfg.DefaultRegistryName, "bp")
			})
		})

		when("validation", func() {
			it("fails with missing args", func() {
				cmd.SetOut(io.Discard)
				cmd.SetArgs([]string{"add"})
				err := cmd.Execute()
				assert.ErrorContains(err, "accepts 2 arg")
			})

			it("should validate type", func() {
				cmd.SetArgs(append(args, "--type=bogus"))
				assert.Error(cmd.Execute())

				output := outBuf.String()
				assert.Contains(output, "'bogus' is not a valid type. Supported types are: 'git', 'github'.")
			})

			it("should throw error when registry already exists", func() {
				command := commands.ConfigRegistries(logger, config.Config{
					Registries: []config.Registry{
						{
							Name: "bp",
							Type: "github",
							URL:  "https://github.com/buildpacks/registry-index/",
						},
					},
				}, configPath)
				command.SetArgs(args)
				assert.Error(command.Execute())

				output := outBuf.String()
				assert.Contains(output, "Buildpack registry 'bp' already exists.")
			})

			it("should throw error when registry name is official", func() {
				cmd.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))
				cmd.SetErr(&outBuf)
				cmd.SetArgs([]string{"add", "official", "https://github.com/buildpacks/registry-index/", "--type=github"})

				assert.Error(cmd.Execute())

				output := outBuf.String()
				assert.Contains(output, "'official' is a reserved registry, please provide a different name")
			})

			it("returns clear error if fails to write", func() {
				assert.Nil(os.WriteFile(configPath, []byte("something"), 0001))
				cmd.SetArgs(args)
				assert.ErrorContains(cmd.Execute(), "writing config to")
			})
		})
	})

	when("remove", func() {
		it.Before(func() {
			cmd = commands.ConfigRegistries(logger, cfgWithRegistries, configPath)
		})

		it("should remove the registry", func() {
			cmd.SetArgs([]string{"remove", "public registry"})
			assert.Succeeds(cmd.Execute())

			newCfg, err := config.Read(configPath)
			assert.Nil(err)

			assert.Equal(newCfg, config.Config{
				DefaultRegistryName: "private registry",
				Registries: []config.Registry{
					{
						Name: "personal registry",
						Type: "github",
						URL:  "https://github.com/buildpacks/personal-registry",
					},
					{
						Name: "private registry",
						Type: "github",
						URL:  "https://github.com/buildpacks/private-registry",
					},
				},
			})
		})

		it("should remove the registry and matching default registry name", func() {
			cmd.SetArgs([]string{"remove", "private registry"})
			assert.Succeeds(cmd.Execute())

			newCfg, err := config.Read(configPath)
			assert.Nil(err)

			assert.Equal(newCfg, config.Config{
				DefaultRegistryName: config.OfficialRegistryName,
				Registries: []config.Registry{
					{
						Name: "public registry",
						Type: "github",
						URL:  "https://github.com/buildpacks/public-registry",
					},
					{
						Name: "personal registry",
						Type: "github",
						URL:  "https://github.com/buildpacks/personal-registry",
					},
				},
			})
		})

		it("should return error when registry does NOT already exist", func() {
			cmd.SetArgs([]string{"remove", "missing-registry"})
			assert.Error(cmd.Execute())

			output := outBuf.String()
			assert.Contains(output, "registry 'missing-registry' does not exist")
		})

		it("should throw error when registry name is official", func() {
			cmd.SetArgs([]string{"remove", "official"})
			assert.Error(cmd.Execute())

			output := outBuf.String()
			assert.Contains(output, "'official' is a reserved registry name, please provide a different registry")
		})

		it("returns clear error if fails to write", func() {
			assert.Nil(os.WriteFile(configPath, []byte("something"), 0001))
			cmd.SetArgs([]string{"remove", "public registry"})
			assert.ErrorContains(cmd.Execute(), "writing config to")
		})
	})
}
