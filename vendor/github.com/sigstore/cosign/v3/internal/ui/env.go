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
	"bytes"
	"context"
	"io"
	"os"
)

// An Env is the environment that the CLI exists in.
//
// It contains handles to STDERR and STDIN. Eventually, it will contain
// configuration pertaining to the current invocation (e.g., is this a terminal
// or not).
//
// UI methods should be defined on an Env. Then, the Env can be
// changed for easy testing. The Env will be retrieved from the current
// application context.
type Env struct {
	Stderr io.Writer
	Stdin  io.Reader
}

// defaultEnv returns the default environment (writing to os.Stderr and
// reading from os.Stdin).
func defaultEnv() *Env {
	return &Env{
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}
}

type ctxKey struct{}

func (c ctxKey) String() string {
	return "cosign/ui:env"
}

var ctxKeyEnv = ctxKey{}

// getEnv gets the environment from ctx.
//
// If ctx does not contain an environment, getEnv returns the default
// environment (see defaultEnvironment).
func getEnv(ctx context.Context) *Env {
	e, ok := ctx.Value(ctxKeyEnv).(*Env)
	if !ok {
		return defaultEnv()
	}
	return e
}

// WithEnv adds the environment to the context.
func WithEnv(ctx context.Context, e *Env) context.Context {
	return context.WithValue(ctx, ctxKeyEnv, e)
}

type WriteFunc func(string)
type callbackFunc func(context.Context, WriteFunc)

// RunWithTestCtx runs the provided callback in a context with the UI
// environment swapped out for one that allows for easy testing and captures
// STDOUT.
//
// The callback has access to a function that writes to the test STDIN.
func RunWithTestCtx(callback callbackFunc) string {
	var stdin bytes.Buffer
	var stderr bytes.Buffer
	e := Env{&stderr, &stdin}

	ctx := WithEnv(context.Background(), &e)
	write := func(msg string) { stdin.WriteString(msg) }
	callback(ctx, write)

	return stderr.String()
}
