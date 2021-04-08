/*
Copyright 2021 The Skaffold Authors

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

package fsnotify

import (
	"os"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

// For Testing
var (
	fileAttributes = getFileAttributes
)

// Hidden checks if the change detected is to be ignored or not for windows
func (t *Trigger) hidden(path string) bool {
	for _, p := range strings.Split(path, string(os.PathSeparator)) {
		if attributes, err := fileAttributes(p); err != nil {
			logrus.Debugf("could not determine if file %s was hidden due to %e", path, err)
			return false
		} else if attributes&syscall.FILE_ATTRIBUTE_HIDDEN == 1 {
			logrus.Debugf("ignoring hidden file %s", path)
			return true
		}
	}
	return false
}

func getFileAttributes(path string) (uint32, error) {
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	attributes, err := syscall.GetFileAttributes(pointer)
	if err != nil {
		return 0, err
	}
	return attributes, nil
}
