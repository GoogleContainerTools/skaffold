/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package configmap

import "reflect"

// TypeFilter accepts instances of types to check against and returns a function transformer that would only let
// the call to f through if value is assignable to any one of types of ts. Example:
//
// F := configmap.TypeFilter(&config.Domain{})(f)
//
// The result is a function F(name string, value interface{}) that will call the underlying function
// f(name, value) iff value is a *config.Domain
func TypeFilter(ts ...interface{}) func(func(string, interface{})) func(string, interface{}) {
	return func(f func(string, interface{})) func(string, interface{}) {
		return func(name string, value interface{}) {
			satisfies := false
			for _, t := range ts {
				t := reflect.TypeOf(t)
				if reflect.TypeOf(value).AssignableTo(t) {
					satisfies = true
					break
				}
			}
			if satisfies {
				f(name, value)
			}
		}
	}
}
