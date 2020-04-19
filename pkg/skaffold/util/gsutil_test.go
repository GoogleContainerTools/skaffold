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
		src         string
		dst         string
		recursive   bool
		description string
		commands    Command
		shouldErr   bool
	}{
		{src: file,
			dst:         gcsFile,
			description: "copy single file",
			commands:    testutil.CmdRunOut(fmt.Sprintf("gsutil cp %s %s", file, gcsFile), "logs"),
		},
		{src: folder,
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
