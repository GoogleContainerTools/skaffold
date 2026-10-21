//go:build !noasm && gc && arm64 && linux

package cpu

import (
	"os"
	_ "unsafe" // for go:linkname
)

// runtime_getAuxv returns the auxiliary vector the kernel passed to the
// process. The runtime keeps it reachable for golang.org/x/sys/cpu and
// others, see go.dev/issue/57336 and go.dev/issue/67401.
//
//go:linkname runtime_getAuxv runtime.getAuxv
func runtime_getAuxv() []uintptr

// Linux, and so Android, reports the SHA1 instructions through HWCAP, as
// not every kernel lets user space read the ID registers.
func init() {
	hwcap, ok := hwcapFromAuxv(runtime_getAuxv())
	if !ok {
		// The procfs copy may not be readable in restricted environments,
		// in which case the generic implementation is used.
		if buf, err := os.ReadFile("/proc/self/auxv"); err == nil {
			hwcap, _ = hwcapFromProcAuxv(buf)
		}
	}
	ARM64.HasSHA1 = hwcap&hwcapSHA1 != 0
}
