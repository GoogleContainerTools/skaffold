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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type T struct {
	*testing.T
	teardownActions []func()
}

func (t *T) NewFakeCmd() *FakeCmd {
	return NewFakeCmd(t.T)
}

func (t *T) FakeRunOut(command string, output string) *FakeCmd {
	return FakeRunOut(t.T, command, output)
}

func (t *T) Override(dest, tmp interface{}) {
	teardown := Override(t.T, dest, tmp)
	t.teardownActions = append(t.teardownActions, teardown)
}

func (t *T) CheckContains(expected, actual string) {
	CheckContains(t.T, expected, actual)
}

func (t *T) CheckDeepEqual(expected, actual interface{}, opts ...cmp.Option) {
	CheckDeepEqual(t.T, expected, actual, opts...)
}

func (t *T) CheckErrorAndDeepEqual(shouldErr bool, err error, expected, actual interface{}, opts ...cmp.Option) {
	CheckErrorAndDeepEqual(t.T, shouldErr, err, expected, actual, opts...)
}

func (t *T) CheckError(shouldErr bool, err error) {
	CheckError(t.T, shouldErr, err)
}

func (t *T) CheckErrorContains(message string, err error) {
	CheckErrorContains(t.T, message, err)
}

func (t *T) TempFile(prefix string, content []byte) string {
	name, teardown := TempFile(t.T, prefix, content)
	t.teardownActions = append(t.teardownActions, teardown)
	return name
}

func (t *T) NewTempDir() *TempDir {
	tmpDir, teardown := NewTempDir(t.T)
	t.teardownActions = append(t.teardownActions, teardown)
	return tmpDir
}

func NewTest(t *testing.T) *T {
	return &T{
		T: t,
	}
}

func Run(t *testing.T, name string, f func(t *T)) {
	if name == "" {
		name = t.Name()
	}

	t.Run(name, func(tt *testing.T) {
		testWrapper := NewTest(tt)

		defer func() {
			for _, teardownAction := range testWrapper.teardownActions {
				teardownAction()
			}
		}()

		f(testWrapper)
	})
}

////

func CheckContains(t *testing.T, expected, actual string) {
	t.Helper()
	if !strings.Contains(actual, expected) {
		t.Errorf("expected output %s not found in output: %s", expected, actual)
		return
	}
}

func CheckDeepEqual(t *testing.T, expected, actual interface{}, opts ...cmp.Option) {
	t.Helper()
	if diff := cmp.Diff(actual, expected, opts...); diff != "" {
		t.Errorf("%T differ (-got, +want): %s", expected, diff)
		return
	}
}

func CheckErrorAndDeepEqual(t *testing.T, shouldErr bool, err error, expected, actual interface{}, opts ...cmp.Option) {
	t.Helper()
	if err := checkErr(shouldErr, err); err != nil {
		t.Error(err)
		return
	}
	if diff := cmp.Diff(actual, expected, opts...); diff != "" {
		t.Errorf("%T differ (-got, +want): %s", expected, diff)
		return
	}
}

func CheckErrorAndTypeEquality(t *testing.T, shouldErr bool, err error, expected, actual interface{}) {
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

func CheckError(t *testing.T, shouldErr bool, err error) {
	t.Helper()
	if err := checkErr(shouldErr, err); err != nil {
		t.Error(err)
	}
}

// CheckErrorContains checks that an error is not nil and contains
// a given message.
func CheckErrorContains(t *testing.T, message string, err error) {
	t.Helper()
	if err == nil {
		t.Error("expected error, but returned none")
		return
	}
	if !strings.Contains(err.Error(), message) {
		t.Errorf("expected message [%s] not found in error: %s", message, err.Error())
		return
	}
}

func EnsureTestPanicked(t *testing.T) {
	if recover() == nil {
		t.Errorf("should have panicked")
	}
}

// Chdir changes current directory for a test
func Chdir(t *testing.T, dir string) func() {
	t.Helper()

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal("unable to get current directory")
	}

	err = os.Chdir(dir)
	if err != nil {
		t.Fatal("unable to change current directory")
	}

	return func() {
		if err := os.Chdir(pwd); err != nil {
			t.Fatal("unable to reset current directory")
		}
	}
}

func checkErr(shouldErr bool, err error) error {
	if err == nil && shouldErr {
		return errors.New("expected error, but returned none")
	}
	if err != nil && !shouldErr {
		return fmt.Errorf("unexpected error: %s", err)
	}
	return nil
}

// SetEnvs takes a map of key values to set using os.Setenv and returns
// a function that can be called to reset the envs to their previous values.
func SetEnvs(t *testing.T, envs map[string]string) func() {
	prevEnvs := map[string]string{}
	for key, value := range envs {
		prevEnv := os.Getenv(key)
		prevEnvs[key] = prevEnv
		err := os.Setenv(key, value)
		if err != nil {
			t.Error(err)
		}
	}

	return func() {
		for key, value := range prevEnvs {
			err := os.Setenv(key, value)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

// ServeFile serves a file with http. Returns the url to the file and a teardown
// function that should be called to properly stop the server.
func ServeFile(t *testing.T, content []byte) (url string, tearDown func()) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))

	return ts.URL, ts.Close
}

// Override sets a dest variable to a given value.
// Returns the function to call to restore the variable
// to its original state.
func Override(t *testing.T, dest, tmp interface{}) func() {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Error("temporary value is of invalid type")
		}
	}()

	dValue := reflect.ValueOf(dest).Elem()

	// Save current value
	curValue := reflect.New(dValue.Type()).Elem()
	curValue.Set(dValue)

	// Set to temporary value
	var tmpV reflect.Value
	if tmp == nil {
		tmpV = reflect.Zero(dValue.Type())
	} else {
		tmpV = reflect.ValueOf(tmp)
	}
	dValue.Set(tmpV)

	return func() { dValue.Set(curValue) }
}
