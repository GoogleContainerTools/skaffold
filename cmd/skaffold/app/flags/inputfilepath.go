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
	"fmt"
)

// InputFilepath represents a input file command line argument
// InputFilepath type makes sure the file exists.
type InputFilepath struct {
	filepathFlag
}

func NewInputFilepath(value string) *InputFilepath {
	return &InputFilepath{
		filepathFlag{
			path:        value,
			shouldExist: true,
		}}
}

func (f *InputFilepath) Usage() string {
	return "Path to an input file."
}

func (f *InputFilepath) Type() string {
	return fmt.Sprintf("%T", f)
}

func (f *InputFilepath) Set(value string) error {
	return f.SetIfValid(value)
}

func (f *InputFilepath) String() string {
	return f.path
}
