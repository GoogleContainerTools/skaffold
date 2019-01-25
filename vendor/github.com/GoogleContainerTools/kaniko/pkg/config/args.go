/*
Copyright 2018 Google LLC

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

package config

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// This type is used to supported passing in multiple flags
type multiArg []string

// Now, for our new type, implement the two methods of
// the flag.Value interface...
// The first method is String() string
func (b *multiArg) String() string {
	return strings.Join(*b, ",")
}

// The second method is Set(value string) error
func (b *multiArg) Set(value string) error {
	logrus.Debugf("appending to multi args %s", value)
	*b = append(*b, value)
	return nil
}

func (b *multiArg) Type() string {
	return "multi-arg type"
}
