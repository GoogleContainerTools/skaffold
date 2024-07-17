package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/log"
)

const (
	EnvDeprecationMode     = "CNB_DEPRECATION_MODE"
	DefaultDeprecationMode = ModeWarn

	ModeQuiet = "quiet"
	ModeWarn  = "warn"
	ModeError = "error"
)

var DeprecationMode = EnvOrDefault(EnvDeprecationMode, DefaultDeprecationMode)

type BuildpackAPIVerifier struct{}

func (v *BuildpackAPIVerifier) VerifyBuildpackAPI(kind, name, requested string, logger log.Logger) error {
	return VerifyBuildpackAPI(kind, name, requested, logger)
}

func VerifyBuildpackAPI(kind, name, requested string, logger log.Logger) error {
	requestedAPI, err := api.NewVersion(requested)
	if err != nil {
		return FailErrCode(
			nil,
			CodeForIncompatibleBuildpackAPI,
			fmt.Sprintf("parse buildpack API '%s' for %s '%s'", requestedAPI, strings.ToLower(kind), name),
		)
	}
	if api.Buildpack.IsSupported(requestedAPI) {
		if api.Buildpack.IsDeprecated(requestedAPI) {
			switch DeprecationMode {
			case ModeQuiet:
				break
			case ModeError:
				logger.Errorf("%s '%s' requests deprecated API '%s'", kind, name, requested)
				logger.Errorf("Deprecated APIs are disabled by %s=%s", EnvDeprecationMode, ModeError)
				return buildpackAPIError(kind, name, requested)
			case ModeWarn:
				logger.Warnf("%s '%s' requests deprecated API '%s'", kind, name, requested)
			default:
				logger.Warnf("%s '%s' requests deprecated API '%s'", kind, name, requested)
			}
		}
		return nil
	}
	return buildpackAPIError(kind, name, requested)
}

func buildpackAPIError(moduleKind string, name string, requested string) error {
	return FailErrCode(
		fmt.Errorf("buildpack API version '%s' is incompatible with the lifecycle", requested),
		CodeForIncompatibleBuildpackAPI,
		fmt.Sprintf("set API for %s '%s'", moduleKind, name),
	)
}

func VerifyPlatformAPI(requested string, logger log.Logger) error {
	requestedAPI, err := api.NewVersion(requested)
	if err != nil {
		return FailErrCode(
			nil,
			CodeForIncompatiblePlatformAPI,
			fmt.Sprintf("parse platform API '%s'", requested),
		)
	}
	if api.Platform.IsSupported(requestedAPI) {
		if api.Platform.IsDeprecated(requestedAPI) {
			switch DeprecationMode {
			case ModeQuiet:
				break
			case ModeError:
				logger.Errorf("Platform requested deprecated API '%s'", requested)
				logger.Errorf("Deprecated APIs are disabled by %s=%s", EnvDeprecationMode, ModeError)
				return platformAPIError(requested)
			case ModeWarn:
				logger.Warnf("Platform requested deprecated API '%s'", requested)
			default:
				logger.Warnf("Platform requested deprecated API '%s'", requested)
			}
		}
		return nil
	}
	return platformAPIError(requested)
}

func platformAPIError(requested string) error {
	return FailErrCode(
		fmt.Errorf("platform API version '%s' is incompatible with the lifecycle", requested),
		CodeForIncompatiblePlatformAPI,
		"set platform API",
	)
}

func EnvOrDefault(key string, defaultVal string) string {
	if envVal := os.Getenv(key); envVal != "" {
		return envVal
	}
	return defaultVal
}
