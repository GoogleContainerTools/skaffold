/*
Copyright 2019 The Skaffold Authors

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
	"context"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

const (
	file      = "source/file"
	gcsFile   = "gs://bucket/file"
	folder    = "source/"
	gcsFolder = "gs://bucket/folder/"
)

func TestCopy(t *testing.T) {
	tests := []struct {
		src string
		dst string

		description string
		commands    Command
		recursive   bool
		shouldErr   bool
	}{
		{
			src:         file,
			dst:         gcsFile,
			description: "copy single file",
			commands:    testutil.CmdRunOut(fmt.Sprintf("gsutil cp %s %s", file, gcsFile), "logs"),
		},
		{
			src:         folder,
			dst:         gcsFolder,
			recursive:   true,
			description: "copy recursively",
			commands:    testutil.CmdRunOut(fmt.Sprintf("gsutil cp -r %s %s", folder, gcsFolder), "logs"),
		},
		{
			src:         file,
			dst:         gcsFile,
			description: "copy failed",
			commands:    testutil.CmdRunOutErr(fmt.Sprintf("gsutil cp %s %s", file, gcsFile), "logs", fmt.Errorf("file not found")),
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&DefaultExecCommand, test.commands)

			gcs := Gsutil{}
			err := gcs.Copy(context.Background(), test.src, test.dst, test.recursive)

			t.CheckError(test.shouldErr, err)
		})
	}
}
