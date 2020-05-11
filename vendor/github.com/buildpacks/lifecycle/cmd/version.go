package cmd

import (
	"fmt"
	"os"

	"github.com/buildpacks/lifecycle/api"
)

// The following variables are injected at compile time.
var (
	// Version is the version of the lifecycle and all produced binaries.
	Version = "0.0.0"
	// SCMCommit is the commit information provided by SCM.
	SCMCommit = ""
	// SCMRepository is the source repository.
	SCMRepository = ""
	// PlatformAPI is the version of the Platform API implemented.
	PlatformAPI = "0.0"
)

// buildVersion is a display format of the version and build metadata in compliance with semver.
func buildVersion() string {
	// noinspection GoBoolExpressions
	if SCMCommit == "" {
		return Version
	}

	return fmt.Sprintf("%s+%s", Version, SCMCommit)
}

func VerifyCompatibility() error {
	pAPI := os.Getenv("CNB_PLATFORM_API")
	if pAPI != "" {
		platformAPIFromPlatform, err := api.NewVersion(pAPI)
		if err != nil {
			return err
		}

		platformAPIFromLifecycle := api.MustParse(PlatformAPI)
		if !api.IsAPICompatible(platformAPIFromLifecycle, platformAPIFromPlatform) {
			return FailErrCode(
				fmt.Errorf(
					"the Lifecycle's Platform API version is %s which is incompatible with Platform API version %s",
					platformAPIFromLifecycle.String(),
					platformAPIFromPlatform.String(),
				),
				CodeIncompatible,
			)
		}
	}

	return nil
}
