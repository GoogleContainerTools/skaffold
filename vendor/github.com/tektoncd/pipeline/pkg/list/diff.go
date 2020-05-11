/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either extress or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package list

import "golang.org/x/xerrors"

// IsSame will return an error indicating if there are extra or missing strings
// between the required and provided strings, or will return no error if the two
// contain the same values.
func IsSame(required, provided []string) error {
	missing := DiffLeft(required, provided)
	if len(missing) > 0 {
		return xerrors.Errorf("Didn't provide required values: %s", missing)
	}
	extra := DiffLeft(provided, required)
	if len(extra) > 0 {
		return xerrors.Errorf("Provided extra values: %s", extra)
	}
	return nil
}

// DiffLeft will return all strings which are in the left slice of strings but
// not in the right.
func DiffLeft(left, right []string) []string {
	extra := []string{}
	for _, s := range left {
		found := false
		for _, s2 := range right {
			if s == s2 {
				found = true
			}
		}
		if !found {
			extra = append(extra, s)
		}
	}
	return extra
}
