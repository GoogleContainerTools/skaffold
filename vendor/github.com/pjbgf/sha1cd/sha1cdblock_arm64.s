//go:build !noasm && gc && arm64 && !amd64

#include "textflag.h"

#define RoundConst0 $1518500249 // 0x5A827999
#define RoundConst1 $1859775393 // 0x6ED9EBA1
#define RoundConst2 $2400959708 // 0x8F1BBCDC
#define RoundConst3 $3395469782 // 0xCA62C1D6

// FUNC1 f = (b & c) | ((~b) & d)
#define FUNC1(b, c, d) \
	MOVW d, R15; \
	EORW c, R15; \
	ANDW b, R15; \
	EORW d, R15

// FUNC2 f = b ^ c ^ d
#define FUNC2(b, c, d) \
	MOVW b, R15; \
	EORW c, R15; \
	EORW d, R15

// FUNC3 f = (b & c) | (b & d) | (c & d)
#define FUNC3(b, c, d) \
	MOVW b, R27; \
	ORR c, R27, R27; \
	ANDW d, R27, R27; \
	MOVW b, R15; \
	ANDW c, R15, R15; \
	ORR R27, R15, R15

#define FUNC4(b, c, d) FUNC2(b, c, d)
	
#define MIX(a, b, c, d, e, k) \
	RORW $2, b, b; \
	ADDW R15, e, e; \
	MOVW a, R27; \
	RORW $27, R27, R27; \
	MOVW k, R19; \
	ADDW R19, e, e; \
	ADDW R9, e, e; \
	ADDW R27, e, e

#define LOAD(index) \
	MOVWU (index*4)(R16), R9; \
	REVW R9, R9; \
	MOVW R9, (index*4)(RSP)

#define LOADCS(a, b, c, d, e, index) \
	MOVD cs_base+56(FP), R27; \
	MOVW a, ((index*20))(R27); \
	MOVW b, ((index*20)+4)(R27); \
	MOVW c, ((index*20)+8)(R27); \
	MOVW d, ((index*20)+12)(R27); \
	MOVW e, ((index*20)+16)(R27)

#define SHUFFLE(index) \
	MOVW ((index&0xf)*4)(RSP), R9; \
	MOVW (((index-3)&0xf)*4)(RSP), R20; \
	EORW R20, R9; \
	MOVW (((index-8)&0xf)*4)(RSP), R20; \
	EORW R20, R9; \
	MOVW (((index-14)&0xf)*4)(RSP), R20; \
	EORW R20, R9; \
	RORW $31, R9, R9; \
	MOVW R9, ((index&0xf)*4)(RSP)

// LOADM1 stores message word to m1 array.
#define LOADM1(index) \
	MOVD m1_base+32(FP), R27; \
	MOVW ((index&0xf)*4)(RSP), R9; \
	MOVW R9, (index*4)(R27)

#define ROUND1(a, b, c, d, e, index) \
	LOAD(index); \
	FUNC1(b, c, d); \
	MIX(a, b, c, d, e, RoundConst0); \
	LOADM1(index)

#define ROUND1x(a, b, c, d, e, index) \
	SHUFFLE(index); \
	FUNC1(b, c, d); \
	MIX(a, b, c, d, e, RoundConst0); \
	LOADM1(index)

#define ROUND2(a, b, c, d, e, index) \
	SHUFFLE(index); \
	FUNC2(b, c, d); \
	MIX(a, b, c, d, e, RoundConst1); \
	LOADM1(index)

#define ROUND3(a, b, c, d, e, index) \
	SHUFFLE(index); \
	FUNC3(b, c, d); \
	MIX(a, b, c, d, e, RoundConst2); \
	LOADM1(index)

#define ROUND4(a, b, c, d, e, index) \
	SHUFFLE(index); \
	FUNC4(b, c, d); \
	MIX(a, b, c, d, e, RoundConst3); \
	LOADM1(index)

// func blockARM64(dig *digest, p []byte, m1 []uint32, cs [][5]uint32)
TEXT ·blockARM64(SB), NOSPLIT, $64-80
    MOVD    dig+0(FP), R8
    MOVD    p_base+8(FP), R16
    MOVD    p_len+16(FP), R10

    LSR     $6, R10, R10
    LSL     $6, R10, R10
    ADD     R16, R10, R21

    // Load h0-h4 into R1–R5.
    MOVW    (R8), R1                   // R1 = h0
    MOVW    4(R8), R2                  // R2 = h1
    MOVW    8(R8), R3                  // R3 = h2
    MOVW    12(R8), R4                 // R4 = h3
    MOVW    16(R8), R5                 // R5 = h4

