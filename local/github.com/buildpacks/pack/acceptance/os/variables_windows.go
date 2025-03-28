//go:build acceptance && windows

package os

import (
	"os"
	"regexp"
)

const PackBinaryName = "pack.exe"

var (
	PackBinaryExp   = regexp.MustCompile(`pack-v\d+.\d+.\d+-windows`)
	InterruptSignal = os.Kill
)
