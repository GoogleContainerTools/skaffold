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

package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// DetectWSL checks for Windows Subsystem for Linux
func DetectWSL() (bool, error) {
	if _, err := os.Stat("/proc/version"); err == nil {
		b, err := ioutil.ReadFile("/proc/version")
		if err != nil {
			return false, fmt.Errorf("read /proc/version: %w", err)
		}

		// Microsoft changed the case between WSL1 and WSL2
		str := strings.ToLower(string(b))
		if strings.Contains(str, "microsoft") {
			return true, nil
		}
	}
	return false, nil
}
