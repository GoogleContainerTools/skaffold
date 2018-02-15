/*
Copyright 2018 Google LLC

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
	"fmt"
	"os"
	"reflect"
	"testing"
)

type BadReader struct{}

func (BadReader) Read([]byte) (int, error) { return 0, fmt.Errorf("Bad read") }

type BadWriter struct{}

func (BadWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("Bad write") }

type FakeReaderCloser struct {
	Err error
}

func (f FakeReaderCloser) Close() error             { return nil }
func (f FakeReaderCloser) Read([]byte) (int, error) { return 0, f.Err }

func CheckErrorAndDeepEqual(t *testing.T, shouldErr bool, err error, expected, actual interface{}) {
	if err := checkErr(shouldErr, err); err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%T differ.\nExpected\n%+v\nActual\n%+v", expected, expected, actual)
		return
	}
}

func CheckErrorAndTypeEquality(t *testing.T, shouldErr bool, err error, expected, actual interface{}) {
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

func CheckError(t *testing.T, shouldErr bool, err error) {
	if err := checkErr(shouldErr, err); err != nil {
		t.Error(err)
	}
}

func checkErr(shouldErr bool, err error) error {
	if err == nil && shouldErr {
		return fmt.Errorf("Expected error, but returned none")
	}
	if err != nil && !shouldErr {
		return fmt.Errorf("Unexpected error: %s", err)
	}
	return nil
}

// SetEnvs takes a map of key values to set using os.Setenv and returns
// a function that can be called to reset the envs to their previous values.
func SetEnvs(t *testing.T, envs map[string]string) func(*testing.T) {
	prevEnvs := map[string]string{}
	for key, value := range envs {
		prevEnv := os.Getenv(key)
		prevEnvs[key] = prevEnv
		err := os.Setenv(key, value)
		if err != nil {
			t.Error(err)
		}
	}
	return func(t *testing.T) {
		for key, value := range prevEnvs {
			err := os.Setenv(key, value)
			if err != nil {
				t.Error(err)
			}
		}
	}
}
