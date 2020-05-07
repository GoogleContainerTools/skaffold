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

package trigger

import (
	"os"
	"runtime"
)

// On macOS, $PWD can have a different case that the actual current
// working directory. That happens for example if one does:
// `cd /home/me/MY-PROJECT` before running Skaffold,
// instead of `cd /home/me/my-project`.
// In such a situation, the file watcher will fail to detect changes.
// To solve that, we force `os.Getwd()` not to use $PWD.
// See: https://github.com/rjeczalik/notify/issues/96
func RealWorkDir() (string, error) {
	if runtime.GOOS == "darwin" {
		if pwd, present := os.LookupEnv("PWD"); present {
			os.Unsetenv("PWD")
			defer os.Setenv("PWD", pwd)
		}
	}

	return os.Getwd()
}
