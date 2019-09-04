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

package kubectl

import (
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Replacer is used to replace portions of yaml manifests that match a given key.
type Replacer interface {
	Matches(key string) bool

	NewValue(old interface{}) (bool, interface{})

	ObjMatcher() Matcher
}

// Visit recursively visits a list of manifests and applies transformations of them.
func (l *ManifestList) Visit(replacer Replacer) (ManifestList, error) {
	var updated ManifestList

	for _, manifest := range *l {
		m := yaml.MapSlice{}
		if err := yaml.Unmarshal(manifest, &m); err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		if len(m) == 0 {
			continue
		}

		recursiveVisit(&m, replacer)

		updatedManifest, err := yaml.Marshal(m)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling yaml")
		}

		updated = append(updated, updatedManifest)
	}

	return updated, nil
}

func recursiveVisit(i interface{}, replacer Replacer) {
	fmt.Printf("\n%T -> %v\n", i, i)
	switch t := i.(type) {
	case []interface{}:
		for _, v := range t {
			recursiveVisit(v, replacer)
		}
	case yaml.MapSlice:
		for _, mi := range t {
			recursiveVisit(&mi, replacer)
		}
	case *yaml.MapSlice:
		for _, mi := range *t {
				recursiveVisit(&mi, replacer)
		}
	case *yaml.MapItem:
		key := t.Key.(string)
		switch {
		case replacer.ObjMatcher() != nil && replacer.ObjMatcher().IsMatchKey(key):
			if !replacer.ObjMatcher().Matches(t.Value) {
				return
			}
		case replacer.Matches(key):
			fmt.Println("\nnnnnn came here")
			ok, newValue := replacer.NewValue(t.Value)
			if ok {
				fmt.Println("changed")
				t.Value = newValue
			}
		default:
			recursiveVisit(t.Value, replacer)
		}
	}


	//	for i, mi := range ms {
	//	k := mi.Key
	//	switch v := mi.Value.(type) {
	//	case []interface{}:
	//		for _, ms := range v {
	//			switch v := ms.(type) {
	//			case yaml.MapSlice:
	//				recursiveVisit(v, replacer)
	//			}
	//		}
	//	case yaml.MapSlice:
	//		recursiveVisit(v, replacer)
	//	case interface{}:
	//			key := k.(string)
	//			switch {
	//			case replacer.ObjMatcher() != nil && replacer.ObjMatcher().IsMatchKey(key):
	//				if !replacer.ObjMatcher().Matches(v) {
	//					return
	//				}
	//			case replacer.Matches(key):
	//				ok, newValue := replacer.NewValue(v)
	//				if ok {
	//					(&ms[i]).Value = newValue
	//				}
	//			default:
	//			}
	//	}
	//}
}
