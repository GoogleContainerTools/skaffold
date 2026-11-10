//go:build !noasm && gc && arm64 && freebsd

package cpu

// getisar0 is implemented in cpu_arm64_freebsd.s.
func getisar0() uint64

// FreeBSD emulates user space reads of ID_AA64ISAR0_EL1. Its SHA1 field, bits
// [11:8], is non zero when the SHA1 instructions are implemented.
func init() {
	ARM64.HasSHA1 = (getisar0()>>8)&0xf != 0
}
