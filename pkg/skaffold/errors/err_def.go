/*
<<<<<<< HEAD
Copyright 2019 The Skaffold Authors
=======
Copyright 2020 The Skaffold Authors
>>>>>>> c0434f6aa (add tests)

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
<<<<<<< HEAD
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
=======
}

type ErrDef struct {
	ae proto.ActionableErr
}

func (e ErrDef) Error() string {
	return fmt.Sprintf("%s. %s", e.ae.Message, concatSuggestions(e.Suggestions()))
>>>>>>> c0434f6aa (add tests)
}

func (e ErrDef) StatusCode() proto.StatusCode {
	return e.ae.ErrCode
}

func (e ErrDef) Suggestions() []*proto.Suggestion {
	return e.ae.Suggestions
}

<<<<<<< HEAD
func NewError(err error, ae proto.ActionableErr) ErrDef {
	return ErrDef{
		err: err,
		ae:  ae,
	}
}

func NewErrorWithStatusCode(ae proto.ActionableErr) ErrDef {
=======
func NewError(ae proto.ActionableErr) ErrDef {
>>>>>>> c0434f6aa (add tests)
	return ErrDef{
		ae: ae,
	}
}
