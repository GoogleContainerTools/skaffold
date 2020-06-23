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
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/update"
)

func GetLatestVersion() (string, bool) {
	current := strings.TrimPrefix(latest.Version, "skaffold/")
	logrus.Debugf("Current Skaffold version: %s", current)

	config, err := ioutil.ReadFile("pkg/skaffold/schema/latest/config.go")
	if err != nil {
		logrus.Fatalf("failed to read latest config: %s", err)
	}

	if strings.Contains(string(config), "This config version is already released") {
		return current, true
	}

	logrus.Infof("Checking for released status of %s...", current)
	lastReleased := getLastReleasedConfigVersion()
	logrus.Infof("Last released version: %s", lastReleased)

	latestIsReleased := lastReleased == current
	return current, latestIsReleased
}

func getLastReleasedConfigVersion() string {
	lastTag, err := update.DownloadLatestVersion()
	if err != nil {
		logrus.Fatalf("error getting latest version: %s", err)
	}
	logrus.Infof("last release tag: %s", lastTag)
	configURL := fmt.Sprintf("https://raw.githubusercontent.com/GoogleContainerTools/skaffold/%s/pkg/skaffold/schema/latest/config.go", lastTag)
	resp, err := http.Get(configURL)
	if err != nil {
		logrus.Fatalf("can't determine latest released config version, failed to download %s: %s", configURL, err)
	}
	defer resp.Body.Close()
	config, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Fatalf("failed to read during download %s, err: %s", configURL, err)
	}
	versionPattern := regexp.MustCompile("const Version string = \"skaffold/(.*)\"")
	lastReleased := versionPattern.FindStringSubmatch(string(config))[1]
	return lastReleased
}
