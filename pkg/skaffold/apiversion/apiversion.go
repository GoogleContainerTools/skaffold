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

package apiversion

import (
	"fmt"
	"regexp"

	"github.com/blang/semver"
)

var re = regexp.MustCompile(`^skaffold/v(\d)(?:(alpha|beta)([1-9]?[0-9]))?$`)

// Parse parses a string into a semver.Version.
func Parse(v string) (semver.Version, error) {
	res := re.FindStringSubmatch(v)
	if res == nil {
		return semver.Version{}, fmt.Errorf("%s is an invalid api version", v)
	}
	if res[2] == "" || res[3] == "" {
		return semver.Parse(fmt.Sprintf("%s.0.0", res[1]))
	}
	return semver.Parse(fmt.Sprintf("%s.0.0-%s.%s", res[1], res[2], res[3]))
}
