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
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/hack/versions/pkg/diff"
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
			return fmt.Errorf("reading %q: %w", configFile, err)
		}
		changedFile := path.Join(root, "changed.go")
		if err = ioutil.WriteFile(changedFile, content, 0666); err != nil {
			return fmt.Errorf("writing changed version of %q: %w", configFile, err)
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
			return fmt.Errorf("writing %s version of %q: %w", baseRef, configFile, err)
		}

		diff, err := diff.CompareGoStructs(baseFile, changedFile)
		if err != nil {
			return fmt.Errorf("failed to compare go files %s vs %q: %w", baseFile, changedFile, err)
		}

		isLatest := strings.Contains(configFile, "latest")
		if diff == "" {
			continue
		}
		if !isLatest {
			filesInError = append(filesInError, configFile)
			continue
		}

		logrus.Warnf("Detected changes to the latest config. Checking on Github if it's released...")
		latestVersion, isReleased := GetLatestVersion()
		if !isReleased {
			logrus.Infof("Schema %q is not yet released. Changes are ok.", latestVersion)
			continue
		}

		logrus.Errorf("Schema %q is already released. Changing it is not allowed.", latestVersion)

		fmt.Printf("\nWhat should I do?\n-----------------\n")
		fmt.Printf(" + If this retroactive change is required and is harmless to users, indicate on your PR.\n")
		fmt.Printf(" + Check if a new unreleased version has been created:\n")
		fmt.Printf("     - Ensure that your branch is up-to-date with the %q branch.\n", baseRef)
		fmt.Printf("     - Check for a pending PR to create a new version.\n")
		fmt.Printf(" + Create a separate PR with just the result of running the 'hack/new_version.sh' script.\n")

		filesInError = append(filesInError, configFile)
	}

	if len(filesInError) > 0 {
		fmt.Printf("\nInvalid changes:\n----------------\n")
		for _, file := range filesInError {
			changes, err := git.diffWithBaseline(file)
			if err != nil {
				logrus.Errorf("failed to get diff: %s", err)
			}
			fmt.Print(string(changes))
		}

		return errors.New("structural changes detected")
	}

	return nil
}
