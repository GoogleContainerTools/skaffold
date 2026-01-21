//go:build unix && !linux

package client

import (
	"fmt"
	"strings"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
)

func processVolumes(imgOS string, volumes []string) (processed []string, warnings []string, err error) {
	for _, v := range volumes {
		volume, err := parseVolume(v)
		if err != nil {
			return nil, nil, err
		}
		sensitiveDirs := []string{"/cnb", "/layers", "/workspace"}
		if imgOS == "windows" {
			sensitiveDirs = []string{`c:/cnb`, `c:\cnb`, `c:/layers`, `c:\layers`}
		}
		for _, p := range sensitiveDirs {
			if strings.HasPrefix(strings.ToLower(volume.Target), p) {
				warnings = append(warnings, fmt.Sprintf("Mounting to a sensitive directory %s", style.Symbol(volume.Target)))
			}
		}
		mode := "ro"
		if strings.HasSuffix(v, ":rw") && !volume.ReadOnly {
			mode = "rw"
		}
		processed = append(processed, fmt.Sprintf("%s:%s:%s", volume.Source, volume.Target, mode))
	}
	return processed, warnings, nil
}

func parseVolume(volume string) (types.ServiceVolumeConfig, error) {
	// volume format: '<host path>:<target path>[:<options>]'
	split := strings.Split(volume, ":")
	if len(split) == 3 {
		if split[2] != "ro" && split[2] != "rw" && !strings.Contains(split[2], "volume-opt") {
			return types.ServiceVolumeConfig{}, errors.New(fmt.Sprintf("platform volume %q has invalid format: invalid mode: %s", volume, split[2]))
		}
	}
	config, err := loader.ParseVolume(volume)
	if err != nil {
		return config, errors.Wrapf(err, "platform volume %q has invalid format", volume)
	}
	return config, nil
}
