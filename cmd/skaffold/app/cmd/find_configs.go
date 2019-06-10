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
)

// NewCmdInit describes the CLI command to generate a Skaffold configuration.
func NewCmdFindConfigs(out io.Writer) *cobra.Command {
	return NewCmd(out, "find-configs").
		WithDescription("Find in a given directory all skaffold yamls files that are parseable or upgradeable with their versions.").
		WithFlags(func(f *pflag.FlagSet) {
			// Default to current directory
			f.StringVarP(&directory, "directory", "d", ".", "Root directory to lookup the config files.")
		}).
		NoArgs(doFindConfigs)
}

func doFindConfigs(out io.Writer) error {
	pathOutLen, versionOutLen := 70, 20

	return filepath.Walk(directory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Find files ending in ".yaml" and parseable to skaffold config in the specified root directory recursively.
			if !info.IsDir() && strings.HasSuffix(path, ".yaml") {
				cfg, err := schema.ParseConfig(path, false)
				if err == nil {
					if cfg.GetVersion() == latest.Version {
						printColoredRow(out, pathOutLen, versionOutLen, path, "LATEST", color.Default)
					} else {
						printColoredRow(out, pathOutLen, versionOutLen, path, cfg.GetVersion(), color.Green)
					}
				}
			}
			return nil
		})
}

func printColoredRow(out io.Writer, pathLen, versionLen int, path, version string, c color.Color) {
	formatTemplate := fmt.Sprintf("%%-%ds\t%%-%ds\n", pathLen, versionLen)
	c.Fprintf(out, formatTemplate, path, version)
}
