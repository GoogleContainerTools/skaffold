/*
 Copyright 2019 Knative Authors LLC
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

package templating

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/knative/pkg/apis"
)

const parameterSubstitution = "[_a-zA-Z][_a-zA-Z0-9.-]*"

func ValidateVariable(name, value, prefix, contextPrefix, locationName, path string, vars map[string]struct{}) *apis.FieldError {
	if vs, present := extractVariablesFromString(value, contextPrefix+prefix); present {
		for _, v := range vs {
			if _, ok := vars[v]; !ok {
				return &apis.FieldError{
					Message: fmt.Sprintf("non-existent variable in %q for %s %s", value, locationName, name),
					Paths:   []string{path + "." + name},
				}
			}
		}
	}
	return nil
}

func extractVariablesFromString(s, prefix string) ([]string, bool) {
	pattern := fmt.Sprintf("\\$({%s.(?P<var>%s)})", prefix, parameterSubstitution)
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return []string{}, false
	}
	vars := make([]string, len(matches))
	for i, match := range matches {
		groups := matchGroups(match, re)
		// foo -> foo
		// foo.bar -> foo
		// foo.bar.baz -> foo
		vars[i] = strings.SplitN(groups["var"], ".", 2)[0]
	}
	return vars, true
}

func matchGroups(matches []string, pattern *regexp.Regexp) map[string]string {
	groups := make(map[string]string)
	for i, name := range pattern.SubexpNames()[1:] {
		groups[name] = matches[i+1]
	}
	return groups
}

func ApplyReplacements(in string, replacements map[string]string) string {
	for k, v := range replacements {
		in = strings.Replace(in, fmt.Sprintf("${%s}", k), v, -1)
	}
	return in
}
