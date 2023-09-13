//go:build windows || plan9 || netbsd || aix || illumos || solaris || js
// +build windows plan9 netbsd aix illumos solaris js

package copy

import (
	"os"
	"testing"
)

func setup(m *testing.M) {
	os.RemoveAll("test/data.copy")
	os.MkdirAll("test/data.copy", os.ModePerm)
	os.Symlink("test/data/case01", "test/data/case03/case01")
	os.Chmod("test/data/case07/dir_0555", 0555)
	os.Chmod("test/data/case07/file_0444", 0444)
}
