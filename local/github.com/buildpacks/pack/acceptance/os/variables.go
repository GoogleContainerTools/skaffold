//go:build acceptance && !windows

package os

import "os"

const PackBinaryName = "pack"

var InterruptSignal = os.Interrupt
