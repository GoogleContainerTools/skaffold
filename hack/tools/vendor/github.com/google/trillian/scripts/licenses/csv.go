// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/csv"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/google/trillian/scripts/licenses/licenses"
	"github.com/spf13/cobra"
)

var (
	csvCmd = &cobra.Command{
		Use:   "csv <package>",
		Short: "Prints all licenses that apply to a Go package and its dependencies",
		Args:  cobra.ExactArgs(1),
		RunE:  csvMain,
	}

	gitRemotes []string
)

func init() {
	csvCmd.Flags().StringArrayVar(&gitRemotes, "git_remote", []string{"origin", "upstream"}, "Remote Git repositories to try")

	rootCmd.AddCommand(csvCmd)
}

func csvMain(_ *cobra.Command, args []string) error {
	importPath := args[0]
	writer := csv.NewWriter(os.Stdout)

	classifier, err := licenses.NewClassifier(confidenceThreshold)
	if err != nil {
		return err
	}

	libs, err := licenses.Libraries(context.Background(), importPath)
	if err != nil {
		return err
	}
	for _, lib := range libs {
		licenseURL := "Unknown"
		licenseName := "Unknown"
		if lib.LicensePath != "" {
			// Find a URL for the license file, based on the URL of a remote for the Git repository.
			var errs []string
			for _, remote := range gitRemotes {
				url, err := licenses.GitFileURL(lib.LicensePath, remote)
				if err != nil {
					errs = append(errs, err.Error())
					continue
				}
				licenseURL = url.String()
				break
			}
			if licenseURL == "Unknown" {
				glog.Errorf("Error discovering URL for %q:\n- %s", lib.LicensePath, strings.Join(errs, "\n- "))
			}
			licenseName, _, err = classifier.Identify(lib.LicensePath)
			if err != nil {
				return err
			}
		}
		// Remove the "*/vendor/" prefix from the library name for conciseness.
		if err := writer.Write([]string{unvendor(lib.Name()), licenseURL, licenseName}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
