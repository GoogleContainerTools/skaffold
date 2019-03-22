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

// OutputFilepath represents a output file command line argument.
// OutputFilepath currently does not provide any validation.
type OutputFilepath struct {
	filepathFlag
}

func NewOutputFilepath(value string) *OutputFilepath {
	return &OutputFilepath{filepathFlag{
		path: value,
	}}
}

func (f *OutputFilepath) Type() string {
	return fmt.Sprintf("%T", f)
}

func (f *OutputFilepath) Usage() string {
	return "Path to an output filepath."
}

func (f *OutputFilepath) Set(value string) error {
	return f.SetIfValid(value)
}

func (f *OutputFilepath) String() string {
	return f.path
}
