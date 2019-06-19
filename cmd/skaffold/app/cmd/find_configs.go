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

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	directory string
	format    string
)

// NewCmdFindConfigs list the skaffold config files in the specified directory.
func NewCmdFindConfigs(out io.Writer) *cobra.Command {
	return NewCmd(out, "find-configs").
		WithDescription("Find in a given directory all skaffold yamls files that are parseable or upgradeable with their versions.").
		WithFlags(func(f *pflag.FlagSet) {
			// Default to current directory
			f.StringVarP(&directory, "directory", "d", ".", "Root directory to lookup the config files.")
			// Output format of this Command
			f.StringVarP(&format, "output", "o", "table", "Result format, default to table. [(-o|--output=)json|table]")
		}).
		NoArgs(doFindConfigs)
}

func doFindConfigs(out io.Writer) error {
	pathToVersion, err := findConfigs(directory)

	if err != nil {
		return err
	}

	switch format {
	case "json":
		jsonBytes, err := json.Marshal(pathToVersion)
		if err != nil {
			return err
		}

		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, jsonBytes, "", "\t")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, prettyJSON.String())
	case "table":
		pathOutLen, versionOutLen := 70, 30
		for p, v := range pathToVersion {
			var c color.Color
			if v == latest.Version {
				c = color.Default
			} else {
				c = color.Green
			}
			c.Fprintf(out, fmt.Sprintf("%%-%ds\t%%-%ds\n", pathOutLen, versionOutLen), p, v)
		}
	default:
		return errors.New("unsupported template")
	}

	return nil
}

func findConfigs(directory string) (map[string]string, error) {
	pathToVersion := make(map[string]string)

	err := filepath.Walk(directory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println(err)
				return err
			}

			// Find files ending in ".yaml" and parseable to skaffold config in the specified root directory recursively.
			if !info.IsDir() && (strings.Contains(path, ".yaml") || strings.Contains(path, ".yml")) {
				cfg, err := schema.ParseConfig(path, false)
				if err == nil {
					pathToVersion[path] = cfg.GetVersion()
				}
			}
			return nil
		})

	return pathToVersion, err
}
