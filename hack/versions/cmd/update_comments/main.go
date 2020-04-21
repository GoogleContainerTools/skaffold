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
	"path"
	"strings"

	"github.com/sirupsen/logrus"

	hackschema "github.com/GoogleContainerTools/skaffold/hack/versions/pkg/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
)

func main() {
	_, isReleased := hackschema.GetLatestVersion()

	updateVersionComment := func(path string, _ walk.Dirent) error {
		released := !strings.Contains(path, "latest") || isReleased
		return hackschema.UpdateVersionComment(path, released)
	}

	schemaDir := path.Join("pkg", "skaffold", "schema")
	if err := walk.From(schemaDir).WhenHasName("config.go").Do(updateVersionComment); err != nil {
		logrus.Fatalf("%s", err)
	}
}
