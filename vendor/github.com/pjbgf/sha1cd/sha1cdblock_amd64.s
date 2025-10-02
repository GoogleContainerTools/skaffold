//go:build !noasm && gc && amd64 && !arm64

#include "textflag.h"

// License information for the original SHA1 arm64 implemention:
// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found at:
// 	- https://github.com/golang/go/blob/master/LICENSE
//
// Reference implementations:
// 	- https://github.com/golang/go/blob/master/src/crypto/sha1/sha1block_amd64.s

#define LOADCS(abcd, e, index, target) \
	VPEXTRD $3, abcd, ((index*20)+0)(target); \
	VPEXTRD $2, abcd, ((index*20)+4)(target); \
	VPEXTRD $1, abcd, ((index*20)+8)(target); \
	VPEXTRD $0, abcd, ((index*20)+12)(target); \
	MOVL e, ((index*20)+16)(target);

#define LOADM1(m1, index, target) \
	VPSHUFD $0x1B, m1, X8; \
	VMOVDQU X8, ((index*16)+0)(target);
	
// func blockAMD64(h []uint32, p []byte, m1 []uint32, cs [][5]uint32)
// Requires: AVX, SHA, SSE2, SSE4.1, SSSE3
TEXT ·blockAMD64(SB), NOSPLIT, $80-96
	MOVQ h_base+0(FP), DI
	MOVQ p_base+24(FP), SI
	MOVQ p_len+32(FP), DX
	MOVQ m1_base+48(FP), R13
	MOVQ cs_base+72(FP), R15
	CMPQ DX, $0x00
	JEQ  done
	ADDQ SI, DX

	// Allocate space on the stack for saving ABCD and E0, and align it to 16 bytes
	LEAQ 15(SP), AX
	MOVQ $0x000000000000000f, CX
	NOTQ CX
	ANDQ CX, AX

	// Load initial hash state
	PINSRD  $0x03, 16(DI), X5
	VMOVDQU (DI), X0
	PAND    upper_mask<>+0(SB), X5
	PSHUFD  $0x1b, X0, X0
	VMOVDQA shuffle_mask<>+0(SB), X7

