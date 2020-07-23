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

package kubectl

import (
	"context"
	"fmt"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

var jobObject windows.Handle

func init() {
	var err error
	jobObject, err = createJobObject()
	if err != nil {
		panic("unable to create job object: " + err.Error())
	}
}

func createJobObject() (handle windows.Handle, err error) {
	jobObject, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("unable to create job object: %w", err)
	}

	// https://gist.github.com/hallazzang/76f3970bfc949831808bbebc8ca15209
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(
		jobObject,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info))); err != nil {

		return 0, fmt.Errorf("unable to set job object: %w", err)
	}

	return jobObject, nil
}

// Cmd represents an external command being prepared to run within a job object
type Cmd struct {
	*exec.Cmd
}

// CommandContext creates a new Cmd
func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	return &Cmd{Cmd: exec.CommandContext(ctx, name, arg...)}
}

// Start starts the specified command in a job object but does not wait for it to complete
func (c *Cmd) Start() error {
	if err := c.Cmd.Start(); err != nil {
		return fmt.Errorf("unable to start: %w", err)
	}

	handle := (*struct {
		Pid    int
		handle windows.Handle
	})(unsafe.Pointer(c.Cmd.Process)).handle

	if err := windows.AssignProcessToJobObject(jobObject, handle); err != nil {
		return fmt.Errorf("unable assign process to job object: %w", err)
	}

	return nil
}
