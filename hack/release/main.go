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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/v2/hack/versions/pkg/schema"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type changelogData struct {
	SkaffoldVersion string
	Date            string
	SchemaString    string
}

type versionNote struct {
	BinVersion      string `json:"binVersion"`
	ReleaseNoteLink string `json:"releaseNoteLink"`
}

// TODO(marlongamez): do some autosorting of `release-notes` binary output
func main() {
	var skaffoldVersion string
	// Get skaffold version from user
	if err := survey.AskOne(&survey.Input{Message: "Input skaffold version:"}, &skaffoldVersion); err != nil {
		fmt.Printf("failed to get skaffold version from input: %v\n", err)
		os.Exit(1)
	}

	// Add extra string if new schema version is being released
	schemaPath := path.Join("pkg", "skaffold", "schema", "latest", "config.go")
	released, err := schema.IsReleased(schemaPath)
	if err != nil {
		fmt.Printf("error occurred: %v\n", err)
		os.Exit(1)
	}
	data := getChangelogData(skaffoldVersion, released)
	if !released {
		schemaVersion := strings.TrimPrefix(latest.Version, "skaffold/")
		output := filepath.Join(".", "docs-v2", "content", "en", "schemas", "version-mappings", schemaVersion+"-version.json")
		err = writeVersionMapping(skaffoldVersion, output)
		if err != nil {
			fmt.Printf("error occurred: %v\n", err)
			os.Exit(1)
		}
	}

	if err = updateChangelog(path.Join("CHANGELOG.md"), path.Join("hack", "release", "template.md"), data); err != nil {
		fmt.Printf("error occurred: %v\n", err)
		os.Exit(1)
	}
}

func writeVersionMapping(binVersion string, output string) error {
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()

	b, err := json.Marshal(versionNote{
		BinVersion:      binVersion,
		ReleaseNoteLink: "https://github.com/GoogleContainerTools/skaffold/releases/tag/" + binVersion,
	})
	if err != nil {
		return err
	}
	_, err = file.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func getChangelogData(skaffoldVersion string, released bool) changelogData {
	data := changelogData{}

	data.SkaffoldVersion = strings.TrimPrefix(skaffoldVersion, "v")
	semver.MustParse(data.SkaffoldVersion)

	// Get current time
	currentTime := time.Now()
	data.Date = currentTime.Format("01/02/2006")

	if !released {
		schemaVersion := strings.TrimPrefix(latest.Version, "skaffold/")
		data.SchemaString = fmt.Sprintf("\nNote: This release comes with a new config version, `%s`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.\n", schemaVersion)
	}

	return data
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
	b, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("reading changelog file: %w", err)
	}
	_, err = fmt.Fprint(&buf, string(b))
	if err != nil {
		return fmt.Errorf("writing to changelog buffer: %w", err)
	}
	if err = os.WriteFile(filepath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing to changelog file: %w", err)
	}

	return nil
}
