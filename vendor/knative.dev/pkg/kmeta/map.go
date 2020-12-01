/*
Copyright 2020 The Knative Authors

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

package kmeta

// CopyMap makes a copy of the map.
func CopyMap(a map[string]string) map[string]string {
	ret := make(map[string]string, len(a))
	for k, v := range a {
		ret[k] = v
	}
	return ret
}

// UnionMaps returns a map constructed from the union of input maps.
// where values from latter maps win.
func UnionMaps(maps ...map[string]string) map[string]string {
	if len(maps) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(maps[0]))

	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// FilterMap creates a copy of the provided map, filtering out the elements
// that match `filter`.
// nil `filter` is accepted.
func FilterMap(in map[string]string, filter func(string) bool) map[string]string {
	ret := make(map[string]string, len(in))
	for k, v := range in {
		if filter != nil && filter(k) {
			continue
		}
		ret[k] = v
	}
	return ret
}
