//go:build !noasm && gc && amd64

#include "textflag.h"

// func calculateDvMaskAVX512(W *[80]uint32, terms *uint32, groups int) uint32
// Requires: AVX, AVX512F
//
// Each group of avx512Terms describes 16 of the bit tests of CalculateDvMask,
// as vectors of word indices, shift counts, expected parities and the masks
// that clear the DVs of a failing test. One pass of the loop runs a group.
//
// terms only has to be 64 byte aligned for speed, as otherwise every load of
// it straddles two cache lines. The word indices are biased by 34, matching
// the two vectors of W loaded below.
TEXT ·calculateDvMaskAVX512(SB), NOSPLIT, $0-28
	MOVQ W+0(FP), AX
	MOVQ terms+8(FP), BX
	MOVQ groups+16(FP), CX
	VMOVDQU32 136(AX), Z0 // W[34..49]
	VMOVDQU32 200(AX), Z1 // W[50..65]
	VPTERNLOGD $0xff, Z2, Z2, Z2 // mask = all ones
	MOVL $1, DX
	VPBROADCASTD DX, Z3

loop:
	VMOVDQU32 0(BX), Z4
	VMOVDQU32 64(BX), Z5
	VPERMI2D Z1, Z0, Z4 // W[a]
	VPERMI2D Z1, Z0, Z5 // W[b]
	VPSRLVD 128(BX), Z4, Z4 // >> ka
	VPSRLVD 192(BX), Z5, Z5 // >> kb
	VPTERNLOGD $0x96, 256(BX), Z5, Z4 // ^ e
	VPTESTMD Z3, Z4, K1 // the terms whose bit test fails
	VPANDD 320(BX), Z2, K1, Z2 // clear their DVs
	ADDQ $384, BX
	DECQ CX
	JNZ loop

	// AND the 16 lanes together, first the 128 bit lanes and then the
	// words within them.
	VSHUFI64X2 $0x4e, Z2, Z2, Z4
	VPANDD Z4, Z2, Z2
	VSHUFI64X2 $0xb1, Z2, Z2, Z4
	VPANDD Z4, Z2, Z2
	VPSHUFD $0x4e, Z2, Z4
	VPANDD Z4, Z2, Z2
	VPSHUFD $0xb1, Z2, Z4
	VPANDD Z4, Z2, Z2
	VMOVD X2, AX
	VZEROUPPER
	MOVL AX, ret+24(FP)
	RET
