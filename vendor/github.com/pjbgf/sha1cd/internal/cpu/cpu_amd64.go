//go:build !noasm && gc && amd64

package cpu

// cpuid and xgetbv are implemented in cpu_amd64.s.
func cpuid(eaxArg, ecxArg uint32) (eax, ebx, ecx, edx uint32)
func xgetbv() (eax, edx uint32)

func init() {
	const (
		// CPUID EAX=1: ECX
		ssse3   = 1 << 9
		sse41   = 1 << 19
		osxsave = 1 << 27
		avx     = 1 << 28

		// CPUID EAX=7, ECX=0: EBX
		bmi1    = 1 << 3
		avx2    = 1 << 5
		bmi2    = 1 << 8
		avx512f = 1 << 16
		sha     = 1 << 29

		// XCR0
		xmmState      = 1 << 1
		ymmState      = 1 << 2
		opmaskState   = 1 << 5
		zmmHi256State = 1 << 6
		hi16ZMMState  = 1 << 7
		zmmState      = opmaskState | zmmHi256State | hi16ZMMState
	)

	maxID, _, _, _ := cpuid(0, 0)
	if maxID < 1 {
		return
	}

	_, _, ecx1, _ := cpuid(1, 0)
	X86.HasSSSE3 = ecx1&ssse3 != 0
	X86.HasSSE41 = ecx1&sse41 != 0

	// VEX encoded instructions also need the OS to preserve the YMM state,
	// which XGETBV reports once OSXSAVE says it is available.
	// macOS enables the ZMM state lazily, so XCR0 may not report it and
	// AVX-512 is then left unused, which is safe.
	var osYMM, osZMM bool
	if ecx1&(osxsave|avx) == osxsave|avx {
		xcr0, _ := xgetbv()
		osYMM = xcr0&(xmmState|ymmState) == xmmState|ymmState
		osZMM = osYMM && xcr0&zmmState == zmmState
	}
	X86.HasAVX = osYMM

	if maxID < 7 {
		return
	}
	_, ebx7, _, _ := cpuid(7, 0)
	X86.HasAVX2 = osYMM && ebx7&avx2 != 0
	X86.HasAVX512F = osZMM && ebx7&avx512f != 0
	X86.HasBMI1 = ebx7&bmi1 != 0
	X86.HasBMI2 = ebx7&bmi2 != 0
	X86.HasSHA = ebx7&sha != 0
}
