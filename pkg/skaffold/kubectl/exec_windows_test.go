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

package kubectl

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

func TestGetHandleFromProcess(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("getHandleFromProcess only works on Windows")
	}

	ctx := context.TODO()
	c := CommandContext(ctx, "kubectl", "--help")
	err := c.Cmd.Start()
	assert.Nil(t, err, "could not start command")

	h, err := getHandleFromProcess(c.Process)
	assert.NotEqual(t, h, windows.InvalidHandle, "handle is invalid")
	assert.Nil(t, err, "could not get handle from process")

	err = c.Cmd.Wait()
	assert.Nil(t, err, "could not wait command")
}
