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
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/blang/semver"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
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

// IsUpdateCheckEnabled returns whether or not the update check is enabled
// It is true by default, but setting it to any other value than true will disable the check
func IsUpdateCheckEnabled(configfile string) bool {
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
	resp, err := http.Get(LatestVersionURL)
	if err != nil {
		return "", fmt.Errorf("getting latest version info from GCS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http %d, error %q", resp.StatusCode, resp.Status)
	}
	versionBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading version file from GCS: %w", err)
	}
	return strings.TrimSuffix(string(versionBytes), "\n"), nil
}
