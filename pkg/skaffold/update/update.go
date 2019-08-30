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
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/blang/semver"
	"github.com/pkg/errors"
)

// Fot testing
var (
	GetLatestAndCurrentVersion = getLatestAndCurrentVersion
	isConfigUpdateCheckEnabled = config.IsUpdateCheckEnabled
	getEnv                     = os.Getenv
)

const latestVersionURL = "https://storage.googleapis.com/skaffold/releases/latest/VERSION"

// IsUpdateCheckEnabled returns whether or not the update check is enabled
// It is true by default, but setting it to any other value than true will disable the check
func IsUpdateCheckEnabled(configfile string) bool {
	// Don't perform a version check on dirty trees
	if version.Get().GitTreeState == "dirty" {
		return false
	}

	return isUpdateCheckEnabledByEnvOrConfig(configfile)
}

func isUpdateCheckEnabledByEnvOrConfig(configfile string) bool {
	if v := getEnv(constants.UpdateCheckEnvironmentVariable); v != "" {
		return strings.ToLower(v) == "true"
	}
	return isConfigUpdateCheckEnabled(configfile)
}

// getLatestAndCurrentVersion uses a VERSION file stored on GCS to determine the latest released version
// and returns it with the current version of Skaffold
func getLatestAndCurrentVersion() (semver.Version, semver.Version, error) {
	none := semver.Version{}
	resp, err := http.Get(latestVersionURL)
	if err != nil {
		return none, none, errors.Wrap(err, "getting latest version info from GCS")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return none, none, errors.Wrapf(err, "http %d, error: %s", resp.StatusCode, resp.Status)
	}
	versionBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return none, none, errors.Wrap(err, "reading version file from GCS")
	}
	latest, err := version.ParseVersion(string(versionBytes))
	if err != nil {
		return none, none, errors.Wrap(err, "parsing latest version from GCS")
	}
	current, err := version.ParseVersion(version.Get().Version)
	if err != nil {
		return none, none, errors.Wrap(err, "parsing current semver, skipping update check")
	}
	return latest, current, nil
}
