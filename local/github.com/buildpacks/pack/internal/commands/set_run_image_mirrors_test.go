package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestSetRunImageMirrorsCommand(t *testing.T) {
	spec.Run(t, "Commands", testSetRunImageMirrorsCommand, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testSetRunImageMirrorsCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command      *cobra.Command
		logger       logging.Logger
		outBuf       bytes.Buffer
		cfg          config.Config
		tempPackHome string
		cfgPath      string
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		cfg = config.Config{}
		var err error
		tempPackHome, err = os.MkdirTemp("", "pack-home")
		h.AssertNil(t, err)
		cfgPath = filepath.Join(tempPackHome, "config.toml")

		command = commands.SetRunImagesMirrors(logger, cfg, cfgPath)
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(tempPackHome))
	})

	when("#SetRunImageMirrors", func() {
		var (
			runImage        string
			testMirror1     string
			testMirror2     string
			testRunImageCfg []config.RunImage
		)
		it.Before(func() {
			runImage = "test/image"
			testMirror1 = "example.com/some/run1"
			testMirror2 = "example.com/some/run2"
			testRunImageCfg = []config.RunImage{{
				Image:   runImage,
				Mirrors: []string{testMirror1, testMirror2},
			}}
		})

		when("no run image is specified", func() {
			it("fails to run", func() {
				err := command.Execute()
				h.AssertError(t, err, "accepts 1 arg")
			})
		})

		when("mirrors are provided", func() {
			it("adds them as mirrors to the config", func() {
				command.SetArgs([]string{runImage, "-m", testMirror1, "-m", testMirror2})
				h.AssertNil(t, command.Execute())
				cfg, err := config.Read(cfgPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.RunImages, testRunImageCfg)
			})
		})

		when("no mirrors are provided", func() {
			it.Before(func() {
				cfg.RunImages = testRunImageCfg
				command = commands.SetRunImagesMirrors(logger, cfg, cfgPath)
			})

			it("removes all mirrors for the run image", func() {
				command.SetArgs([]string{runImage})
				h.AssertNil(t, command.Execute())

				cfg, err := config.Read(cfgPath)
				h.AssertNil(t, err)
				h.AssertEq(t, cfg.RunImages, []config.RunImage{{Image: runImage}})
			})
		})
	})
}
