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

  "github.com/blang/semver"
	"github.com/AlecAivazis/survey/v2"
	"github.com/GoogleContainerTools/skaffold/hack/versions/pkg/schema"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

type ChangeLogData struct {
	SkaffoldVersion string
	Date string
	SchemaString string
}

// TODO(marlongamez): do some autosorting of `release-notes` binary output
func main() {
	data := ChangeLogData{}

	// Get skaffold version from user
	if err := survey.AskOne(&survey.Input{Message: "Input skaffold version:"}, &data.SkaffoldVersion); err != nil {
		os.Exit(1)
	}
	data.SkaffoldVersion = strings.TrimPrefix(data.SkaffoldVersion, "v")
	semver.MustParse(data.SkaffoldVersion)

	// Get current time
	currentTime := time.Now()
	data.Date = currentTime.Format("01/02/2006")

	// Add extra string if new schema version is being released
	latest := path.Join("pkg", "skaffold", "schema", "latest", "v1", "config.go")
	released, err := schema.IsReleased(latest)
	if err != nil {
		os.Exit(1)
	}
	if !released {
		schemaVersion := strings.TrimPrefix(latestV1.Version, "skaffold/")
		data.SchemaString = fmt.Sprintf("\nNote: This release comes with a new config version, `%s`. To upgrade your skaffold.yaml, use `skaffold fix`. If you choose not to upgrade, skaffold will auto-upgrade as best as it can.\n", schemaVersion)
	}

	// Execute template and combine output with existing CHANGELOG.md contents
	var buf bytes.Buffer
	tmpl, err := template.ParseFiles(path.Join("hack", "release", "changelog", "template.md"))
	if err = tmpl.Execute(&buf, data); err != nil {
		os.Exit(1)
	}
	b, err := ioutil.ReadFile(path.Join("CHANGELOG.md"))
	if err != nil {
		os.Exit(1)
	}
	_, err = fmt.Fprint(&buf, string(b))
	if err != nil {
		os.Exit(1)
	}
	if err = ioutil.WriteFile(path.Join("CHANGELOG.md"), buf.Bytes(), 0644); err != nil {
		os.Exit(1)
	}
}
