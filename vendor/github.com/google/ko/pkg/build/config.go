// Copyright 2021 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package build

import "strings"

// Note: The structs, types, and functions are based upon GoReleaser build
// configuration to have a loosely compatible YAML configuration:
// https://github.com/goreleaser/goreleaser/blob/master/pkg/config/config.go

// StringArray is a wrapper for an array of strings.
type StringArray []string

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (a *StringArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strings []string
	if err := unmarshal(&strings); err != nil {
		var str string
		if err := unmarshal(&str); err != nil {
			return err
		}
		*a = []string{str}
	} else {
		*a = strings
	}
	return nil
}

// FlagArray is a wrapper for an array of strings.
type FlagArray []string

// UnmarshalYAML is a custom unmarshaler that wraps strings in arrays.
func (a *FlagArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var flags []string
	if err := unmarshal(&flags); err != nil {
		var flagstr string
		if err := unmarshal(&flagstr); err != nil {
			return err
		}
		*a = strings.Fields(flagstr)
	} else {
		*a = flags
	}
	return nil
}

// Config contains the build configuration section. The name was changed from
// the original GoReleaser name to match better with the ko naming.
//
// TODO: Introduce support for more fields where possible and where it makes
// /      sense for `ko`, for example ModTimestamp or GoBinary.
type Config struct {
	// ID only serves as an identifier internally
	ID string `yaml:",omitempty"`

	// Dir is the directory out of which the build should be triggered
	Dir string `yaml:",omitempty"`

	// Main points to the main package, or the source file with the main
	// function, in which case only the package will be used for the importpath
	Main string `yaml:",omitempty"`

	// Ldflags and Flags will be used for the Go build command line arguments
	Ldflags StringArray `yaml:",omitempty"`
	Flags   FlagArray   `yaml:",omitempty"`

	// Env allows setting environment variables for `go build`
	Env []string `yaml:",omitempty"`

	// Other GoReleaser fields that are not supported or do not make sense
	// in the context of ko, for reference or for future use:
	// Goos         []string    `yaml:",omitempty"`
	// Goarch       []string    `yaml:",omitempty"`
	// Goarm        []string    `yaml:",omitempty"`
	// Gomips       []string    `yaml:",omitempty"`
	// Targets      []string    `yaml:",omitempty"`
	// Binary       string      `yaml:",omitempty"`
	// Lang         string      `yaml:",omitempty"`
	// Asmflags     StringArray `yaml:",omitempty"`
	// Gcflags      StringArray `yaml:",omitempty"`
	// ModTimestamp string      `yaml:"mod_timestamp,omitempty"`
	// GoBinary     string      `yaml:",omitempty"`
}
