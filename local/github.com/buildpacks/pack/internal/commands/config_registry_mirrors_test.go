package commands_test

import (
	"bytes"
	"fmt"
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
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestConfigRegistryMirrors(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ConfigRunImageMirrorsCommand", testConfigRegistryMirrorsCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testConfigRegistryMirrorsCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		cmd          *cobra.Command
		logger       logging.Logger
		outBuf       bytes.Buffer
		tempPackHome string
		configPath   string
		registry1    = "index.docker.io"
		registry2    = "us.gcr.io"
		testMirror1  = "10.0.0.1"
		testMirror2  = "10.0.0.2"
		testCfg      = config.Config{
			RegistryMirrors: map[string]string{
				registry1: testMirror1,
				registry2: testMirror2,
			},
		}
	)

	it.Before(func() {
		var err error
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		tempPackHome, err = os.MkdirTemp("", "pack-home")
		h.AssertNil(t, err)
		configPath = filepath.Join(tempPackHome, "config.toml")

		cmd = commands.ConfigRegistryMirrors(logger, testCfg, configPath)
		cmd.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(tempPackHome))
	})

	when("-h", func() {
		it("prints available commands", func() {
			cmd.SetArgs([]string{"-h"})
			h.AssertNil(t, cmd.Execute())
			output := outBuf.String()
			h.AssertContains(t, output, "Usage:")
			for _, command := range []string{"add", "remove", "list"} {
				h.AssertContains(t, output, command)
			}
		})
	})

	when("no arguments", func() {
		it("lists registry mirrors", func() {
			cmd.SetArgs([]string{})
			h.AssertNil(t, cmd.Execute())
			output := outBuf.String()
			h.AssertContains(t, strings.TrimSpace(output), `Registry Mirrors:`)
			h.AssertContains(t, strings.TrimSpace(output), `index.docker.io: '10.0.0.1'`)
			h.AssertContains(t, strings.TrimSpace(output), `us.gcr.io: '10.0.0.2'`)
		})
	})

	when("add", func() {
		when("no registry is specified", func() {
			it("fails to run", func() {
				cmd.SetArgs([]string{"add"})
				err := cmd.Execute()
				h.AssertError(t, err, "accepts 1 arg")
			})
		})

		when("config path doesn't exist", func() {
			it("fails to run", func() {
				fakePath := filepath.Join(tempPackHome, "not-exist.toml")
				h.AssertNil(t, os.WriteFile(fakePath, []byte("something"), 0001))
				cmd = commands.ConfigRegistryMirrors(logger, config.Config{}, fakePath)
				cmd.SetArgs([]string{"add", registry1, "-m", testMirror1})

				err := cmd.Execute()
				h.AssertError(t, err, "failed to write to")
			})
		})

		when("mirrors are provided", func() {
			it("adds them as mirrors to the config", func() {
				cmd.SetArgs([]string{"add", "asia.gcr.io", "-m", "10.0.0.3"})
				h.AssertNil(t, cmd.Execute())
				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg, config.Config{
					RegistryMirrors: map[string]string{
						registry1:     testMirror1,
						registry2:     testMirror2,
						"asia.gcr.io": "10.0.0.3",
					},
				})
			})

			it("replaces pre-existing mirrors in the config", func() {
				cmd.SetArgs([]string{"add", registry1, "-m", "10.0.0.3"})
				h.AssertNil(t, cmd.Execute())
				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg, config.Config{
					RegistryMirrors: map[string]string{
						registry1: "10.0.0.3",
						registry2: testMirror2,
					},
				})
			})
		})

		when("no mirrors are provided", func() {
			it("preserves old mirrors, and prints helpful message", func() {
				cmd.SetArgs([]string{"add", registry1})
				h.AssertNil(t, cmd.Execute())
				h.AssertContains(t, outBuf.String(), "A registry mirror was not provided")
			})
		})
	})

	when("remove", func() {
		when("no registry is specified", func() {
			it("fails to run", func() {
				cmd.SetArgs([]string{"remove"})
				err := cmd.Execute()
				h.AssertError(t, err, "accepts 1 arg")
			})
		})

		when("registry provided isn't present", func() {
			it("prints a clear message", func() {
				fakeImage := "not-set-image"
				cmd.SetArgs([]string{"remove", fakeImage})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertContains(t, output, fmt.Sprintf("No registry mirror has been set for %s", style.Symbol(fakeImage)))
			})
		})

		when("config path doesn't exist", func() {
			it("fails to run", func() {
				fakePath := filepath.Join(tempPackHome, "not-exist.toml")
				h.AssertNil(t, os.WriteFile(fakePath, []byte("something"), 0001))
				cmd = commands.ConfigRegistryMirrors(logger, testCfg, fakePath)
				cmd.SetArgs([]string{"remove", registry1})

				err := cmd.Execute()
				h.AssertError(t, err, "failed to write to")
			})
		})

		when("registry is provided", func() {
			it("removes the given registry", func() {
				cmd.SetArgs([]string{"remove", registry1})
				h.AssertNil(t, cmd.Execute())
				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.RegistryMirrors, map[string]string{
					registry2: testMirror2,
				})
			})
		})
	})

	when("list", func() {
		when("mirrors were previously set", func() {
			it("lists registry mirrors", func() {
				cmd.SetArgs([]string{"list"})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertContains(t, output, registry1)
				h.AssertContains(t, output, testMirror1)
				h.AssertContains(t, output, registry2)
				h.AssertContains(t, output, testMirror2)
			})
		})

		when("no registry mirrors were set", func() {
			it("prints a clear message", func() {
				cmd = commands.ConfigRegistryMirrors(logger, config.Config{}, configPath)
				cmd.SetArgs([]string{"list"})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertNotContains(t, output, registry1)
				h.AssertNotContains(t, output, testMirror1)
				h.AssertNotContains(t, output, registry2)
				h.AssertNotContains(t, output, testMirror2)

				h.AssertContains(t, output, "No registry mirrors have been set")
			})
		})
	})
}
