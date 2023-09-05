//go:build !windows && !plan9 && !netbsd && !aix && !illumos && !solaris && !js
// +build !windows,!plan9,!netbsd,!aix,!illumos,!solaris,!js

package copy

import (
	"os"
	"syscall"
	"testing"
)

func setup(m *testing.M) {
	os.RemoveAll("test/data.copy")
	os.MkdirAll("test/data.copy", os.ModePerm)
	os.Symlink("test/data/case01", "test/data/case03/case01")
	os.Chmod("test/data/case07/dir_0555", 0o555)
	os.Chmod("test/data/case07/file_0444", 0o444)
	syscall.Mkfifo("test/data/case11/foo/bar", 0o555)
	Copy("test/data/case18/assets", "test/data/case18/assets.backup")
}
