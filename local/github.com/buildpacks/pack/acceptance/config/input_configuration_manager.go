//go:build acceptance

package config

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const (
	envAcceptanceSuiteConfig    = "ACCEPTANCE_SUITE_CONFIG"
	envCompilePackWithVersion   = "COMPILE_PACK_WITH_VERSION"
	envGitHubToken              = "GITHUB_TOKEN"
	envLifecycleImage           = "LIFECYCLE_IMAGE"
	envLifecyclePath            = "LIFECYCLE_PATH"
	envPackPath                 = "PACK_PATH"
	envPreviousLifecycleImage   = "PREVIOUS_LIFECYCLE_IMAGE"
	envPreviousLifecyclePath    = "PREVIOUS_LIFECYCLE_PATH"
	envPreviousPackFixturesPath = "PREVIOUS_PACK_FIXTURES_PATH"
	envPreviousPackPath         = "PREVIOUS_PACK_PATH"
)

type InputConfigurationManager struct {
	packPath                 string
	previousPackPath         string
	previousPackFixturesPath string
	lifecyclePath            string
	previousLifecyclePath    string
	lifecycleImage           string
	previousLifecycleImage   string
	compilePackWithVersion   string
	githubToken              string
	combinations             ComboSet
}

func NewInputConfigurationManager() (InputConfigurationManager, error) {
	packPath := os.Getenv(envPackPath)
	previousPackPath := os.Getenv(envPreviousPackPath)
	previousPackFixturesPath := os.Getenv(envPreviousPackFixturesPath)
	lifecyclePath := os.Getenv(envLifecyclePath)
	previousLifecyclePath := os.Getenv(envPreviousLifecyclePath)

	err := resolveAbsolutePaths(&packPath, &previousPackPath, &previousPackFixturesPath, &lifecyclePath, &previousLifecyclePath)
	if err != nil {
		return InputConfigurationManager{}, err
	}

	var combos ComboSet

	comboConfig := os.Getenv(envAcceptanceSuiteConfig)
	if comboConfig != "" {
		if err := json.Unmarshal([]byte(comboConfig), &combos); err != nil {
			return InputConfigurationManager{}, errors.Errorf("failed to parse combination config: %s", err)
		}
	} else {
		combos = defaultRunCombo
	}

	if lifecyclePath != "" && len(combos) == 1 && combos[0] == defaultRunCombo[0] {
		combos[0].Lifecycle = Current
	}

	return InputConfigurationManager{
		packPath:                 packPath,
		previousPackPath:         previousPackPath,
		previousPackFixturesPath: previousPackFixturesPath,
		lifecyclePath:            lifecyclePath,
		previousLifecyclePath:    previousLifecyclePath,
		lifecycleImage:           os.Getenv(envLifecycleImage),
		previousLifecycleImage:   os.Getenv(envPreviousLifecycleImage),
		compilePackWithVersion:   os.Getenv(envCompilePackWithVersion),
		githubToken:              os.Getenv(envGitHubToken),
		combinations:             combos,
	}, nil
}

func (i InputConfigurationManager) Combinations() ComboSet {
	return i.combinations
}

func resolveAbsolutePaths(paths ...*string) error {
	for _, path := range paths {
		if *path == "" {
			continue
		}

		// Manually expand ~ to home dir
		if strings.HasPrefix(*path, "~/") {
			usr, err := user.Current()
			if err != nil {
				return errors.Wrapf(err, "getting current user")
			}
			dir := usr.HomeDir
			*path = filepath.Join(dir, (*path)[2:])
		} else {
			absPath, err := filepath.Abs(*path)
			if err != nil {
				return errors.Wrapf(err, "getting absolute path for %s", *path)
			}
			*path = absPath
		}
	}

	return nil
}
