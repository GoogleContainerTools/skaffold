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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
)

var (
	directory string
	format    string
)

// NewCmdFindConfigs list the skaffold config files in the specified directory.
func NewCmdFindConfigs() *cobra.Command {
	return NewCmd("find-configs").
		WithDescription("Find in a given directory all skaffold yamls files that are parseable or upgradeable with their versions.").
		WithFlags(func(f *pflag.FlagSet) {
			// Default to current directory
			f.StringVarP(&directory, "directory", "d", ".", "Root directory to lookup the config files.")
			// Output format of this Command
			f.StringVarP(&format, "output", "o", "table", "Result format, default to table. [(-o|--output=)json|table]")
		}).
		Hidden().
		NoArgs(doFindConfigs)
}

func doFindConfigs(_ context.Context, out io.Writer) error {
	pathToVersion, err := findConfigs(directory)
	if err != nil {
		return err
	}

	switch format {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "\t")
		return encoder.Encode(pathToVersion)

	case "table":
		pathOutLen, versionOutLen := 70, 30
		for p, v := range pathToVersion {
			c := color.Default
			if v != latest.Version {
				c = color.Green
			}
			c.Fprintf(out, fmt.Sprintf("%%-%ds\t%%-%ds\n", pathOutLen, versionOutLen), p, v)
		}

		return nil

	default:
		return fmt.Errorf("unsupported template: %s", format)
	}
}

func findConfigs(directory string) (map[string]string, error) {
	pathToVersion := make(map[string]string)

	// Find files ending in ".yaml" and parseable to skaffold config in the specified root directory recursively.
	isYaml := func(path string, info walk.Dirent) (bool, error) {
		return !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")), nil
	}

	err := walk.From(directory).When(isYaml).Do(func(path string, _ walk.Dirent) error {
		if cfg, err := schema.ParseConfig(path); err == nil {
			pathToVersion[path] = cfg.GetVersion()
		}
		return nil
	})

	return pathToVersion, err
}
