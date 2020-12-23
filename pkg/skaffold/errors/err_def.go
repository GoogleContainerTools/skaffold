/*
Copyright 2020 The Skaffold Authors

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

package errors

import (
	"fmt"

	"github.com/GoogleContainerTools/skaffold/proto"
)

type Error interface {
	Error() string
	StatusCode() proto.StatusCode
	Suggestions() []*proto.Suggestion
	Unwrap() error
}

type ErrDef struct {
	err error
	ae  proto.ActionableErr
}

func (e ErrDef) Error() string {
	if s := concatSuggestions(e.Suggestions()); s != "" {
		return fmt.Sprintf("%s. %s", e.ae.Message, concatSuggestions(e.Suggestions()))
	}
	return e.ae.Message
}

func (e ErrDef) Unwrap() error {
	return e.err
}

func (e ErrDef) StatusCode() proto.StatusCode {
	return e.ae.ErrCode
}

func (e ErrDef) Suggestions() []*proto.Suggestion {
	return e.ae.Suggestions
}

// NewError creates an actionable error message preserving the actual error.
func NewError(err error, ae proto.ActionableErr) ErrDef {
	return ErrDef{
		err: err,
		ae:  ae,
	}
}

// NewError creates an actionable error message.
func NewErrorWithStatusCode(ae proto.ActionableErr) ErrDef {
	return ErrDef{
		ae: ae,
	}
}

func IsSkaffoldErr(err error) bool {
	if _, ok := err.(Error); ok {
		return true
	}
	return false
}
