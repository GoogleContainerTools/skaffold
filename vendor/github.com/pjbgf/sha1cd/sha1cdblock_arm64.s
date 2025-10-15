//go:build !noasm && gc && arm64 && !amd64

#include "textflag.h"

// License information for the original SHA1 arm64 implemention:
// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found at:
// 	- https://github.com/golang/go/blob/master/LICENSE
//
// Reference implementations:
// 	- https://github.com/noloader/SHA-Intrinsics/blob/master/sha1-arm.c
// 	- https://github.com/golang/go/blob/master/src/crypto/sha1/sha1block_arm64.s

#define HASHUPDATECHOOSE \
	SHA1C	V16.S4, V1, V2 \
	SHA1H	V3, V1 \
	VMOV	V2.B16, V3.B16

#define HASHUPDATEPARITY \
	SHA1P	V16.S4, V1, V2 \
	SHA1H	V3, V1 \
	VMOV	V2.B16, V3.B16

#define HASHUPDATEMAJ \
	SHA1M	V16.S4, V1, V2 \
	SHA1H	V3, V1 \
	VMOV	V2.B16, V3.B16

// func blockARM64(h []uint32, p []byte, m1 []uint32, cs [][5]uint32)
TEXT ·blockARM64(SB), NOSPLIT, $80-96
	MOVD	h_base+0(FP), R0
	MOVD	p_base+24(FP), R1
	MOVD	p_len+32(FP), R2
	MOVD	m1_base+48(FP), R3
	MOVD	cs_base+72(FP), R4

    LSR     $6, R2, R2
    LSL     $6, R2, R2
    ADD     R16, R2, R21

	VLD1.P	16(R0), [V0.S4]
	FMOVS	(R0), F20
	SUB	$16, R0, R0

