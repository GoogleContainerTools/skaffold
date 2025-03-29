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

func TestConfigRunImageMirrors(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ConfigRunImageMirrorsCommand", testConfigRunImageMirrorsCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testConfigRunImageMirrorsCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		cmd          *cobra.Command
		logger       logging.Logger
		outBuf       bytes.Buffer
		tempPackHome string
		configPath   string
		runImage     = "test/image"
		testMirror1  = "example.com/some/run1"
		testMirror2  = "example.com/some/run2"
		testCfg      = config.Config{
			Experimental: true,
			RunImages: []config.RunImage{{
				Image:   runImage,
				Mirrors: []string{testMirror1, testMirror2},
			}},
		}
		expandedCfg = config.Config{
			Experimental: true,
			RunImages: append(testCfg.RunImages, config.RunImage{
				Image:   "new-image",
				Mirrors: []string{"some-mirror1", "some-mirror2"},
			}),
		}
	)

	it.Before(func() {
		var err error
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		tempPackHome, err = os.MkdirTemp("", "pack-home")
		h.AssertNil(t, err)
		configPath = filepath.Join(tempPackHome, "config.toml")

		cmd = commands.ConfigRunImagesMirrors(logger, testCfg, configPath)
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
		it("lists run image mirrors", func() {
			cmd.SetArgs([]string{})
			h.AssertNil(t, cmd.Execute())
			output := outBuf.String()
			h.AssertEq(t, strings.TrimSpace(output), `Run Image Mirrors:
  'test/image':
    example.com/some/run1
    example.com/some/run2`)
		})
	})

	when("add", func() {
		when("no run image is specified", func() {
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
				cmd = commands.ConfigRunImagesMirrors(logger, config.Config{}, fakePath)
				cmd.SetArgs([]string{"add", runImage, "-m", testMirror1})

				err := cmd.Execute()
				h.AssertError(t, err, "failed to write to")
			})
		})

		when("mirrors are provided", func() {
			it("adds them as mirrors to the config", func() {
				cmd.SetArgs([]string{"add", runImage, "-m", testMirror1, "-m", testMirror2})
				h.AssertNil(t, cmd.Execute())
				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg, testCfg)
				// This ensures that there are no dups
				h.AssertEq(t, len(cfg.RunImages[0].Mirrors), 2)
			})
		})

		when("no mirrors are provided", func() {
			it("preserves old mirrors, and prints helpful message", func() {
				cmd.SetArgs([]string{"add", runImage})
				h.AssertNil(t, cmd.Execute())
				h.AssertContains(t, outBuf.String(), "No run image mirrors were provided")
			})
		})
	})

	when("remove", func() {
		when("no run image is specified", func() {
			it("fails to run", func() {
				cmd.SetArgs([]string{"remove"})
				err := cmd.Execute()
				h.AssertError(t, err, "accepts 1 arg")
			})
		})

		when("run image provided isn't present", func() {
			it("prints a clear message", func() {
				fakeImage := "not-set-image"
				cmd.SetArgs([]string{"remove", fakeImage})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertContains(t, output, fmt.Sprintf("No run image mirrors have been set for %s", style.Symbol(fakeImage)))
			})
		})

		when("config path doesn't exist", func() {
			it("fails to run", func() {
				fakePath := filepath.Join(tempPackHome, "not-exist.toml")
				h.AssertNil(t, os.WriteFile(fakePath, []byte("something"), 0001))
				cmd = commands.ConfigRunImagesMirrors(logger, testCfg, fakePath)
				cmd.SetArgs([]string{"remove", runImage, "-m", testMirror1})

				err := cmd.Execute()
				h.AssertError(t, err, "failed to write to")
			})
		})

		when("mirrors are provided", func() {
			it("removes them for the given run image", func() {
				cmd.SetArgs([]string{"remove", runImage, "-m", testMirror2})
				h.AssertNil(t, cmd.Execute())
				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.RunImages, []config.RunImage{{
					Image:   runImage,
					Mirrors: []string{testMirror1},
				}})
			})

			it("removes the image if all mirrors are removed", func() {
				cmd.SetArgs([]string{"remove", runImage, "-m", testMirror1, "-m", testMirror2})
				h.AssertNil(t, cmd.Execute())
				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.RunImages, []config.RunImage{})
			})
		})

		when("no mirrors are provided", func() {
			it("removes all mirrors for the given run image", func() {
				cmd.SetArgs([]string{"remove", runImage})
				h.AssertNil(t, cmd.Execute())

				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.Experimental, testCfg.Experimental)
				h.AssertEq(t, cfg.RunImages, []config.RunImage{})
			})

			it("preserves all mirrors aside from the given run image", func() {
				cmd = commands.ConfigRunImagesMirrors(logger, expandedCfg, configPath)
				cmd.SetArgs([]string{"remove", runImage})
				h.AssertNil(t, cmd.Execute())

				cfg, err := config.Read(configPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.Experimental, testCfg.Experimental)
				h.AssertNotEq(t, cfg.RunImages, []config.RunImage{})
				h.AssertEq(t, cfg.RunImages, []config.RunImage{expandedCfg.RunImages[1]})
			})
		})
	})

	when("list", func() {
		when("mirrors were previously set", func() {
			it("lists run image mirrors", func() {
				cmd.SetArgs([]string{"list"})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertContains(t, output, runImage)
				h.AssertContains(t, output, testMirror1)
				h.AssertContains(t, output, testMirror2)
			})
		})

		when("no run image mirrors were set", func() {
			it("prints a clear message", func() {
				cmd = commands.ConfigRunImagesMirrors(logger, config.Config{}, configPath)
				cmd.SetArgs([]string{"list"})
				h.AssertNil(t, cmd.Execute())
				output := outBuf.String()
				h.AssertNotContains(t, output, runImage)
				h.AssertNotContains(t, output, testMirror1)

				h.AssertContains(t, output, "No run image mirrors have been set")
			})
		})

		when("run image provided", func() {
			when("mirrors are set", func() {
				it("returns image mirrors", func() {
					cmd = commands.ConfigRunImagesMirrors(logger, expandedCfg, configPath)
					cmd.SetArgs([]string{"list", "new-image"})
					h.AssertNil(t, cmd.Execute())
					output := outBuf.String()
					h.AssertNotContains(t, output, runImage)
					h.AssertNotContains(t, output, testMirror1)
					h.AssertContains(t, output, "new-image")
					h.AssertContains(t, output, "some-mirror1")
					h.AssertContains(t, output, "some-mirror2")
				})
			})

			when("mirrors aren't set", func() {
				it("prints a clear message", func() {
					fakeImage := "not-set-image"
					cmd.SetArgs([]string{"list", fakeImage})
					h.AssertNil(t, cmd.Execute())
					output := outBuf.String()
					h.AssertNotContains(t, output, runImage)
					h.AssertNotContains(t, output, testMirror1)
					h.AssertContains(t, output, fmt.Sprintf("No run image mirrors have been set for %s", style.Symbol(fakeImage)))
				})
			})
		})
	})
}
