package commands_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestConfigCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ConfigCommands", testConfigCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testConfigCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command      *cobra.Command
		logger       logging.Logger
		outBuf       bytes.Buffer
		tempPackHome string
		configPath   string
		mockClient   *testmocks.MockPackClient
	)

	it.Before(func() {
		var err error

		mockController := gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)

		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		tempPackHome, err = os.MkdirTemp("", "pack-home")
		h.AssertNil(t, err)
		configPath = filepath.Join(tempPackHome, "config.toml")

		command = commands.NewConfigCommand(logger, config.Config{Experimental: true}, configPath, mockClient)
		command.SetOut(logging.GetWriterForLevel(logger, logging.InfoLevel))
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(tempPackHome))
	})

	when("config", func() {
		it("prints help text", func() {
			command.SetArgs([]string{})
			h.AssertNil(t, command.Execute())
			output := outBuf.String()
			h.AssertContains(t, output, "Usage:")
			for _, command := range []string{"trusted-builders", "run-image-mirrors", "default-builder", "experimental", "registries", "pull-policy", "registry-mirrors"} {
				h.AssertContains(t, output, command)
			}
		})
	})
}

type configManager struct {
	testObject *testing.T
	configPath string
}

func newConfigManager(t *testing.T, configPath string) configManager {
	return configManager{
		testObject: t,
		configPath: configPath,
	}
}

func (c configManager) configWithTrustedBuilders(trustedBuilders ...string) config.Config {
	c.testObject.Helper()

	cfg := config.Config{}
	for _, builderName := range trustedBuilders {
		cfg.TrustedBuilders = append(cfg.TrustedBuilders, config.TrustedBuilder{Name: builderName})
	}
	err := config.Write(cfg, c.configPath)
	h.AssertNil(c.testObject, err)

	return cfg
}
