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
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/watch"
	fake_testing "k8s.io/client-go/testing"
)

type T struct {
	*testing.T
}

type ForTester interface {
	ForTest(t *testing.T)
}

func (t *T) Override(dest, tmp interface{}) {
	err := override(t.T, dest, tmp)
	if err != nil {
		t.Errorf("temporary override value is invalid: %v", err)
		return
	}

	if forTester, ok := tmp.(ForTester); ok {
		forTester.ForTest(t.T)
	}
}

func (t *T) CheckMatches(pattern, actual string) {
	t.Helper()
	if matches, _ := regexp.MatchString(pattern, actual); !matches {
		t.Errorf("expected output %s to match: %s", actual, pattern)
	}
}

func (t *T) CheckContains(expected, actual string) {
	t.Helper()
	CheckContains(t.T, expected, actual)
}

func (t *T) CheckNil(actual interface{}) {
	t.Helper()

	if !isNil(actual) {
		t.Errorf("expected `nil`, but was `%+v`", actual)
	}
}

func (t *T) CheckNotNil(actual interface{}) {
	t.Helper()

	if isNil(actual) {
		t.Error("expected `not nil`, but was `nil`")
	}
}

func isNil(actual interface{}) bool {
	return actual == nil || (reflect.ValueOf(actual).Kind() == reflect.Ptr && reflect.ValueOf(actual).IsNil())
}

func (t *T) CheckTrue(actual bool) {
	t.Helper()
	if !actual {
		t.Error("expected `true`, but was `false`")
	}
}

func (t *T) CheckFalse(actual bool) {
	t.Helper()
	if actual {
		t.Error("expected `false`, but was `true`")
	}
}

func (t *T) CheckEmpty(actual interface{}) {
	t.Helper()

	var len int
	v := reflect.ValueOf(actual)
	switch v.Kind() {
	case reflect.Array:
		len = v.Len()
	case reflect.Chan:
		len = v.Len()
	case reflect.Map:
		len = v.Len()
	case reflect.Slice:
		len = v.Len()
	case reflect.String:
		len = v.Len()
	default:
		t.Errorf("expected `empty`, but was `%+v`", actual)
		return
	}

	if len != 0 {
		t.Errorf("expected `empty`, but was `%+v`", actual)
	}
}

func (t *T) CheckDeepEqual(expected, actual interface{}, opts ...cmp.Option) {
	t.Helper()
	CheckDeepEqual(t.T, expected, actual, opts...)
}

func (t *T) CheckErrorAndDeepEqual(shouldErr bool, err error, expected, actual interface{}, opts ...cmp.Option) {
	t.Helper()
	CheckErrorAndDeepEqual(t.T, shouldErr, err, expected, actual, opts...)
}

func (t *T) CheckError(shouldErr bool, err error) {
	t.Helper()
	CheckError(t.T, shouldErr, err)
}

// CheckErrorContains checks that an error is not nil and contains
// a given message.
func (t *T) CheckErrorContains(message string, err error) {
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

// SetArgs override os.Args for the duration of a test.
func (t *T) SetArgs(args []string) {
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	os.Args = args
}

// SetStdin replaces os.Stdin with a given content.
func (t *T) SetStdin(content []byte) {
	origStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = origStdin })

	tmpFile := t.TempFile("stdin", content)
	file, err := os.Open(tmpFile)
	if err != nil {
		t.Error("unable to read temp file")
		return
	}

	os.Stdin = file
}

func (t *T) TempFile(prefix string, content []byte) string {
	return TempFile(t.T, prefix, content)
}

func (t *T) NewTempDir() *TempDir {
	return NewTempDir(t.T)
}

func (t *T) Chdir(dir string) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal("unable to get current directory")
	}

	t.Cleanup(func() {
		if err = os.Chdir(pwd); err != nil {
			t.Fatal("unable to reset working direcrory")
		}
	})

	if err = os.Chdir(dir); err != nil {
		t.Fatal("unable to change current directory")
	}
}

func Abs(t *testing.T, path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Failed to get absolute path for file %s: %s", path, absPath)
	}
	return absPath
}

func Run(t *testing.T, name string, f func(t *T)) {
	if name == "" {
		name = t.Name()
	}

	t.Run(name, func(tt *testing.T) {
		f(&T{T: tt})
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

func CheckError(t *testing.T, shouldErr bool, err error) {
	t.Helper()
	if err := checkErr(shouldErr, err); err != nil {
		t.Error(err)
	}
}

func EnsureTestPanicked(t *testing.T) {
	if recover() == nil {
		t.Errorf("should have panicked")
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

// ServeFile serves a file with http. Returns the url to the file and a teardown
// function that should be called to properly stop the server.
func ServeFile(t *testing.T, content []byte) string {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))

	t.Cleanup(ts.Close)

	return ts.URL
}

func override(t *testing.T, dest, tmp interface{}) (err error) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				t.Fatalf("unable to override value: %s", x)
			case error:
				t.Fatalf("unable to override value: %s", x)
			default:
				t.Fatal("unable to override value")
			}
		}
	}()

	dValue := reflect.ValueOf(dest).Elem()

	// Save current value
	curValue := reflect.New(dValue.Type()).Elem()
	t.Cleanup(func() { dValue.Set(curValue) })

	// Set to temporary value
	curValue.Set(dValue)

	var tmpV reflect.Value
	if tmp == nil {
		tmpV = reflect.Zero(dValue.Type())
	} else {
		tmpV = reflect.ValueOf(tmp)
	}
	dValue.Set(tmpV)

	return nil
}

// SetupFakeWatcher helps set up a fake Kubernetes watcher
func SetupFakeWatcher(w watch.Interface) func(a fake_testing.Action) (handled bool, ret watch.Interface, err error) {
	return func(a fake_testing.Action) (handled bool, ret watch.Interface, err error) {
		return true, w, nil
	}
}
