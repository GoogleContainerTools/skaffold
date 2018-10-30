/*
Copyright 2018 The Skaffold Authors

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

package util

import (
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

// IsTerminatedError returns true if the error is type exec.ExitError and the corresponding process was terminated by SIGTERM
// This error is given when a exec.Command is ran and terminated with a SIGTERM.
func IsTerminatedError(err error) bool {
	// unwrap to discover original cause
	err = errors.Cause(err)
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	ws := exitError.Sys().(syscall.WaitStatus)
	signal := ws.Signal()
	return signal == syscall.SIGTERM || signal == syscall.SIGKILL
}
