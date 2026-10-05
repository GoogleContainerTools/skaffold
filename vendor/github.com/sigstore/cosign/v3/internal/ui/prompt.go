// Copyright 2023 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

type ErrPromptDeclined struct{}

func (e *ErrPromptDeclined) Error() string {
	return "user declined the prompt"
}

type ErrInvalidInput struct {
	Got     string
	Allowed string
}

func (e *ErrInvalidInput) Error() string {
	return fmt.Sprintf("invalid input %#v (allowed values %v)", e.Got, e.Allowed)
}

func newInvalidYesOrNoInput(got string) error {
	return &ErrInvalidInput{Got: got, Allowed: "y, n"}
}

func (w *Env) prompt() error {
	fmt.Fprint(w.Stderr, "Are you sure you would like to continue? [y/N] ")

	// TODO: what if it's not a terminal?
	r, err := bufio.NewReader(w.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	value := strings.Trim(r, "\r\n")
	switch strings.ToLower(value) {
	case "y":
		return nil
	case "":
		fallthrough // TODO: allow setting default=true?
	case "n":
		return &ErrPromptDeclined{}
	default:
		// TODO: allow retry on invalid input?
		return newInvalidYesOrNoInput(value)
	}
}

// ConfirmContinue prompts the user whether they would like to continue and
// returns the parsed answer.
//
// If the user enters anything other than "y" or "Y", ConfirmContinue returns an
// error.
func ConfirmContinue(ctx context.Context) error {
	return getEnv(ctx).prompt()
}
