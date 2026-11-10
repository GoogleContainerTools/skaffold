//go:build !noasm && gc && arm64 && windows

package cpu

import "syscall"

// Windows reports the ARMv8 Cryptographic Extension, which includes the SHA1
// instructions, through IsProcessorFeaturePresent.
func init() {
	const _PF_ARM_V8_CRYPTO_INSTRUCTIONS_AVAILABLE = 30

	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("IsProcessorFeaturePresent")
	if proc.Find() != nil {
		return
	}
	ret, _, _ := proc.Call(_PF_ARM_V8_CRYPTO_INSTRUCTIONS_AVAILABLE)
	ARM64.HasSHA1 = ret != 0
}
