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
	re "regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// RegexEqual matches the string 'actual' against a regex compiled from 'expected'
// If 'expected' is not a valid regex, string comparison is used as fallback
func RegexEqual(expected, actual string) bool {
	if strings.HasPrefix(expected, "!") {
		notExpected := expected[1:]

		return !regexMatch(notExpected, actual)
	}

	return regexMatch(expected, actual)
}

func regexMatch(expected, actual string) bool {
	if actual == expected {
		return true
	}

	matcher, err := re.Compile(expected)
	if err != nil {
		logrus.Infof("context activation criteria '%s' is not a valid regexp, falling back to string", expected)
		return false
	}

	return matcher.MatchString(actual)
}