loop:
	CMP     R16, R21
	BLS     end

	// Load block (p) into 16-bytes vectors.
	VLD1.P	16(R1), [V4.B16]
	VLD1.P	16(R1), [V5.B16]
	VLD1.P	16(R1), [V6.B16]
	VLD1.P	16(R1), [V7.B16]
	
	// Load K constants to V19
	MOVD  $·sha1Ks(SB), R22
	VLD1  (R22), [V19.S4]
                              
	VMOV	V0.B16, V2.B16
	VMOV	V20.S[0], V1
	VMOV	V2.B16, V3.B16
	VDUP	V19.S[0], V17.S4
	
	// Little Endian
	VREV32	V4.B16, V4.B16
	VREV32	V5.B16, V5.B16
	VREV32	V6.B16, V6.B16
	VREV32	V7.B16, V7.B16
	
	// LOAD M1 rounds 0-15
	VST1.P    [V4.S4], (R3)
	VST1.P    [V5.S4], (R3)
	VST1.P    [V6.S4], (R3)
	VST1.P    [V7.S4], (R3)

	// LOAD CS 0
    VST1.P    [V0.S4], (R4)  // ABCD pre-round 0
	VST1.P    V1.S[0], 4(R4) // E pre-round 0

	// Rounds 0-3
	VDUP	V19.S[1], V18.S4
	VADD	V17.S4, V4.S4, V16.S4
	SHA1SU0	V6.S4, V5.S4, V4.S4
	HASHUPDATECHOOSE
	SHA1SU1	V7.S4, V4.S4

	// Rounds 4-7
	VADD	V17.S4, V5.S4, V16.S4
	SHA1SU0	V7.S4, V6.S4, V5.S4
	HASHUPDATECHOOSE
	SHA1SU1	V4.S4, V5.S4
	// LOAD M1 rounds 16-19
	VST1.P    [V4.S4], (R3)

	// Rounds 8-11
	VADD	V17.S4, V6.S4, V16.S4
	SHA1SU0	V4.S4, V7.S4, V6.S4
	HASHUPDATECHOOSE
	SHA1SU1	V5.S4, V6.S4
	// LOAD M1 rounds 20-23
	VST1.P    [V5.S4], (R3)

	// Rounds 12-15
	VADD	V17.S4, V7.S4, V16.S4
	SHA1SU0	V5.S4, V4.S4, V7.S4
	HASHUPDATECHOOSE
	SHA1SU1	V6.S4, V7.S4
	// LOAD M1 rounds 24-27
	VST1.P    [V6.S4], (R3)

	// Rounds 16-19
	VADD	V17.S4, V4.S4, V16.S4
	SHA1SU0	V6.S4, V5.S4, V4.S4
	HASHUPDATECHOOSE
	SHA1SU1	V7.S4, V4.S4
	// LOAD M1 rounds 28-31
	VST1.P    [V7.S4], (R3)

	// Rounds 20-23
	VDUP	V19.S[2], V17.S4
	VADD	V18.S4, V5.S4, V16.S4
	SHA1SU0	V7.S4, V6.S4, V5.S4
	HASHUPDATEPARITY
	SHA1SU1	V4.S4, V5.S4
	// LOAD M1 rounds 32-35
	VST1.P    [V4.S4], (R3)

	// Rounds 24-27
	VADD	V18.S4, V6.S4, V16.S4
	SHA1SU0	V4.S4, V7.S4, V6.S4
	HASHUPDATEPARITY
	SHA1SU1	V5.S4, V6.S4
	// LOAD M1 rounds 36-39
	VST1.P    [V5.S4], (R3)

	// Rounds 28-31
	VADD	V18.S4, V7.S4, V16.S4
	SHA1SU0	V5.S4, V4.S4, V7.S4
	HASHUPDATEPARITY
	SHA1SU1	V6.S4, V7.S4
	// LOAD M1 rounds 40-43
	VST1.P    [V6.S4], (R3)

	// Rounds 32-35
	VADD	V18.S4, V4.S4, V16.S4
	SHA1SU0	V6.S4, V5.S4, V4.S4
	HASHUPDATEPARITY
	SHA1SU1	V7.S4, V4.S4
	// LOAD M1 rounds 44-47
	VST1.P    [V7.S4], (R3)

	// Rounds 36-39
	VADD	V18.S4, V5.S4, V16.S4
	SHA1SU0	V7.S4, V6.S4, V5.S4
	HASHUPDATEPARITY
	SHA1SU1	V4.S4, V5.S4
	// LOAD M1 rounds 48-51
	VST1.P    [V4.S4], (R3)

	// Rounds 44-47
	VDUP	V19.S[3], V18.S4
	VADD	V17.S4, V6.S4, V16.S4
	SHA1SU0	V4.S4, V7.S4, V6.S4
	HASHUPDATEMAJ
	SHA1SU1	V5.S4, V6.S4
	// LOAD M1 rounds 52-55
	VST1.P    [V5.S4], (R3)

	// Rounds 44-47
	VADD	V17.S4, V7.S4, V16.S4
	SHA1SU0	V5.S4, V4.S4, V7.S4
	HASHUPDATEMAJ
	SHA1SU1	V6.S4, V7.S4
	// LOAD M1 rounds 56-59
	VST1.P    [V6.S4], (R3)

	// Rounds 48-51
	VADD	V17.S4, V4.S4, V16.S4
	SHA1SU0	V6.S4, V5.S4, V4.S4
	HASHUPDATEMAJ
	SHA1SU1	V7.S4, V4.S4
	// LOAD M1 rounds 60-63
	VST1.P    [V7.S4], (R3)
	
	// Rounds 52-55
	VADD	V17.S4, V5.S4, V16.S4
	SHA1SU0	V7.S4, V6.S4, V5.S4
	HASHUPDATEMAJ
	SHA1SU1	V4.S4, V5.S4

	// LOAD CS 58
    VST1.P    [V3.S4], (R4)  // ABCD pre-round 56
	VST1.P    V1.S[0], 4(R4) // E pre-round 56

	// Rounds 56-59
	VADD	V17.S4, V6.S4, V16.S4
	SHA1SU0	V4.S4, V7.S4, V6.S4
	HASHUPDATEMAJ
	SHA1SU1	V5.S4, V6.S4

	// Rounds 60-63
	VADD	V18.S4, V7.S4, V16.S4
	SHA1SU0	V5.S4, V4.S4, V7.S4
	HASHUPDATEPARITY
	SHA1SU1	V6.S4, V7.S4

	// LOAD CS 65
    VST1.P    [V3.S4], (R4)  // ABCD pre-round 64
	VST1.P    V1.S[0], 4(R4) // E pre-round 64

	// Rounds 64-67
	VADD	V18.S4, V4.S4, V16.S4
	HASHUPDATEPARITY

	// LOAD M1 rounds 68-79
	VST1.P    [V4.S4], (R3)
	VST1.P    [V5.S4], (R3)
	VST1.P    [V6.S4], (R3)
	VST1.P    [V7.S4], (R3)

	// Rounds 68-71
	VADD	V18.S4, V5.S4, V16.S4
	HASHUPDATEPARITY

	// Rounds 72-75
	VADD	V18.S4, V6.S4, V16.S4
	HASHUPDATEPARITY

	// Rounds 76-79
	VADD	V18.S4, V7.S4, V16.S4
	HASHUPDATEPARITY

	// Add working registers to hash state.
	VADD	V2.S4, V0.S4, V0.S4
	VADD	V1.S4, V20.S4, V20.S4

end:
	// Update h with final hash values.
	VST1.P	[V0.S4], (R0)
	FMOVS	F20, (R0)
	
	RET

DATA ·sha1Ks+0(SB)/4,  $0x5A827999 // K0
DATA ·sha1Ks+4(SB)/4,  $0x6ED9EBA1 // K1
DATA ·sha1Ks+8(SB)/4,  $0x8F1BBCDC // K2
DATA ·sha1Ks+12(SB)/4, $0xCA62C1D6 // K3
GLOBL ·sha1Ks(SB), RODATA, $16
