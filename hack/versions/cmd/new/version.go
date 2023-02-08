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

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	hackschema "github.com/GoogleContainerTools/skaffold/v2/hack/versions/pkg/schema"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/walk"
)

// TODO(yuwenma): Upgrade the version to include v3alpha* once it's available.
// Before: prev -> current (latest)
// After:  prev -> current -> new (latest)
func main() {
	logrus.SetLevel(logrus.DebugLevel)
	prev := strings.TrimPrefix(schema.SchemaVersionsV1[len(schema.SchemaVersionsV1)-2].APIVersion, "skaffold/")
	logrus.Infof("Previous Skaffold version: %s", prev)

	current, latestIsReleased := hackschema.GetLatestVersion()

	if !latestIsReleased {
		logrus.Fatalf("There is no need to create a new version, %s is still not released", current)
	}

	next := readNextVersion(current)
	logrus.Infof("Next Skaffold version: %s", next)

	makeSchemaDir(current)

	// Create a package for current version
	walk.From(path("latest")).WhenIsFile().MustDo(func(file string, info walk.Dirent) error {
		cp(file, path(current, info.Name()))
		sed(path(current, info.Name()), "package v1", "package "+current)
		return nil
	})

	// Create code to upgrade from current to new
	cp(template("upgrade.template"), path(current, "upgrade.go"))
	sed(path(current, "upgrade.go"), "%NEXT_VERSION%", next)
	sed(path(current, "upgrade.go"), "%PREV_VERSION%", current)

	// Create a test for the upgrade from current to new
	cp(template("upgrade_test.template"), path(current, "upgrade_test.go"))
	sed(path(current, "upgrade_test.go"), "%NEXT_VERSION%", next)
	sed(path(current, "upgrade_test.go"), "%PREV_VERSION%", current)

	// Previous version now upgrades to current instead of latest
	sed(path(prev, "upgrade.go"), "latest", current)
	sed(path(prev, "upgrade_test.go"), "latest", current)

	// Latest uses the new version
	sed(path("latest", "config.go"), current, next)

	hackschema.UpdateVersionComment(path("latest", "config.go"), false)

	// Update skaffold.yaml in integration tests
	walk.From("integration").WhenNameMatches("*skaffold*.yaml").MustDo(func(path string, _ walk.Dirent) error {
		sed(path, current, next)
		return nil
	})

	// Update skaffold.yaml in init tests
	walk.From("pkg/skaffold/initializer/testdata").WhenNameMatches("*skaffold*.yaml").MustDo(func(path string, _ walk.Dirent) error {
		sed(path, current, next)
		return nil
	})

	// Update diagnose.tmpl in diagnose tests
	walk.From("integration/testdata/diagnose").WhenNameMatches("diagnose.tmpl").MustDo(func(path string, _ walk.Dirent) error {
		sed(path, current, next)
		return nil
	})

	// Add the new version to the list of versions
	lines := lines(path("versions.go"))
	var content string
	for _, line := range lines {
		content += line + "\n"
		if strings.Contains(line, prev) {
			content += strings.ReplaceAll(line, prev, current) + "\n"
		}
	}
	write(path("versions.go"), []byte(content))

	// Update the docs with the new version
	sed("docs-v2/config.toml", current, next)
}

func makeSchemaDir(new string) {
	latestDir, _ := os.Stat(path("latest"))
	newDirPath := path(new)
	if err := os.Mkdir(newDirPath, latestDir.Mode()); err != nil {
		logrus.Fatalf("creating dir %s: %s", newDirPath, err)
	}
}

func readNextVersion(current string) string {
	var new string
	if len(os.Args) <= 1 {
		new = bumpVersion(current)
		output.Red.Fprintf(os.Stdout, "Please enter new version (default: %s): ", new)
		reader := bufio.NewReader(os.Stdin)
		if line, err := reader.ReadString('\n'); err != nil {
			logrus.Fatalf("error reading input: %s", err)
		} else if strings.TrimSpace(line) != "" {
			new = line
		}
	} else {
		new = os.Args[1]
	}
	return strings.TrimSpace(new)
}

// bumpVersion increments a KRM-style version string (v1 -> v2alpha1, v2beta11 -> v2beta12).
func bumpVersion(version string) string {
	// turn a released version into next alpha (v1 -> v2alpha1)
	if m := regexp.MustCompile(`^v([0-9]+)$`).FindStringSubmatch(version); len(m) > 0 {
		i, _ := strconv.Atoi(m[1])
		return fmt.Sprintf("v%dalpha1", i+1)
	}
	// bump alpha/beta version by 1 (v1beta2 -> v1beta2)
	if m := regexp.MustCompile(`^(v[0-9]+(?:alpha|beta))([0-9]+)$`).FindStringSubmatch(version); len(m) > 0 {
		i, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("%s%d", m[1], i+1)
	}
	logrus.Warnf("Unrecognized version string: %s", version)
	return version
}

func path(elem ...string) string {
	return filepath.Join(append([]string{"pkg", "skaffold", "schema"}, elem...)...)
}

func template(file string) string {
	return filepath.Join(append([]string{"hack", "versions", "cmd", "new", "templates"}, file)...)
}

func read(path string) []byte {
	buf, err := os.ReadFile(path)
	if err != nil {
		panic("unable to read " + path)
	}
	return buf
}

func write(path string, buf []byte) {
	if err := os.WriteFile(path, buf, os.ModePerm); err != nil {
		panic("unable to write " + path)
	}
}

func sed(path string, old, new string) {
	buf := read(path)
	replaced := regexp.MustCompile(old).ReplaceAll(buf, []byte(new))
	write(path, replaced)
}

func cp(path string, dest string) {
	buf := read(path)
	os.Mkdir(filepath.Dir(dest), os.ModePerm)
	write(dest, buf)
}

func lines(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		panic("unable to open " + path)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
