package platform

import (
	"fmt"

	"github.com/buildpacks/lifecycle/log"
)

const (
	FeatureDockerfiles = "Dockerfiles"
	LayoutFormat       = "export to OCI layout format"
)

var ExperimentalMode = envOrDefault(EnvExperimentalMode, DefaultExperimentalMode)

func GuardExperimental(requested string, logger log.Logger) error {
	switch ExperimentalMode {
	case ModeQuiet:
		break
	case ModeError:
		logger.Errorf("Platform requested experimental feature '%s'", requested)
		return fmt.Errorf("experimental features are disabled by %s=%s", EnvExperimentalMode, ModeError)
	case ModeWarn:
		logger.Warnf("Platform requested experimental feature '%s'", requested)
	default:
		// This shouldn't be reached, as ExperimentalMode is always set.
		logger.Warnf("Platform requested experimental feature '%s'", requested)
	}
	return nil
}
