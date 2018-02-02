/*
Copyright 2018 Google LLC

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

package test

import (
	"io"
	"os/exec"
)

type FakeRunCommand struct {
	stdout string
	stderr string
	err    error
}

func (f *FakeRunCommand) RunCommand(*exec.Cmd, io.Reader) ([]byte, []byte, error) {
	return []byte(f.stdout), []byte(f.stderr), f.err
}

func NewFakeRunCommand(stdout, stderr string, err error) *FakeRunCommand {
	return &FakeRunCommand{stdout, stderr, err}
}
