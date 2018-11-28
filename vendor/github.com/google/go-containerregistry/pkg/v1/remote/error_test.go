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

package remote

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/google/go-containerregistry/pkg/v1/v1util"
)

func TestCheckErrorNil(t *testing.T) {
	tests := []int{
		http.StatusOK,
		http.StatusAccepted,
		http.StatusCreated,
		http.StatusMovedPermanently,
		http.StatusInternalServerError,
	}

	for _, code := range tests {
		resp := &http.Response{StatusCode: code}

		if err := CheckError(resp, code); err != nil {
			t.Errorf("CheckError(%d) = %v", code, err)
		}
	}
}

func TestCheckErrorNotError(t *testing.T) {
	tests := []struct {
		code int
		body string
	}{{
		code: http.StatusBadRequest,
		body: "",
	}, {
		code: http.StatusUnauthorized,
		body: "Not JSON",
	}}

	for _, test := range tests {
		resp := &http.Response{
			StatusCode: test.code,
			Body:       v1util.NopReadCloser(bytes.NewBufferString(test.body)),
		}

		if err := CheckError(resp, http.StatusOK); err == nil {
			t.Errorf("CheckError(%d, %s) = nil, wanted error", test.code, test.body)
		} else if se, ok := err.(*Error); ok {
			t.Errorf("CheckError(%d, %s) = %v, wanted another type", test.code, test.body, se)
		}
	}
}

func TestCheckErrorWithError(t *testing.T) {
	tests := []struct {
		code  int
		error *Error
	}{{
		code: http.StatusBadRequest,
		error: &Error{
			Errors: []Diagnostic{{
				Code:    NameInvalidErrorCode,
				Message: "a message for you",
			}},
		},
	}, {
		code:  http.StatusBadRequest,
		error: &Error{},
	}, {
		code: http.StatusBadRequest,
		error: &Error{
			Errors: []Diagnostic{{
				Code:    NameInvalidErrorCode,
				Message: "a message for you",
			}, {
				Code:    SizeInvalidErrorCode,
				Message: "another message for you",
			}},
		},
	}}

	for _, test := range tests {
		b, err := json.Marshal(test.error)
		if err != nil {
			t.Errorf("json.Marshal(%v) = %v", test.error, err)
		}
		resp := &http.Response{
			StatusCode: test.code,
			Body:       v1util.NopReadCloser(bytes.NewBuffer(b)),
		}

		if err := CheckError(resp, http.StatusOK); err == nil {
			t.Errorf("CheckError(%d, %s) = nil, wanted error", test.code, string(b))
		} else if se, ok := err.(*Error); !ok {
			t.Errorf("CheckError(%d, %s) = %T, wanted *remote.Error", test.code, string(b), se)
		} else if diff := cmp.Diff(test.error, se); diff != "" {
			t.Errorf("CheckError(%d, %s); (-want +got) %s", test.code, string(b), diff)
		}
	}
}
