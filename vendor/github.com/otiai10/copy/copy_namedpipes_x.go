//go:build windows || plan9 || netbsd || aix || illumos || solaris || js
// +build windows plan9 netbsd aix illumos solaris js

package copy

import (
	"os"
)

// TODO: check plan9 netbsd aix illumos solaris in future

// pcopy is for just named pipes. Windows doesn't support them
func pcopy(dest string, info os.FileInfo) error {
	return nil
}
