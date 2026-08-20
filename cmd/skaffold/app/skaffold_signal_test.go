//go:build !windows
// +build !windows

/*
Copyright 2025 The Skaffold Authors

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

package app

import (
	"context"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// TestRootContextIgnoresSIGPIPE checks that SIGPIPE does not cancel the root
// context. Before the fix it was passed to signal.NotifyContext, which
// cancelled the run when a registry reset an idle connection during a build.
//
// https://github.com/GoogleContainerTools/skaffold/issues/10106
func TestRootContextIgnoresSIGPIPE(t *testing.T) {
	defer signal.Reset(syscall.SIGPIPE)

	ctx, cancel := rootContext()
	defer cancel()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGPIPE); err != nil {
		t.Fatalf("sending SIGPIPE: %v", err)
	}

	select {
	case <-ctx.Done():
		t.Fatalf("SIGPIPE cancelled the root context: %v", context.Cause(ctx))
	case <-time.After(200 * time.Millisecond):
		// expected: SIGPIPE is ignored and the context stays alive.
	}
}
