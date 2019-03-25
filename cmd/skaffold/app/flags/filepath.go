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

package flags

import (
	"os"
)

type filepathFlag struct {
	path        string
	shouldExist bool
}

func (f *filepathFlag) SetIfValid(value string) error {
	copy := *f
	copy.path = value
	if err := copy.isValid(); err != nil {
		return err
	}
	*f = copy
	return nil
}

func (f filepathFlag) isValid() error {
	if !f.shouldExist {
		// Currently no validation implemented for output files.
		return nil
	}
	if _, err := os.Stat(f.path); os.IsNotExist(err) {
		return err
	}
	return nil
}
