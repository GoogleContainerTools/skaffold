/*
Copyright 2018 The Skaffold Authors

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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("%T differ (-got, +want): %s", expected, diff)
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

// TempDir creates a temporary directory. Returns its name and a teardown function
// that should be called to properly delete the directory content.
func TempDir(t *testing.T) (name string, tearDown func()) {
	dir, err := ioutil.TempDir("", "skaffold")
	if err != nil {
		t.Error(err)
	}

	return dir, func() {
		os.RemoveAll(dir)
	}
}

// TempFile creates a temporary file with a given content. Returns the file name
// and a teardown function that should be called to properly delete the file.
func TempFile(t *testing.T, prefix string, content []byte) (name string, tearDown func()) {
	file, err := ioutil.TempFile("", prefix)
	if err != nil {
		t.Error(err)
	}

	if err = ioutil.WriteFile(file.Name(), content, 0644); err != nil {
		t.Error(err)
	}

	return file.Name(), func() {
		syscall.Unlink(file.Name())
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
