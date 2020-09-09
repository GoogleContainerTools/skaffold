/*
Copyright 2020 The Skaffold Authors

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

package logfile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Create creates or truncates a file to be used to output logs.
func Create(path ...string) (*os.File, error) {
	logfile := filepath.Join(os.TempDir(), "skaffold")
	for _, p := range path {
		logfile = filepath.Join(logfile, escape(p))
	}

	dir := filepath.Dir(logfile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("unable to create temp directory %q: %w", dir, err)
	}

	return os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
}

var escapeRegexp = regexp.MustCompile(`[^a-zA-Z0-9-_.]`)

func escape(s string) string {
	return escapeRegexp.ReplaceAllString(s, "-")
}
