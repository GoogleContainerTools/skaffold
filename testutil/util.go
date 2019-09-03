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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
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
	teardownActions []func()
}

type ForTester interface {
	ForTest(t *testing.T)
}

func (t *T) Override(dest, tmp interface{}) {
	teardown, err := override(t.T, dest, tmp)
	if err != nil {
		t.Errorf("temporary override value is invalid: %v", err)
		return
	}

	if forTester, ok := tmp.(ForTester); ok {
		forTester.ForTest(t.T)
	}

	t.teardownActions = append(t.teardownActions, teardown)
}

func (t *T) CopyFile(src, dst string) {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		t.Fatalf("can't read source file: %s: %s", src, err)
	}
	err = ioutil.WriteFile(dst, content, 0666)
	if err != nil {
		t.Fatalf("failed to copy file %s to %s: %s", src, dst, err)
	}
	t.teardownActions = append(t.teardownActions, func() {
		if err := os.Remove(dst); err != nil {
			t.Errorf("failed to remove %s: %s", dst, err)
		}
	})
}

func (t *T) CheckMatches(pattern, actual string) {
	t.T.Helper()
	if matches, _ := regexp.MatchString(pattern, actual); !matches {
		t.Errorf("expected output %s to match: %s", actual, pattern)
	}
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

func Run(t *testing.T, name string, f func(t *T)) {
	if name == "" {
		name = t.Name()
	}

	t.Run(name, func(tt *testing.T) {
		testWrapper := &T{
			T: tt,
		}

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
func ServeFile(t *testing.T, content []byte) (url string, tearDown func()) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(content)
	}))

	return ts.URL, ts.Close
}

func override(t *testing.T, dest, tmp interface{}) (f func(), err error) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			f = nil
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
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

	return func() {
		defer func() {
			if r := recover(); r != nil {
				t.Error("panic while restoring original value")
			}
		}()
		dValue.Set(curValue)
	}, nil
}

// SetupFakeWatcher helps set up a fake Kubernetes watcher
func SetupFakeWatcher(w watch.Interface) func(a fake_testing.Action) (handled bool, ret watch.Interface, err error) {
	return func(a fake_testing.Action) (handled bool, ret watch.Interface, err error) {
		return true, w, nil
	}
}
