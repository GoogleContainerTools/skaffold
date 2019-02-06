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

package warnings

import (
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"
)

// Warner prints warnings
type Warner func(format string, args ...interface{})

// Printf can be overridden for testing
var Printf = logrus.Warnf

// Collect is used for testing to collect warnings
// instead of printing them
type Collect struct {
	Warnings []string
}

// Warnf collects all the warnings for unit tests
func (l *Collect) Warnf(format string, args ...interface{}) {
	l.Warnings = append(l.Warnings, fmt.Sprintf(format, args...))
	sort.Strings(l.Warnings)
}
