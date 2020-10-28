/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package update

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

// EnableCheck enabled the check for a more recent version of Skaffold.
var EnableCheck bool

// For testing
var (
	GetLatestAndCurrentVersion = getLatestAndCurrentVersion
	isConfigUpdateCheckEnabled = config.IsUpdateCheckEnabled
)

const LatestVersionURL = "https://storage.googleapis.com/skaffold/releases/latest/VERSION"

// CheckVersion returns an update message when update check is enabled and skaffold binary in not latest
func CheckVersion(config string) (string, error) {
	return checkVersion(config, false)
}

// CheckVersionOnError returns an error message when update check is enabled and skaffold binary in not latest
func CheckVersionOnError(config string) (string, error) {
	return checkVersion(config, true)
}

func checkVersion(config string, onError bool) (string, error) {
	if !isUpdateCheckEnabled(config) {
		logrus.Debugf("Update check not enabled, skipping.")
		return "", nil
	}
	latest, current, err := GetLatestAndCurrentVersion()
	if err != nil {
		return "", fmt.Errorf("getting latest and current skaffold versions: %w", err)
	}
	if !latest.GT(current) {
		return "", nil
	}
	if onError {
		return fmt.Sprintf("Your Skaffold version might be too old. Download the latest version (%s) from:\n  %s\n", latest, releaseURL(latest)), nil
	}
	return fmt.Sprintf("There is a new version (%s) of Skaffold available. Download it from:\n  %s\n", latest, releaseURL(latest)), nil
}

// isUpdateCheckEnabled returns whether or not the update check is enabled
// It is true by default, but setting it to any other value than true will disable the check
func isUpdateCheckEnabled(configfile string) bool {
	// Don't perform a version check on dirty trees
	if version.Get().GitTreeState == "dirty" {
		return false
	}

	return EnableCheck && isConfigUpdateCheckEnabled(configfile)
}

// getLatestAndCurrentVersion uses a VERSION file stored on GCS to determine the latest released version
// and returns it with the current version of Skaffold
func getLatestAndCurrentVersion() (semver.Version, semver.Version, error) {
	none := semver.Version{}
	versionString, err := DownloadLatestVersion()
	if err != nil {
		return none, none, err
	}
	logrus.Tracef("latest skaffold version: %s", versionString)
	latest, err := version.ParseVersion(versionString)
	if err != nil {
		return none, none, fmt.Errorf("parsing latest version from GCS: %w", err)
	}
	current, err := version.ParseVersion(version.Get().Version)
	if err != nil {
		return none, none, fmt.Errorf("parsing current semver, skipping update check: %w", err)
	}
	return latest, current, nil
}

func DownloadLatestVersion() (string, error) {
	versionBytes, err := util.Download(LatestVersionURL)
	if err != nil {
		return "", fmt.Errorf("getting latest version info from GCS: %w", err)
	}
	return strings.TrimSuffix(string(versionBytes), "\n"), nil
}

func releaseURL(v semver.Version) string {
	return fmt.Sprintf("https://github.com/GoogleContainerTools/skaffold/releases/tag/v" + v.String())
}