loop:
	// Save ABCD and E working values
	VMOVDQA X5, (AX)
	VMOVDQA X0, 16(AX)

	// LOAD CS 0
	VPEXTRD $3, X5, R12
	LOADCS(X0, R12, 0, R15)

	// Rounds 0-3
	VMOVDQU   (SI), X1
	PSHUFB    X7, X1
	PADDD     X1, X5
	VMOVDQA   X0, X6
	SHA1RNDS4 $0x00, X5, X0
	LOADM1(X1, 0, R13)

	// Rounds 4-7
	VMOVDQU   16(SI), X2
	PSHUFB    X7, X2
	SHA1NEXTE X2, X6
	VMOVDQA   X0, X5
	SHA1RNDS4 $0x00, X6, X0
	SHA1MSG1  X2, X1
	LOADM1(X2, 1, R13)

	// Rounds 8-11
	VMOVDQU   32(SI), X3
	PSHUFB    X7, X3
	SHA1NEXTE X3, X5
	VMOVDQA   X0, X6
	SHA1RNDS4 $0x00, X5, X0
	SHA1MSG1  X3, X2
	PXOR      X3, X1
	LOADM1(X3, 2, R13)

	// Rounds 12-15
	VMOVDQU   48(SI), X4
	PSHUFB    X7, X4
	SHA1NEXTE X4, X6
	VMOVDQA   X0, X5
	SHA1MSG2  X4, X1
	SHA1RNDS4 $0x00, X6, X0
	SHA1MSG1  X4, X3
	PXOR      X4, X2
	LOADM1(X4, 3, R13)

	// Rounds 16-19
	SHA1NEXTE X1, X5
	VMOVDQA   X0, X6
	SHA1MSG2  X1, X2
	SHA1RNDS4 $0x00, X5, X0
	SHA1MSG1  X1, X4
	PXOR      X1, X3
	LOADM1(X1, 4, R13)

	// Rounds 20-23
	SHA1NEXTE X2, X6
	VMOVDQA   X0, X5
	SHA1MSG2  X2, X3
	SHA1RNDS4 $0x01, X6, X0
	SHA1MSG1  X2, X1
	PXOR      X2, X4
	LOADM1(X2, 5, R13)

	// Rounds 24-27
	SHA1NEXTE X3, X5
	VMOVDQA   X0, X6
	SHA1MSG2  X3, X4
	SHA1RNDS4 $0x01, X5, X0
	SHA1MSG1  X3, X2
	PXOR      X3, X1
	LOADM1(X3, 6, R13)

	// Rounds 28-31
	SHA1NEXTE X4, X6
	VMOVDQA   X0, X5
	SHA1MSG2  X4, X1
	SHA1RNDS4 $0x01, X6, X0
	SHA1MSG1  X4, X3
	PXOR      X4, X2
	LOADM1(X4, 7, R13)

	// Rounds 32-35
	SHA1NEXTE X1, X5
	VMOVDQA   X0, X6
	SHA1MSG2  X1, X2
	SHA1RNDS4 $0x01, X5, X0
	SHA1MSG1  X1, X4
	PXOR      X1, X3
	LOADM1(X1, 8, R13)

	// Rounds 36-39
	SHA1NEXTE X2, X6
	VMOVDQA   X0, X5
	SHA1MSG2  X2, X3
	SHA1RNDS4 $0x01, X6, X0
	SHA1MSG1  X2, X1
	PXOR      X2, X4
	LOADM1(X2, 9, R13)

	// Rounds 40-43
	SHA1NEXTE X3, X5
	VMOVDQA   X0, X6
	SHA1MSG2  X3, X4
	SHA1RNDS4 $0x02, X5, X0
	SHA1MSG1  X3, X2
	PXOR      X3, X1
	LOADM1(X3, 10, R13)

	// Rounds 44-47
	SHA1NEXTE X4, X6
	VMOVDQA   X0, X5
	SHA1MSG2  X4, X1
	SHA1RNDS4 $0x02, X6, X0
	SHA1MSG1  X4, X3
	PXOR      X4, X2
	LOADM1(X4, 11, R13)

	// Rounds 48-51
	SHA1NEXTE X1, X5
	VMOVDQA   X0, X6
	SHA1MSG2  X1, X2
	SHA1RNDS4 $0x02, X5, X0
	VPEXTRD $0, X5, R12
	SHA1MSG1  X1, X4
	PXOR      X1, X3
	LOADM1(X1, 12, R13)

	// derive pre-round 56's E out of round 51's A.
	VPEXTRD $3, X0, R12
	ROLL $30, R12

	// Rounds 52-55
	SHA1NEXTE X2, X6
	VMOVDQA   X0, X5
	SHA1MSG2  X2, X3
	SHA1RNDS4 $0x02, X6, X0
	SHA1MSG1  X2, X1
	PXOR      X2, X4
	LOADM1(X2, 13, R13)

	// LOAD CS 58 (gathers 56 which will be rectified in Go)
	LOADCS(X0, R12, 1, R15)

	// Rounds 56-59
	SHA1NEXTE X3, X5
	VMOVDQA   X0, X6
	SHA1MSG2  X3, X4
	SHA1RNDS4 $0x02, X5, X0
	VPEXTRD $0, X5, R12
	SHA1MSG1  X3, X2
	PXOR      X3, X1
	LOADM1(X3, 14, R13)

	// derive pre-round 64's E out of round 59's A.
	VPEXTRD $3, X0, R12
	ROLL $30, R12

	// Rounds 60-63
	SHA1NEXTE X4, X6
	VMOVDQA   X0, X5
	SHA1MSG2  X4, X1
	SHA1RNDS4 $0x03, X6, X0
	SHA1MSG1  X4, X3
	PXOR      X4, X2
	LOADM1(X4, 15, R13)

	// LOAD CS 65 (gathers 64 which will be rectified in Go)
	LOADCS(X0, R12, 2, R15)

	// Rounds 64-67
	SHA1NEXTE X1, X5
	VMOVDQA   X0, X6
	SHA1MSG2  X1, X2
	SHA1RNDS4 $0x03, X5, X0
	SHA1MSG1  X1, X4
	PXOR      X1, X3
	LOADM1(X1, 16, R13)

	// Rounds 68-71
	SHA1NEXTE X2, X6
	VMOVDQA   X0, X5
	SHA1MSG2  X2, X3
	SHA1RNDS4 $0x03, X6, X0
	PXOR      X2, X4
	LOADM1(X2, 17, R13)

	// Rounds 72-75
	SHA1NEXTE X3, X5
	VMOVDQA   X0, X6
	SHA1MSG2  X3, X4
	SHA1RNDS4 $0x03, X5, X0
	LOADM1(X3, 18, R13)

	// Rounds 76-79
	SHA1NEXTE X4, X6
	VMOVDQA   X0, X5
	SHA1RNDS4 $0x03, X6, X0
	LOADM1(X4, 19, R13)

	// Add saved E and ABCD
	SHA1NEXTE (AX), X5
	PADDD     16(AX), X0

	// Check if we are done, if not return to the loop
	ADDQ $0x40, SI
	CMPQ SI, DX
	JNE  loop

	// Write the hash state back to digest
	PSHUFD  $0x1b, X0, X0
	VMOVDQU X0, (DI)
	PEXTRD  $0x03, X5, 16(DI)

done:
	RET

DATA upper_mask<>+0(SB)/8, $0x0000000000000000
DATA upper_mask<>+8(SB)/8, $0xffffffff00000000
GLOBL upper_mask<>(SB), RODATA, $16

DATA shuffle_mask<>+0(SB)/8, $0x08090a0b0c0d0e0f
DATA shuffle_mask<>+8(SB)/8, $0x0001020304050607
GLOBL shuffle_mask<>(SB), RODATA, $16
