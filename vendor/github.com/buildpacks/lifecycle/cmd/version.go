package cmd

import (
	"fmt"
	"strings"
)

// The following variables are injected at compile time.
var (
	// Version is the version of the lifecycle and all produced binaries.
	Version = "0.0.0"
	// SCMCommit is the commit information provided by SCM.
	SCMCommit = ""
	// SCMRepository is the source repository.
	SCMRepository = ""
)

// buildVersion is a display format of the version and build metadata in compliance with semver.
func buildVersion() string {
	// noinspection GoBoolExpressions
	if SCMCommit == "" || strings.Contains(Version, SCMCommit) {
		return Version
	}

	return fmt.Sprintf("%s+%s", Version, SCMCommit)
}
