package cpu

import "encoding/binary"

const (
	_AT_HWCAP = 16

	// hwcapSHA1 is HWCAP_SHA1 on linux/arm64.
	hwcapSHA1 = 1 << 5
)

// hwcapFromAuxv returns AT_HWCAP from an auxiliary vector of tag and value
// pairs, and whether it was present.
func hwcapFromAuxv(auxv []uintptr) (uint64, bool) {
	for i := 0; i+1 < len(auxv); i += 2 {
		if auxv[i] == _AT_HWCAP {
			return uint64(auxv[i+1]), true
		}
	}
	return 0, false
}

// hwcapFromProcAuxv does the same for the contents of /proc/self/auxv on a
// 64-bit little endian system.
func hwcapFromProcAuxv(buf []byte) (uint64, bool) {
	for ; len(buf) >= 16; buf = buf[16:] {
		if binary.LittleEndian.Uint64(buf) == _AT_HWCAP {
			return binary.LittleEndian.Uint64(buf[8:]), true
		}
	}
	return 0, false
}
