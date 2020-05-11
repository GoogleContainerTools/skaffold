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

import (
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

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

// ValidateConstructor checks the type of the constructor it evaluates
// the constructor to be a function with correct signature.
//
// The expectation is for the constructor to receive a single input
// parameter of type corev1.ConfigMap as the input and return two
// values with the second value being of type error
func ValidateConstructor(constructor interface{}) error {
	cType := reflect.TypeOf(constructor)

	if cType.Kind() != reflect.Func {
		return fmt.Errorf("config constructor must be a function")
	}

	if cType.NumIn() != 1 || cType.In(0) != reflect.TypeOf(&corev1.ConfigMap{}) {
		return fmt.Errorf("config constructor must be of the type func(*k8s.io/api/core/v1/ConfigMap) (..., error)")
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()

	if cType.NumOut() != 2 || !cType.Out(1).Implements(errorType) {
		return fmt.Errorf("config constructor must be of the type func(*k8s.io/api/core/v1/ConfigMap) (..., error)")
	}
	return nil
}
