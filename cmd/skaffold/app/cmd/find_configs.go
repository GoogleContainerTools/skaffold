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

// NewCmdFindConfigs list the skaffold config files in the specified directory.
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
	return findConfigs(out, directory)
}

func findConfigs(out io.Writer, directory string) error {
	pathOutLen, versionOutLen := 70, 30

	return filepath.Walk(directory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println(err)
				return err
			}

			// Find files ending in ".yaml" and parseable to skaffold config in the specified root directory recursively.
			if !info.IsDir() && strings.Contains(path, ".yaml") {
				cfg, err := schema.ParseConfig(path, false)
				if err == nil {
					var version string
					var c color.Color

					if cfg.GetVersion() == latest.Version {
						version = cfg.GetVersion() + " (LATEST)"
						c = color.Default
					} else {
						version = cfg.GetVersion()
						c = color.Green
					}
					c.Fprintf(out, getFormatTemplate(pathOutLen, versionOutLen), path, version)
				}
			}
			return nil
		})
}

func getFormatTemplate(pathLen, versionLen int) string {
	return fmt.Sprintf("%%-%ds\t%%-%ds\n", pathLen, versionLen)
}
