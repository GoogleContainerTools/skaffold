package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/log"
)

const (
	// EnvPlatformAPI configures the Platform API version.
	EnvPlatformAPI         = "CNB_PLATFORM_API"
	EnvDeprecationMode     = "CNB_DEPRECATION_MODE"
	DefaultDeprecationMode = ModeWarn

	ModeQuiet = "quiet"
	ModeWarn  = "warn"
	ModeError = "error"
)

var DeprecationMode = EnvOrDefault(EnvDeprecationMode, DefaultDeprecationMode)

type BuildpackAPIVerifier struct{}

// VerifyBuildpackAPI given a Buildpack API version and relevant information for logging
// will error if the requested version is unsupported,
// and will log a warning, error, or do nothing if the requested version is deprecated,
// depending on if the configured deprecation mode is "warn", "error", or "silent", respectively.
func (v *BuildpackAPIVerifier) VerifyBuildpackAPI(kind, name, requestedVersion string, logger log.Logger) error {
	return VerifyBuildpackAPI(kind, name, requestedVersion, logger)
}

// VerifyBuildpackAPI given a Buildpack API version and relevant information for logging
// will error if the requested version is unsupported,
// and will log a warning, error, or do nothing if the requested version is deprecated,
// depending on if the configured deprecation mode is "warn", "error", or "silent", respectively.
func VerifyBuildpackAPI(kind, name, requestedVersion string, logger log.Logger) error {
	requested, err := api.NewVersion(requestedVersion)
	if err != nil {
		return FailErrCode(
			nil,
			CodeForIncompatibleBuildpackAPI,
			fmt.Sprintf("parse buildpack API '%s' for %s '%s'", requested, strings.ToLower(kind), name),
		)
	}
	if api.Buildpack.IsSupported(requested) {
		if api.Buildpack.IsDeprecated(requested) {
			switch DeprecationMode {
			case ModeQuiet:
				break
			case ModeError:
				logger.Errorf("%s '%s' requests deprecated API '%s'", kind, name, requestedVersion)
				logger.Errorf("Deprecated APIs are disabled by %s=%s", EnvDeprecationMode, ModeError)
				return buildpackAPIError(kind, name, requestedVersion)
			case ModeWarn:
				logger.Warnf("%s '%s' requests deprecated API '%s'", kind, name, requestedVersion)
			default:
				logger.Warnf("%s '%s' requests deprecated API '%s'", kind, name, requestedVersion)
			}
		}
		return nil
	}
	return buildpackAPIError(kind, name, requestedVersion)
}

func buildpackAPIError(moduleKind string, name string, requested string) error {
	return FailErrCode(
		fmt.Errorf("buildpack API version '%s' is incompatible with the lifecycle", requested),
		CodeForIncompatibleBuildpackAPI,
		fmt.Sprintf("set API for %s '%s'", moduleKind, name),
	)
}

// VerifyPlatformAPI given a Platform API version and relevant information for logging
// will error if the requested version is unsupported,
// and will log a warning, error, or do nothing if the requested version is deprecated,
// depending on if the configured deprecation mode is "warn", "error", or "silent", respectively.
func VerifyPlatformAPI(requestedVersion string, logger log.Logger) error {
	if strings.TrimSpace(requestedVersion) == "" {
		return FailErrCode(
			nil,
			CodeForIncompatiblePlatformAPI,
			fmt.Sprintf("get platform API version; please set '%s' to specify the desired platform API version", EnvPlatformAPI),
		)
	}
	requested, err := api.NewVersion(requestedVersion)
	if err != nil {
		return FailErrCode(
			nil,
			CodeForIncompatiblePlatformAPI,
			fmt.Sprintf("parse platform API '%s'", requestedVersion),
		)
	}
	if api.Platform.IsSupported(requested) {
		if api.Platform.IsDeprecated(requested) {
			switch DeprecationMode {
			case ModeQuiet:
				break
			case ModeError:
				logger.Errorf("Platform requested deprecated API '%s'", requestedVersion)
				logger.Errorf("Deprecated APIs are disabled by %s=%s", EnvDeprecationMode, ModeError)
				return platformAPIError(requestedVersion)
			case ModeWarn:
				logger.Warnf("Platform requested deprecated API '%s'", requestedVersion)
			default:
				logger.Warnf("Platform requested deprecated API '%s'", requestedVersion)
			}
		}
		return nil
	}
	return platformAPIError(requestedVersion)
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
