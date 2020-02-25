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
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	hackschema "github.com/GoogleContainerTools/skaffold/hack/versions/pkg/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
)

// Print the latest version released.
func main() {
	logrus.SetLevel(logrus.ErrorLevel)

	current, latestIsReleased := hackschema.GetLatestVersion()

	if latestIsReleased {
		fmt.Println(current)
	} else {
		prev := strings.TrimPrefix(schema.SchemaVersions[len(schema.SchemaVersions)-2].APIVersion, "skaffold/")
		fmt.Println(prev)
	}
}