loop:
    // len(p) >= chunk
    CMP     R16, R21
    BLS     end

	// Initialize registers a, b, c, d, e.
	MOVW R1, R10
	MOVW R2, R11
	MOVW R3, R12
	MOVW R4, R13
	MOVW R5, R14

	// ROUND1 (steps 0-15)
	LOADCS(R10, R11, R12, R13, R14, 0)
	ROUND1(R10, R11, R12, R13, R14, 0)
	ROUND1(R14, R10, R11, R12, R13, 1)
	ROUND1(R13, R14, R10, R11, R12, 2)
	ROUND1(R12, R13, R14, R10, R11, 3)
	ROUND1(R11, R12, R13, R14, R10, 4)
	ROUND1(R10, R11, R12, R13, R14, 5)
	ROUND1(R14, R10, R11, R12, R13, 6)
	ROUND1(R13, R14, R10, R11, R12, 7)
	ROUND1(R12, R13, R14, R10, R11, 8)
	ROUND1(R11, R12, R13, R14, R10, 9)
	ROUND1(R10, R11, R12, R13, R14, 10)
	ROUND1(R14, R10, R11, R12, R13, 11)
	ROUND1(R13, R14, R10, R11, R12, 12)
	ROUND1(R12, R13, R14, R10, R11, 13)
	ROUND1(R11, R12, R13, R14, R10, 14)
	ROUND1(R10, R11, R12, R13, R14, 15)

	// ROUND1x (steps 16-19) - same as ROUND1 but with no data load.
	ROUND1x(R14, R10, R11, R12, R13, 16)
	ROUND1x(R13, R14, R10, R11, R12, 17)
	ROUND1x(R12, R13, R14, R10, R11, 18)
	ROUND1x(R11, R12, R13, R14, R10, 19)

	// ROUND2 (steps 20-39)
	ROUND2(R10, R11, R12, R13, R14, 20)
	ROUND2(R14, R10, R11, R12, R13, 21)
	ROUND2(R13, R14, R10, R11, R12, 22)
	ROUND2(R12, R13, R14, R10, R11, 23)
	ROUND2(R11, R12, R13, R14, R10, 24)
	ROUND2(R10, R11, R12, R13, R14, 25)
	ROUND2(R14, R10, R11, R12, R13, 26)
	ROUND2(R13, R14, R10, R11, R12, 27)
	ROUND2(R12, R13, R14, R10, R11, 28)
	ROUND2(R11, R12, R13, R14, R10, 29)
	ROUND2(R10, R11, R12, R13, R14, 30)
	ROUND2(R14, R10, R11, R12, R13, 31)
	ROUND2(R13, R14, R10, R11, R12, 32)
	ROUND2(R12, R13, R14, R10, R11, 33)
	ROUND2(R11, R12, R13, R14, R10, 34)
	ROUND2(R10, R11, R12, R13, R14, 35)
	ROUND2(R14, R10, R11, R12, R13, 36)
	ROUND2(R13, R14, R10, R11, R12, 37)
	ROUND2(R12, R13, R14, R10, R11, 38)
	ROUND2(R11, R12, R13, R14, R10, 39)

	// ROUND3 (steps 40-59)
	ROUND3(R10, R11, R12, R13, R14, 40)
	ROUND3(R14, R10, R11, R12, R13, 41)
	ROUND3(R13, R14, R10, R11, R12, 42)
	ROUND3(R12, R13, R14, R10, R11, 43)
	ROUND3(R11, R12, R13, R14, R10, 44)
	ROUND3(R10, R11, R12, R13, R14, 45)
	ROUND3(R14, R10, R11, R12, R13, 46)
	ROUND3(R13, R14, R10, R11, R12, 47)
	ROUND3(R12, R13, R14, R10, R11, 48)
	ROUND3(R11, R12, R13, R14, R10, 49)
	ROUND3(R10, R11, R12, R13, R14, 50)
	ROUND3(R14, R10, R11, R12, R13, 51)
	ROUND3(R13, R14, R10, R11, R12, 52)
	ROUND3(R12, R13, R14, R10, R11, 53)
	ROUND3(R11, R12, R13, R14, R10, 54)
	ROUND3(R10, R11, R12, R13, R14, 55)
	ROUND3(R14, R10, R11, R12, R13, 56)
	ROUND3(R13, R14, R10, R11, R12, 57)

	LOADCS(R12, R13, R14, R10, R11, 1)
	ROUND3(R12, R13, R14, R10, R11, 58)
	ROUND3(R11, R12, R13, R14, R10, 59)

	// ROUND4 (steps 60-79)
	ROUND4(R10, R11, R12, R13, R14, 60)
	ROUND4(R14, R10, R11, R12, R13, 61)
	ROUND4(R13, R14, R10, R11, R12, 62)
	ROUND4(R12, R13, R14, R10, R11, 63)
	ROUND4(R11, R12, R13, R14, R10, 64)

	LOADCS(R10, R11, R12, R13, R14, 2)
	ROUND4(R10, R11, R12, R13, R14, 65)
	ROUND4(R14, R10, R11, R12, R13, 66)
	ROUND4(R13, R14, R10, R11, R12, 67)
	ROUND4(R12, R13, R14, R10, R11, 68)
	ROUND4(R11, R12, R13, R14, R10, 69)
	ROUND4(R10, R11, R12, R13, R14, 70)
	ROUND4(R14, R10, R11, R12, R13, 71)
	ROUND4(R13, R14, R10, R11, R12, 72)
	ROUND4(R12, R13, R14, R10, R11, 73)
	ROUND4(R11, R12, R13, R14, R10, 74)
	ROUND4(R10, R11, R12, R13, R14, 75)
	ROUND4(R14, R10, R11, R12, R13, 76)
	ROUND4(R13, R14, R10, R11, R12, 77)
	ROUND4(R12, R13, R14, R10, R11, 78)
	ROUND4(R11, R12, R13, R14, R10, 79)

	// Add registers to temp hash.
	ADDW R10, R1, R1
	ADDW R11, R2, R2
	ADDW R12, R3, R3
	ADDW R13, R4, R4
	ADDW R14, R5, R5

	ADD  $64, R16, R16
	B  loop

end:
	MOVW R1, (R8)
	MOVW R2, 4(R8)
	MOVW R3, 8(R8)
	MOVW R4, 12(R8)
	MOVW R5, 16(R8)
	RET
