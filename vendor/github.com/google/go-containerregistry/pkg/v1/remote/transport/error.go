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

package transport

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// The set of query string keys that we expect to send as part of the registry
// protocol. Anything else is potentially dangerous to leak, as it's probably
// from a redirect. These redirects often included tokens or signed URLs.
var paramWhitelist = map[string]struct{}{
	// Token exchange
	"scope":   struct{}{},
	"service": struct{}{},
	// Cross-repo mounting
	"mount": struct{}{},
	"from":  struct{}{},
	// Layer PUT
	"digest": struct{}{},
	// Listing tags and catalog
	"n":    struct{}{},
	"last": struct{}{},
}

// Error implements error to support the following error specification:
// https://github.com/docker/distribution/blob/master/docs/spec/api.md#errors
type Error struct {
	Errors []Diagnostic `json:"errors,omitempty"`
	// The http status code returned.
	StatusCode int
	// The raw body if we couldn't understand it.
	rawBody string
	// The request that failed.
	request *http.Request
}

// Check that Error implements error
var _ error = (*Error)(nil)

// Error implements error
func (e *Error) Error() string {
	prefix := ""
	if e.request != nil {
		prefix = fmt.Sprintf("%s %s: ", e.request.Method, redact(e.request.URL))
	}
	return prefix + e.responseErr()
}

func (e *Error) responseErr() string {
	switch len(e.Errors) {
	case 0:
		if len(e.rawBody) == 0 {
			return fmt.Sprintf("unsupported status code %d", e.StatusCode)
		}
		return fmt.Sprintf("unsupported status code %d; body: %s", e.StatusCode, e.rawBody)
	case 1:
		return e.Errors[0].String()
	default:
		var errors []string
		for _, d := range e.Errors {
			errors = append(errors, d.String())
		}
		return fmt.Sprintf("multiple errors returned: %s",
			strings.Join(errors, "; "))
	}
}

// Temporary returns whether the request that preceded the error is temporary.
func (e *Error) Temporary() bool {
	if len(e.Errors) == 0 {
		return false
	}
	for _, d := range e.Errors {
		// TODO: Include other error types.
		if d.Code != BlobUploadInvalidErrorCode {
			return false
		}
	}
	return true
}

func redact(original *url.URL) *url.URL {
	qs := original.Query()
	for k, v := range qs {
		for i := range v {
			if _, ok := paramWhitelist[k]; !ok {
				// key is not in the whitelist
				v[i] = "REDACTED"
			}
		}
	}
	redacted := *original
	redacted.RawQuery = qs.Encode()
	return &redacted
}

// Diagnostic represents a single error returned by a Docker registry interaction.
type Diagnostic struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message,omitempty"`
	Detail  interface{} `json:"detail,omitempty"`
}

// String stringifies the Diagnostic in the form: $Code: $Message[; $Detail]
func (d Diagnostic) String() string {
	msg := fmt.Sprintf("%s: %s", d.Code, d.Message)
	if d.Detail != nil {
		msg = fmt.Sprintf("%s; %v", msg, d.Detail)
	}
	return msg
}

// ErrorCode is an enumeration of supported error codes.
type ErrorCode string

// The set of error conditions a registry may return:
// https://github.com/docker/distribution/blob/master/docs/spec/api.md#errors-2
const (
	BlobUnknownErrorCode         ErrorCode = "BLOB_UNKNOWN"
	BlobUploadInvalidErrorCode   ErrorCode = "BLOB_UPLOAD_INVALID"
	BlobUploadUnknownErrorCode   ErrorCode = "BLOB_UPLOAD_UNKNOWN"
	DigestInvalidErrorCode       ErrorCode = "DIGEST_INVALID"
	ManifestBlobUnknownErrorCode ErrorCode = "MANIFEST_BLOB_UNKNOWN"
	ManifestInvalidErrorCode     ErrorCode = "MANIFEST_INVALID"
	ManifestUnknownErrorCode     ErrorCode = "MANIFEST_UNKNOWN"
	ManifestUnverifiedErrorCode  ErrorCode = "MANIFEST_UNVERIFIED"
	NameInvalidErrorCode         ErrorCode = "NAME_INVALID"
	NameUnknownErrorCode         ErrorCode = "NAME_UNKNOWN"
	SizeInvalidErrorCode         ErrorCode = "SIZE_INVALID"
	TagInvalidErrorCode          ErrorCode = "TAG_INVALID"
	UnauthorizedErrorCode        ErrorCode = "UNAUTHORIZED"
	DeniedErrorCode              ErrorCode = "DENIED"
	UnsupportedErrorCode         ErrorCode = "UNSUPPORTED"
)

// CheckError returns a structured error if the response status is not in codes.
func CheckError(resp *http.Response, codes ...int) error {
	for _, code := range codes {
		if resp.StatusCode == code {
			// This is one of the supported status codes.
			return nil
		}
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// https://github.com/docker/distribution/blob/master/docs/spec/api.md#errors
	structuredError := &Error{}
	if err := json.Unmarshal(b, structuredError); err != nil {
		structuredError.rawBody = string(b)
	}
	structuredError.StatusCode = resp.StatusCode
	structuredError.request = resp.Request
	return structuredError
}
