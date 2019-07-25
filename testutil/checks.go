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

package testutil

import (
	"reflect"
)

func (t *T) CheckErrorAndTypeEquality(shouldErr bool, err error, expected, actual interface{}) {
	t.Helper()

	if err := checkErr(shouldErr, err); err != nil {
		t.Error(err)
		return
	}
	expectedType := reflect.TypeOf(expected)
	actualType := reflect.TypeOf(actual)

	if expectedType != actualType {
		t.Errorf("Types do not match. Expected %s, Actual %s", expectedType, actualType)
		return
	}
}

func (t *T) CheckNoError(err error) {
	t.Helper()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}
