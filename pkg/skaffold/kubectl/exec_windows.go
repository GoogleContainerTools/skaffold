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

// Cmd represents an external command being prepared to run within a job object
type Cmd struct {
	*exec.Cmd
	handle windows.Handle
	ctx    context.Context
}

// CommandContext creates a new Cmd
func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	return &Cmd{Cmd: exec.CommandContext(ctx, name, arg...), ctx: ctx}
}

// Start starts the specified command in a job object but does not wait for it to complete
func (c *Cmd) Start() error {
	handle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("could not create job object: %w", err)
	}

	// https://gist.github.com/hallazzang/76f3970bfc949831808bbebc8ca15209
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(
		handle,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info))); err != nil {
		return fmt.Errorf("could not set information job object: %w", err)
	}

	if err := c.Cmd.Start(); err != nil {
		return fmt.Errorf("could not start the command: %w", err)
	}

	// Use `unsafe` to extract the process handle.
	processHandle := (*struct {
		Pid    int
		handle windows.Handle
	})(unsafe.Pointer(c.Process)).handle

	if err := windows.AssignProcessToJobObject(handle, processHandle); err != nil {
		return fmt.Errorf("could not assign job object: %w", err)
	}

	c.handle = handle
	go func() {
		<-c.ctx.Done()
		c.Terminate()
	}()

	return nil
}

// Run starts the specified command in a job object and waits for it to complete
func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// Terminate closes the job object handle which kills all connected processes
func (c *Cmd) Terminate() error {
	return windows.CloseHandle(c.handle)
}
