// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authn

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

var (
	testDomain, _ = name.NewRegistry("foo.dev", name.WeakValidation)
)

// errorRunner implements runner to always return an execution error.
type errorRunner struct {
	err error
}

// Run implements runner
func (er *errorRunner) Run(*exec.Cmd) error {
	return er.err
}

// printRunner implements runner to write a fixed message to stdout.
type printRunner struct {
	msg string
}

// Run implements runner
func (pr *printRunner) Run(cmd *exec.Cmd) error {
	_, err := cmd.Stdout.Write([]byte(pr.msg))
	return err
}

// errorPrintRunner implements runner to write a fixed message to stdout
// and exit with an error code.
type errorPrintRunner struct {
	msg string
}

// Run implements runner
func (pr *errorPrintRunner) Run(cmd *exec.Cmd) error {
	_, err := cmd.Stdout.Write([]byte(pr.msg))
	if err != nil {
		return err
	}

	return &exec.ExitError{}
}

func TestHelperError(t *testing.T) {
	want := errors.New("fdhskjdfhkjhsf")
	h := &helper{name: "test", domain: testDomain, r: &errorRunner{err: want}}

	if _, got := h.Authorization(); got != want {
		t.Errorf("Authorization(); got %v, want %v", got, want)
	}
}

func TestMagicString(t *testing.T) {
	h := &helper{name: "test", domain: testDomain, r: &errorPrintRunner{msg: magicNotFoundMessage}}

	got, err := h.Authorization()
	if err != nil {
		t.Errorf("Authorization() = %v", err)
	}

	// When we get the magic not found message we should fall back on anonymous authentication.
	want, _ := Anonymous.Authorization()
	if got != want {
		t.Errorf("Authorization(); got %v, want %v", got, want)
	}
}

func TestGoodOutput(t *testing.T) {
	output := `{"Username": "foo", "Secret": "bar"}`
	h := &helper{name: "test", domain: testDomain, r: &printRunner{msg: output}}

	got, err := h.Authorization()
	if err != nil {
		t.Errorf("Authorization() = %v", err)
	}

	// When we get the magic not found message we should fall back on anonymous authentication.
	want := "Basic Zm9vOmJhcg=="
	if got != want {
		t.Errorf("Authorization(); got %v, want %v", got, want)
	}
}

func TestBadOutput(t *testing.T) {
	// That extra comma will get ya every time.
	output := `{"Username": "foo", "Secret": "bar",}`
	h := &helper{name: "test", domain: testDomain, r: &printRunner{msg: output}}

	got, err := h.Authorization()
	if err == nil {
		t.Errorf("Authorization() = %v", got)
	}
}
