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
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
)

var OutputType string

// List prints to `out` all supported schema versions.
func List(_ context.Context, out io.Writer) error {
	return list(out, OutputType)
}

func list(out io.Writer, outputType string) error {
	switch outputType {
	case "json":
		return printJSON(out)
	case "plain":
		return printPlain(out)
	default:
		return fmt.Errorf(`invalid output type: %q. Must be "plain" or "json"`, outputType)
	}
}

type schemaList struct {
	Versions []string `json:"versions"`
}

func printJSON(out io.Writer) error {
	return json.NewEncoder(out).Encode(schemaList{
		Versions: versions(),
	})
}

func printPlain(out io.Writer) error {
	for _, version := range versions() {
		fmt.Fprintln(out, version)
	}

	return nil
}

func versions() []string {
	var versions []string

	for _, version := range schema.SchemaVersions {
		versions = append(versions, version.APIVersion)
	}

	return versions
}
