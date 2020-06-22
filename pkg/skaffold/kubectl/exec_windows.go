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
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Cmd struct {
	*exec.Cmd
	handle windows.Handle
	ctx    context.Context
}

type process struct {
	Pid    int
	Handle uintptr
}

func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	return &Cmd{Cmd: exec.Command(name, arg...), ctx: ctx}
}

func (c *Cmd) Start() (err error) {
	var handle windows.Handle
	handle, err = windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}

	go func() {
		<-c.ctx.Done()
		c.Terminate()
	}()

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err = windows.SetInformationJobObject(
		handle,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info))); err != nil {
		return
	}

	if err = c.Cmd.Start(); err != nil {
		return
	}

	if err = windows.AssignProcessToJobObject(
		handle,
		windows.Handle((*process)(unsafe.Pointer(c.Process)).Handle)); err != nil {
		return
	}
	c.handle = handle
	return
}

func (c *Cmd) Terminate() error {
	return windows.CloseHandle(c.handle)
}
