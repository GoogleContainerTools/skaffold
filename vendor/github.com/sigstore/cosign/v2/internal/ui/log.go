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
	"context"
	"fmt"
)

func (w *Env) infof(msg string, a ...any) {
	msg = fmt.Sprintf(msg, a...)
	fmt.Fprintln(w.Stderr, msg)
}

// Infof logs an informational message. It works like fmt.Printf, except that it
// always has a trailing newline.
func Infof(ctx context.Context, msg string, a ...any) {
	getEnv(ctx).infof(msg, a...)
}

func (w *Env) warnf(msg string, a ...any) {
	msg = fmt.Sprintf(msg, a...)
	fmt.Fprintf(w.Stderr, "WARNING: %s\n", msg)
}

// Warnf logs a warning message (prefixed by "WARNING:"). It works like
// fmt.Printf, except that it always has a trailing newline.
func Warnf(ctx context.Context, msg string, a ...any) {
	getEnv(ctx).warnf(msg, a...)
}
