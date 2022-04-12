/*
Copyright 2021 The Skaffold Authors

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

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/hack/versions/pkg/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type changelogData struct {
	SkaffoldVersion string
	Date            string
	SchemaString    string
}

// TODO(marlongamez): do some autosorting of `release-notes` binary output
func main() {
	data, err := getChangelogData(schema.IsReleased)
	if err != nil {
		os.Exit(1)
	}
	if err = updateChangelog(path.Join("CHANGELOG.md"), path.Join("hack", "release", "changelog", "template.md"), data); err != nil {
		os.Exit(1)
	}
}

func getChangelogData(schemaIsReleased func(string) (bool, error)) (changelogData, error) {
	data := changelogData{}

	// Get skaffold version from user
	if err := survey.AskOne(&survey.Input{Message: "Input skaffold version:"}, &data.SkaffoldVersion); err != nil {
		return changelogData{}, fmt.Errorf("failed to get skaffold version from input: %w", err)
	}
	data.SkaffoldVersion = strings.TrimPrefix(data.SkaffoldVersion, "v")
	semver.MustParse(data.SkaffoldVersion)

	// Get current time
	currentTime := time.Now()
	data.Date = currentTime.Format("01/02/2006")

	// Add extra string if new schema version is being released
	schema := path.Join("pkg", "skaffold", "schema", "latest", "v1", "config.go")
	released, err := schemaIsReleased(schema)
	if err != nil {
		return changelogData{}, fmt.Errorf("checking if schema is released: %w", err)
	}
	if !released {
		schemaVersion := strings.TrimPrefix(latest.Version, "skaffold/")
		data.SchemaString = fmt.Sprintf("\nNote: This release comes with a new config version, `%s`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.\n", schemaVersion)
	}

	return data, nil
}

func updateChangelog(filepath, templatePath string, data changelogData) error {
	// Execute template and combine output with existing CHANGELOG.md contents
	var buf bytes.Buffer
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("parsing template file: %w", err)
	}
	if err = tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing changelog template: %w", err)
	}
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("reading changelog file: %w", err)
	}
	_, err = fmt.Fprint(&buf, string(b))
	if err != nil {
		return fmt.Errorf("writing to changelog buffer: %w", err)
	}
	if err = ioutil.WriteFile(filepath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing to changelog file: %w", err)
	}

	return nil
}
