// Copyright 2017 Google Inc.
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

package licenseclassifier

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// LicenseDirectory is the directory where the prototype licenses are kept.
	LicenseDirectory = "src/github.com/google/licenseclassifier/licenses"
	// LicenseArchive is the name of the archive containing preprocessed
	// license texts.
	LicenseArchive = "licenses.db"
	// ForbiddenLicenseArchive is the name of the archive containing preprocessed
	// forbidden license texts only.
	ForbiddenLicenseArchive = "forbidden_licenses.db"
)

// lcRoot computes the location of the licenses data in the licenseclassifier source tree based on the location of this file.
func lcRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to compute path of licenseclassifier source")
	}
	// this file must be in the root of the package, or the relative paths will be wrong.
	return filepath.Join(filepath.Dir(filename), "licenses"), nil
}

// ReadLicenseFile locates and reads the license archive file.  Absolute paths are used unmodified.  Relative paths are expected to be in the licenses directory of the licenseclassifier package.
func ReadLicenseFile(filename string) ([]byte, error) {
	if strings.HasPrefix(filename, "/") {
		return ioutil.ReadFile(filename)
	}

	root, err := lcRoot()
	if err != nil {
		return nil, fmt.Errorf("error locating licenses directory: %v", err)
	}
	return ioutil.ReadFile(filepath.Join(root, filename))
}

// ReadLicenseDir reads directory containing the license files.
func ReadLicenseDir() ([]os.FileInfo, error) {
	root, err := lcRoot()
	if err != nil {
		return nil, fmt.Errorf("error locating licenses directory: %v", err)
	}

	return ioutil.ReadDir(root)
}
