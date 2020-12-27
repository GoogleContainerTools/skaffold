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

import "sort"

type unit struct{}

// StringSet helps to de-duplicate a set of strings.
type StringSet map[string]unit

// NewStringSet returns a new StringSet object.
func NewStringSet() StringSet {
	return make(map[string]unit)
}

// Insert adds strings to the set.
func (s StringSet) Insert(strings ...string) {
	for _, item := range strings {
		s[item] = unit{}
	}
}

// ToList returns the sorted list of inserted strings.
func (s StringSet) ToList() []string {
	var res []string
	for item := range s {
		res = append(res, item)
	}
	sort.Strings(res)
	return res
}

func (s StringSet) Contains(str string) bool {
	_, ok := s[str]
	return ok
}
