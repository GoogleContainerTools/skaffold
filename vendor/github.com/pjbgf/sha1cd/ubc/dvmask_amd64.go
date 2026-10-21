//go:build !noasm && gc && amd64

package ubc

import (
	"unsafe"

	"github.com/pjbgf/sha1cd/internal/cpu"
)

// useAVX512 reports whether calculateDvMaskAVX512 can run. Its only
// instructions outside AVX-512F are VMOVD and VZEROUPPER, which need AVX.
// HasAVX512F is set only when the CPU reports AVX as well.
var useAVX512 = cpu.X86.HasAVX512F

//go:noescape
func calculateDvMaskAVX512(W *[80]uint32, terms *uint32, groups int) uint32

// avx512Lanes is the number of terms a group of avx512Terms describes, and
// avx512Fields the vectors it holds for them.
const (
	avx512Lanes  = 16
	avx512Fields = 6
)

// avx512Groups is how many passes of the kernel loop the table needs.
const avx512Groups = len(avx512Terms) / (avx512Fields * avx512Lanes)

// alignedTerms is a copy of avx512Terms that starts on a cache line. The
// linker aligns data to 32 bytes at most, and a 64 byte load that straddles
// two cache lines costs about twice as much.
var alignedTerms = func() *uint32 {
	buf := make([]uint32, len(avx512Terms)+16)
	off := (64 - uintptr(unsafe.Pointer(&buf[0]))%64) % 64 / 4
	copy(buf[off:], avx512Terms[:])
	return &buf[off]
}()

// CalculateDvMask takes as input an expanded message block and
// verifies the unavoidable bitconditions for all listed DVs. It returns
// a dvmask where each bit belonging to a DV is set if all unavoidable
// bitconditions for that DV have been met.
func CalculateDvMask(W *[80]uint32) uint32 {
	if W == nil {
		return 0
	}
	if useAVX512 {
		return calculateDvMaskAVX512(W, alignedTerms, avx512Groups)
	}
	return calculateDvMaskGeneric(W)
}
