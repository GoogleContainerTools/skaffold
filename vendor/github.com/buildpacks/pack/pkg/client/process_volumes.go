//go:build linux || windows

package client

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/docker/docker/volume/mounts"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/style"
)

func processVolumes(imgOS string, volumes []string) (processed []string, warnings []string, err error) {
	var parser mounts.Parser
	switch "windows" {
	case imgOS:
		parser = mounts.NewWindowsParser()
	case runtime.GOOS:
		parser = mounts.NewLCOWParser()
	default:
		parser = mounts.NewLinuxParser()
	}
	for _, v := range volumes {
		volume, err := parser.ParseMountRaw(v, "")
		if err != nil {
			return nil, nil, errors.Wrapf(err, "platform volume %q has invalid format", v)
		}

		sensitiveDirs := []string{"/cnb", "/layers", "/workspace"}
		if imgOS == "windows" {
			sensitiveDirs = []string{`c:/cnb`, `c:\cnb`, `c:/layers`, `c:\layers`, `c:/workspace`, `c:\workspace`}
		}
		for _, p := range sensitiveDirs {
			if strings.HasPrefix(strings.ToLower(volume.Spec.Target), p) {
				warnings = append(warnings, fmt.Sprintf("Mounting to a sensitive directory %s", style.Symbol(volume.Spec.Target)))
			}
		}

		processed = append(processed, fmt.Sprintf("%s:%s:%s", volume.Spec.Source, volume.Spec.Target, processMode(volume.Mode)))
	}
	return processed, warnings, nil
}

func processMode(mode string) string {
	if mode == "" {
		return "ro"
	}

	return mode
}
