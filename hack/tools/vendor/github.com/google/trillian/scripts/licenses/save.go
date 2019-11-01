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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/golang/glog"
	"github.com/google/trillian/scripts/licenses/licenses"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
)

var (
	saveCmd = &cobra.Command{
		Use:   "save <package>",
		Short: "Saves licenses, copyright notices and source code, as required by a Go package's dependencies, to a directory.",
		Args:  cobra.ExactArgs(1),
		RunE:  saveMain,
	}

	noticeRegexp = regexp.MustCompile(`^NOTICE(\.(txt|md))?$`)

	// savePath is where the output of the command is written to.
	savePath string
	// overwriteSavePath controls behaviour when the directory indicated by savePath already exists.
	// If true, the directory will be replaced. If false, the command will fail.
	overwriteSavePath bool
)

func init() {
	saveCmd.Flags().StringVar(&savePath, "save_path", "", "Directory into which files should be saved that are required by license terms")
	if err := saveCmd.MarkFlagRequired("save_path"); err != nil {
		glog.Fatal(err)
	}
	if err := saveCmd.MarkFlagFilename("save_path"); err != nil {
		glog.Fatal(err)
	}

	saveCmd.Flags().BoolVar(&overwriteSavePath, "force", false, "Delete the destination directory if it already exists.")

	rootCmd.AddCommand(saveCmd)
}

func saveMain(_ *cobra.Command, args []string) error {
	importPath := args[0]

	if overwriteSavePath {
		if err := os.RemoveAll(savePath); err != nil {
			return err
		}
	}

	// Check that the save path doesn't exist, otherwise it'd end up with a mix of
	// existing files and the output of this command.
	if d, err := os.Open(savePath); err == nil {
		d.Close()
		return fmt.Errorf("%s already exists", savePath)
	} else if !os.IsNotExist(err) {
		return err
	}

	classifier, err := licenses.NewClassifier(confidenceThreshold)
	if err != nil {
		return err
	}

	libs, err := licenses.Libraries(context.Background(), importPath)
	if err != nil {
		return err
	}
	libsWithBadLicenses := make(map[licenses.Type][]*licenses.Library)
	for _, lib := range libs {
		libSaveDir := filepath.Join(savePath, unvendor(lib.Name()))
		// Detect what type of license this library has and fulfill its requirements, e.g. copy license, copyright notice, source code, etc.
		_, licenseType, err := classifier.Identify(lib.LicensePath)
		if err != nil {
			return err
		}
		switch licenseType {
		case licenses.Restricted, licenses.Reciprocal:
			// Copy the entire source directory for the library.
			libDir := filepath.Dir(lib.LicensePath)
			if err := copySrc(libDir, libSaveDir); err != nil {
				return err
			}
		case licenses.Notice, licenses.Permissive, licenses.Unencumbered:
			// Just copy the license and copyright notice.
			if err := copyNotices(lib.LicensePath, libSaveDir); err != nil {
				return err
			}
		default:
			libsWithBadLicenses[licenseType] = append(libsWithBadLicenses[licenseType], lib)
		}
	}
	if len(libsWithBadLicenses) > 0 {
		return fmt.Errorf("one or more libraries have an incompatible/unknown license: %q", libsWithBadLicenses)
	}
	return nil
}

func copySrc(src, dest string) error {
	if err := copy.Copy(src, dest); err != nil {
		return err
	}
	// Delete the .git directory from the saved copy, if it exists, since we don't want to save the user's
	// local Git config along with the source code.
	if err := os.RemoveAll(filepath.Join(dest, ".git")); err != nil {
		return err
	}
	return nil
}

func copyNotices(licensePath, dest string) error {
	if err := copy.Copy(licensePath, filepath.Join(dest, filepath.Base(licensePath))); err != nil {
		return err
	}

	src := filepath.Dir(licensePath)
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}
	for _, f := range files {
		if fName := f.Name(); !f.IsDir() && noticeRegexp.MatchString(fName) {
			if err := copy.Copy(filepath.Join(src, fName), filepath.Join(dest, fName)); err != nil {
				return err
			}
		}
	}
	return nil
}
