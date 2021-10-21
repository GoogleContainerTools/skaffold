//go:build !windows
// +build !windows

/*
Copyright 2021 The Skaffold Authors

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

package skaffold

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

// trigger stacktrace dump when skaffold process runs too long
func waitAndTriggerStacktrace(t *testing.T, ctx context.Context, process *os.Process) {
	go func() {
		var d time.Duration = 2 * time.Minute
		select {
		case <-ctx.Done():
			break
		case <-time.After(d):
			t.Log("triggering skaffold stacktrace request")
			process.Signal(syscall.SIGUSR1)
		}
	}()
}
