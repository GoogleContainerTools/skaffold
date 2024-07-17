package extend

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Build BuildConfig `toml:"build"`
	Run   BuildConfig `toml:"run"`
}

type BuildConfig struct {
	Args []Arg `toml:"args"`
}

var argsProvidedByLifecycle = []string{"build_id", "user_id", "group_id"}

func ValidateConfig(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}
	var config Config
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		return fmt.Errorf("reading extend config: %w", err)
	}
	checkArgs := func(args []Arg) error {
		for _, arg := range args {
			for _, invalid := range argsProvidedByLifecycle {
				if arg.Name == invalid {
					return fmt.Errorf("invalid content: arg with name %q is not allowed", invalid)
				}
			}
		}
		return nil
	}
	if err = checkArgs(config.Build.Args); err != nil {
		return fmt.Errorf("validating extend config: %w", err)
	}
	if err = checkArgs(config.Run.Args); err != nil {
		return fmt.Errorf("validating extend config: %w", err)
	}
	return nil
}
