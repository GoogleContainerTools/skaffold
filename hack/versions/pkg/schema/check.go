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
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/hack/versions/pkg/diff"
	"github.com/GoogleContainerTools/skaffold/hack/versions/pkg/version"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
)

const baseRef = "origin/master"

func RunSchemaCheckOnChangedFiles() error {
	git, err := newGit(baseRef)
	if err != nil {
		return err
	}
	changedFiles, err := git.getChangedFiles()
	if err != nil {
		return err
	}
	var changedConfigFiles []string
	for _, file := range changedFiles {
		if strings.Contains(file, "config.go") {
			changedConfigFiles = append(changedConfigFiles, file)
		}
	}

	root, err := ioutil.TempDir("", "skaffold")
	if err != nil {
		return err
	}
	var filesInError []string
	for _, configFile := range changedConfigFiles {
		content, err := ioutil.ReadFile(configFile)
		if err != nil {
			return errors.Wrapf(err, "reading %s", configFile)
		}
		changedFile := path.Join(root, "changed.go")
		if err = ioutil.WriteFile(changedFile, content, 0666); err != nil {
			return errors.Wrapf(err, "writing changed version of %s", configFile)
		}

		content, err = git.getFileFromBaseline(configFile)
		if err != nil {
			if strings.Contains(err.Error(), fmt.Sprintf("config.go' exists on disk, but not in '%s'", baseRef)) {
				logrus.Warnf("Can't find %s in %s. Assuming this PR is for a new version creation, skipping...", configFile, baseRef)
				continue
			}
			return err
		}
		baseFile := path.Join(root, "base.go")
		if err = ioutil.WriteFile(baseFile, content, 0666); err != nil {
			return errors.Wrapf(err, "writing %s version of %s", baseRef, configFile)
		}

		diff, err := diff.CompareGoStructs(baseFile, changedFile)
		if err != nil {
			return errors.Wrapf(err, "failed to compare go files %s vs %s", baseFile, changedFile)
		}

		isLatest := strings.Contains(configFile, "latest")
		if diff == "" {
			continue
		}
		if !isLatest {
			filesInError = append(filesInError, configFile)
			continue
		}

		logrus.Infof("structural changes in latest config, checking on Github if latest is released...")
		latestVersion, isReleased := version.GetLatestVersion()
		if !isReleased {
			color.Green.Fprintf(os.Stdout, "%s is unreleased, it is safe to change it.\n", latestVersion)
			continue
		}
		color.Red.Fprintf(os.Stdout, "%s is released, it should NOT be changed!\n", latestVersion)
		filesInError = append(filesInError, configFile)
	}

	for _, file := range filesInError {
		logrus.Errorf(changeDetected(file))
		changes, err := git.diffWithBaseline(file)
		if err != nil {
			logrus.Errorf("failed to get diff: %s", err)
		}
		fmt.Print(string(changes))
	}

	if len(filesInError) > 0 {
		return errors.New("structural changes detected")
	}

	return nil
}

func changeDetected(configFile string) string {
	return fmt.Sprintf(`--------
Structural change detected in a released config: %s
Please create a new PR first with a new version.
You can use 'hack/new_version.sh' to generate the new config version.
If you are running this locally, make sure you have the %s branch up to date!
Admin rights are required to merge this PR!
--------
`, configFile, baseRef)
}
