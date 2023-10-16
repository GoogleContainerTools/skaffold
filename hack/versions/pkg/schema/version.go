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

package schema

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/update"
)

func GetLatestVersion() (string, bool) {
	current := strings.TrimPrefix(latest.Version, "skaffold/")
	logrus.Debugf("Current Skaffold version: %s", current)

	config, err := os.ReadFile("pkg/skaffold/schema/latest/config.go")
	if err != nil {
		logrus.Fatalf("failed to read latest config: %s", err)
	}

	if strings.Contains(string(config), "This config version is already released") {
		return current, true
	}

	logrus.Infof("Checking for released status of %s...", current)
	lastReleased := GetLastReleasedVersion()
	logrus.Infof("Last released version: %s", lastReleased)

	latestIsReleased := lastReleased == current
	return current, latestIsReleased
}

func GetLastReleasedVersion() string {
	lastTag, err := update.DownloadLatestVersion()
	if err != nil {
		logrus.Fatalf("error getting latest version: %s", err)
	}
	logrus.Infof("last release tag: %s", lastTag)
	// we split the config in v1.25.0
	for _, url := range []string{
		fmt.Sprintf("https://raw.githubusercontent.com/GoogleContainerTools/skaffold/%s/pkg/skaffold/schema/latest/config.go", lastTag),
	} {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			defer resp.Body.Close()
			config, err := io.ReadAll(resp.Body)
			if err != nil {
				logrus.Fatalf("failed to fetch config for %s, err: %s", lastTag, err)
			}
			versionPattern := regexp.MustCompile("const Version string = \"skaffold/(.*)\"")
			lastReleased := versionPattern.FindStringSubmatch(string(config))[1]
			return lastReleased
		}
	}
	logrus.Fatalf("can't determine latest released config version, failed to download %s: %s", lastTag, err)
	return ""
}

// IsReleased takes a filepath to a skaffold config in pkg/skaffold/schema and returns true if it's released and false if otherwise.
func IsReleased(filepath string) (bool, error) {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return false, err
	}

	s := string(b)
	if strings.Contains(s, releasedComment) {
		return true, nil
	}

	return false, nil
}
