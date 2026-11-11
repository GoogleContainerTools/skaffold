//go:build !noasm && gc && amd64

#include "textflag.h"

// The fallback for CPUs without SHA-NI. The message schedule is expanded with
// AVX2 for two blocks at a time, one per 128-bit lane, as it does not depend
// on the chaining state. The rounds then run one block at a time with scalar
// BMI instructions, so that the collision detection can run between blocks
// and gets the exact states before steps 58 and 65, with no repair needed.

// func scheduleAVX2(pa, pb *byte, m1a, m1b *[80]uint32)
// Requires: AVX, AVX2
//
// Expands the blocks at pa and pb, which may be the same, into m1a and m1b.
// Y0-Y7 hold the last eight groups of four words, group g in Y(g%8).
TEXT ·scheduleAVX2(SB), NOSPLIT, $0-32
	MOVQ pa+0(FP), AX
	MOVQ pb+8(FP), BX
	MOVQ m1a+16(FP), CX
	MOVQ m1b+24(FP), DX
	VBROADCASTI128 bswap_mask<>(SB), Y13

	// W[0..3] loaded from the blocks
	VMOVDQU 0(AX), X0
	VINSERTI128 $1, 0(BX), Y0, Y0
	VPSHUFB Y13, Y0, Y0
	VMOVDQU X0, 0(CX)
	VEXTRACTI128 $1, Y0, 0(DX)

	// W[4..7] loaded from the blocks
	VMOVDQU 16(AX), X1
	VINSERTI128 $1, 16(BX), Y1, Y1
	VPSHUFB Y13, Y1, Y1
	VMOVDQU X1, 16(CX)
	VEXTRACTI128 $1, Y1, 16(DX)

	// W[8..11] loaded from the blocks
	VMOVDQU 32(AX), X2
	VINSERTI128 $1, 32(BX), Y2, Y2
	VPSHUFB Y13, Y2, Y2
	VMOVDQU X2, 32(CX)
	VEXTRACTI128 $1, Y2, 32(DX)

	// W[12..15] loaded from the blocks
	VMOVDQU 48(AX), X3
	VINSERTI128 $1, 48(BX), Y3, Y3
	VPSHUFB Y13, Y3, Y3
	VMOVDQU X3, 48(CX)
	VEXTRACTI128 $1, Y3, 48(DX)

	// W[16..19] = rol1(W[i-3] ^ W[i-8] ^ W[i-14] ^ W[i-16]). W[i+3] needs
	// W[i] from this group, which is added in afterwards.
	VPALIGNR $8, Y0, Y1, Y8 // W[i-14..i-11]
	VPXOR Y0, Y8, Y8 // W[i-16..i-13]
	VPXOR Y2, Y8, Y8 // W[i-8..i-5]
	VPSRLDQ $4, Y3, Y9 // W[i-3..i-1], 0
	VPXOR Y9, Y8, Y8
	VPSRLD $31, Y8, Y9
	VPSLLD $1, Y8, Y4
	VPOR Y9, Y4, Y4
	VPSLLDQ $12, Y8, Y10 // 0, 0, 0, the input to W[i]
	VPSRLD $30, Y10, Y9
	VPSLLD $2, Y10, Y10
	VPOR Y9, Y10, Y10 // rol1(W[i]) in the lane of W[i+3]
	VPXOR Y10, Y4, Y4
	VMOVDQU X4, 64(CX)
	VEXTRACTI128 $1, Y4, 64(DX)

	// W[20..23] = rol1(W[i-3] ^ W[i-8] ^ W[i-14] ^ W[i-16]). W[i+3] needs
	// W[i] from this group, which is added in afterwards.
	VPALIGNR $8, Y1, Y2, Y8 // W[i-14..i-11]
	VPXOR Y1, Y8, Y8 // W[i-16..i-13]
	VPXOR Y3, Y8, Y8 // W[i-8..i-5]
	VPSRLDQ $4, Y4, Y9 // W[i-3..i-1], 0
	VPXOR Y9, Y8, Y8
	VPSRLD $31, Y8, Y9
	VPSLLD $1, Y8, Y5
	VPOR Y9, Y5, Y5
	VPSLLDQ $12, Y8, Y10 // 0, 0, 0, the input to W[i]
	VPSRLD $30, Y10, Y9
	VPSLLD $2, Y10, Y10
	VPOR Y9, Y10, Y10 // rol1(W[i]) in the lane of W[i+3]
	VPXOR Y10, Y5, Y5
	VMOVDQU X5, 80(CX)
	VEXTRACTI128 $1, Y5, 80(DX)

	// W[24..27] = rol1(W[i-3] ^ W[i-8] ^ W[i-14] ^ W[i-16]). W[i+3] needs
	// W[i] from this group, which is added in afterwards.
	VPALIGNR $8, Y2, Y3, Y8 // W[i-14..i-11]
	VPXOR Y2, Y8, Y8 // W[i-16..i-13]
	VPXOR Y4, Y8, Y8 // W[i-8..i-5]
	VPSRLDQ $4, Y5, Y9 // W[i-3..i-1], 0
	VPXOR Y9, Y8, Y8
	VPSRLD $31, Y8, Y9
	VPSLLD $1, Y8, Y6
	VPOR Y9, Y6, Y6
	VPSLLDQ $12, Y8, Y10 // 0, 0, 0, the input to W[i]
	VPSRLD $30, Y10, Y9
	VPSLLD $2, Y10, Y10
	VPOR Y9, Y10, Y10 // rol1(W[i]) in the lane of W[i+3]
	VPXOR Y10, Y6, Y6
	VMOVDQU X6, 96(CX)
	VEXTRACTI128 $1, Y6, 96(DX)

	// W[28..31] = rol1(W[i-3] ^ W[i-8] ^ W[i-14] ^ W[i-16]). W[i+3] needs
	// W[i] from this group, which is added in afterwards.
	VPALIGNR $8, Y3, Y4, Y8 // W[i-14..i-11]
	VPXOR Y3, Y8, Y8 // W[i-16..i-13]
	VPXOR Y5, Y8, Y8 // W[i-8..i-5]
	VPSRLDQ $4, Y6, Y9 // W[i-3..i-1], 0
	VPXOR Y9, Y8, Y8
	VPSRLD $31, Y8, Y9
	VPSLLD $1, Y8, Y7
	VPOR Y9, Y7, Y7
	VPSLLDQ $12, Y8, Y10 // 0, 0, 0, the input to W[i]
	VPSRLD $30, Y10, Y9
	VPSLLD $2, Y10, Y10
	VPOR Y9, Y10, Y10 // rol1(W[i]) in the lane of W[i+3]
	VPXOR Y10, Y7, Y7
	VMOVDQU X7, 112(CX)
	VEXTRACTI128 $1, Y7, 112(DX)

	// W[32..35] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y6, Y7, Y8 // W[i-6..i-3]
	VPXOR Y4, Y8, Y8 // W[i-16..i-13]
	VPXOR Y1, Y8, Y8 // W[i-28..i-25]
	VPXOR Y0, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y0
	VPOR Y9, Y0, Y0
	VMOVDQU X0, 128(CX)
	VEXTRACTI128 $1, Y0, 128(DX)

	// W[36..39] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y7, Y0, Y8 // W[i-6..i-3]
	VPXOR Y5, Y8, Y8 // W[i-16..i-13]
	VPXOR Y2, Y8, Y8 // W[i-28..i-25]
	VPXOR Y1, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y1
	VPOR Y9, Y1, Y1
	VMOVDQU X1, 144(CX)
	VEXTRACTI128 $1, Y1, 144(DX)

	// W[40..43] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y0, Y1, Y8 // W[i-6..i-3]
	VPXOR Y6, Y8, Y8 // W[i-16..i-13]
	VPXOR Y3, Y8, Y8 // W[i-28..i-25]
	VPXOR Y2, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y2
	VPOR Y9, Y2, Y2
	VMOVDQU X2, 160(CX)
	VEXTRACTI128 $1, Y2, 160(DX)

	// W[44..47] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y1, Y2, Y8 // W[i-6..i-3]
	VPXOR Y7, Y8, Y8 // W[i-16..i-13]
	VPXOR Y4, Y8, Y8 // W[i-28..i-25]
	VPXOR Y3, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y3
	VPOR Y9, Y3, Y3
	VMOVDQU X3, 176(CX)
	VEXTRACTI128 $1, Y3, 176(DX)

	// W[48..51] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y2, Y3, Y8 // W[i-6..i-3]
	VPXOR Y0, Y8, Y8 // W[i-16..i-13]
	VPXOR Y5, Y8, Y8 // W[i-28..i-25]
	VPXOR Y4, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y4
	VPOR Y9, Y4, Y4
	VMOVDQU X4, 192(CX)
	VEXTRACTI128 $1, Y4, 192(DX)

	// W[52..55] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y3, Y4, Y8 // W[i-6..i-3]
	VPXOR Y1, Y8, Y8 // W[i-16..i-13]
	VPXOR Y6, Y8, Y8 // W[i-28..i-25]
	VPXOR Y5, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y5
	VPOR Y9, Y5, Y5
	VMOVDQU X5, 208(CX)
	VEXTRACTI128 $1, Y5, 208(DX)

	// W[56..59] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y4, Y5, Y8 // W[i-6..i-3]
	VPXOR Y2, Y8, Y8 // W[i-16..i-13]
	VPXOR Y7, Y8, Y8 // W[i-28..i-25]
	VPXOR Y6, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y6
	VPOR Y9, Y6, Y6
	VMOVDQU X6, 224(CX)
	VEXTRACTI128 $1, Y6, 224(DX)

	// W[60..63] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y5, Y6, Y8 // W[i-6..i-3]
	VPXOR Y3, Y8, Y8 // W[i-16..i-13]
	VPXOR Y0, Y8, Y8 // W[i-28..i-25]
	VPXOR Y7, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y7
	VPOR Y9, Y7, Y7
	VMOVDQU X7, 240(CX)
	VEXTRACTI128 $1, Y7, 240(DX)

	// W[64..67] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y6, Y7, Y8 // W[i-6..i-3]
	VPXOR Y4, Y8, Y8 // W[i-16..i-13]
	VPXOR Y1, Y8, Y8 // W[i-28..i-25]
	VPXOR Y0, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y0
	VPOR Y9, Y0, Y0
	VMOVDQU X0, 256(CX)
	VEXTRACTI128 $1, Y0, 256(DX)

	// W[68..71] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y7, Y0, Y8 // W[i-6..i-3]
	VPXOR Y5, Y8, Y8 // W[i-16..i-13]
	VPXOR Y2, Y8, Y8 // W[i-28..i-25]
	VPXOR Y1, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y1
	VPOR Y9, Y1, Y1
	VMOVDQU X1, 272(CX)
	VEXTRACTI128 $1, Y1, 272(DX)

	// W[72..75] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y0, Y1, Y8 // W[i-6..i-3]
	VPXOR Y6, Y8, Y8 // W[i-16..i-13]
	VPXOR Y3, Y8, Y8 // W[i-28..i-25]
	VPXOR Y2, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y2
	VPOR Y9, Y2, Y2
	VMOVDQU X2, 288(CX)
	VEXTRACTI128 $1, Y2, 288(DX)

	// W[76..79] = rol2(W[i-6] ^ W[i-16] ^ W[i-28] ^ W[i-32])
	VPALIGNR $8, Y1, Y2, Y8 // W[i-6..i-3]
	VPXOR Y7, Y8, Y8 // W[i-16..i-13]
	VPXOR Y4, Y8, Y8 // W[i-28..i-25]
	VPXOR Y3, Y8, Y8 // W[i-32..i-29]
	VPSRLD $30, Y8, Y9
	VPSLLD $2, Y8, Y3
	VPOR Y9, Y3, Y3
	VMOVDQU X3, 304(CX)
	VEXTRACTI128 $1, Y3, 304(DX)

	VZEROUPPER
	RET

// Each round adds W[i], K, f(b, c, d) and rol5(a) into e and rotates b by 30.
// The callers rotate the register names instead of moving values around, and
// rol5(a) goes in last, as a is the only input the previous round produced.
#define ROUND_CH(a, b, c, d, e, i) \
	ADDL ((i)*4)(SI), e; \
	ADDL $0x5a827999, e; \
	ANDNL d, b, AX; \
	MOVL c, BX; \
	ANDL b, BX; \
	ADDL AX, e; \
	ADDL BX, e; \
	RORXL $27, a, CX; \
	RORXL $2, b, b; \
	ADDL CX, e

#define ROUND_PARITY(a, b, c, d, e, i, k) \
	ADDL ((i)*4)(SI), e; \
	ADDL k, e; \
	MOVL b, AX; \
	XORL c, AX; \
	XORL d, AX; \
	ADDL AX, e; \
	RORXL $27, a, CX; \
	RORXL $2, b, b; \
	ADDL CX, e

// maj(b, c, d) = (b & c) + ((b ^ c) & d), as the two never share a bit.
#define ROUND_MAJ(a, b, c, d, e, i) \
	ADDL ((i)*4)(SI), e; \
	ADDL $0x8f1bbcdc, e; \
	MOVL b, AX; \
	XORL c, AX; \
	ANDL d, AX; \
	MOVL b, BX; \
	ANDL c, BX; \
	ADDL AX, e; \
	ADDL BX, e; \
	RORXL $27, a, CX; \
	RORXL $2, b, b; \
	ADDL CX, e

#define SAVECS(a, b, c, d, e, index) \
	MOVL a, ((index)*20+0)(DX); \
	MOVL b, ((index)*20+4)(DX); \
	MOVL c, ((index)*20+8)(DX); \
	MOVL d, ((index)*20+12)(DX); \
	MOVL e, ((index)*20+16)(DX)

// func roundsBMI2(h *[5]uint32, m1 *[80]uint32, cs *[3][5]uint32)
// Requires: BMI1, BMI2
//
// Compresses the block whose schedule is in m1 into h, storing the states
// before steps 0, 58 and 65 into cs.
TEXT ·roundsBMI2(SB), NOSPLIT, $0-24
	MOVQ h+0(FP), DI
	MOVQ m1+8(FP), SI
	MOVQ cs+16(FP), DX
	MOVL 0(DI), R8
	MOVL 4(DI), R9
	MOVL 8(DI), R10
	MOVL 12(DI), R11
	MOVL 16(DI), R12

	SAVECS(R8, R9, R10, R11, R12, 0)
	ROUND_CH(R8, R9, R10, R11, R12, 0)
	ROUND_CH(R12, R8, R9, R10, R11, 1)
	ROUND_CH(R11, R12, R8, R9, R10, 2)
	ROUND_CH(R10, R11, R12, R8, R9, 3)
	ROUND_CH(R9, R10, R11, R12, R8, 4)
	ROUND_CH(R8, R9, R10, R11, R12, 5)
	ROUND_CH(R12, R8, R9, R10, R11, 6)
	ROUND_CH(R11, R12, R8, R9, R10, 7)
	ROUND_CH(R10, R11, R12, R8, R9, 8)
	ROUND_CH(R9, R10, R11, R12, R8, 9)
	ROUND_CH(R8, R9, R10, R11, R12, 10)
	ROUND_CH(R12, R8, R9, R10, R11, 11)
	ROUND_CH(R11, R12, R8, R9, R10, 12)
	ROUND_CH(R10, R11, R12, R8, R9, 13)
	ROUND_CH(R9, R10, R11, R12, R8, 14)
	ROUND_CH(R8, R9, R10, R11, R12, 15)
	ROUND_CH(R12, R8, R9, R10, R11, 16)
	ROUND_CH(R11, R12, R8, R9, R10, 17)
	ROUND_CH(R10, R11, R12, R8, R9, 18)
	ROUND_CH(R9, R10, R11, R12, R8, 19)
	ROUND_PARITY(R8, R9, R10, R11, R12, 20, $0x6ed9eba1)
	ROUND_PARITY(R12, R8, R9, R10, R11, 21, $0x6ed9eba1)
	ROUND_PARITY(R11, R12, R8, R9, R10, 22, $0x6ed9eba1)
	ROUND_PARITY(R10, R11, R12, R8, R9, 23, $0x6ed9eba1)
	ROUND_PARITY(R9, R10, R11, R12, R8, 24, $0x6ed9eba1)
	ROUND_PARITY(R8, R9, R10, R11, R12, 25, $0x6ed9eba1)
	ROUND_PARITY(R12, R8, R9, R10, R11, 26, $0x6ed9eba1)
	ROUND_PARITY(R11, R12, R8, R9, R10, 27, $0x6ed9eba1)
	ROUND_PARITY(R10, R11, R12, R8, R9, 28, $0x6ed9eba1)
	ROUND_PARITY(R9, R10, R11, R12, R8, 29, $0x6ed9eba1)
	ROUND_PARITY(R8, R9, R10, R11, R12, 30, $0x6ed9eba1)
	ROUND_PARITY(R12, R8, R9, R10, R11, 31, $0x6ed9eba1)
	ROUND_PARITY(R11, R12, R8, R9, R10, 32, $0x6ed9eba1)
	ROUND_PARITY(R10, R11, R12, R8, R9, 33, $0x6ed9eba1)
	ROUND_PARITY(R9, R10, R11, R12, R8, 34, $0x6ed9eba1)
	ROUND_PARITY(R8, R9, R10, R11, R12, 35, $0x6ed9eba1)
	ROUND_PARITY(R12, R8, R9, R10, R11, 36, $0x6ed9eba1)
	ROUND_PARITY(R11, R12, R8, R9, R10, 37, $0x6ed9eba1)
	ROUND_PARITY(R10, R11, R12, R8, R9, 38, $0x6ed9eba1)
	ROUND_PARITY(R9, R10, R11, R12, R8, 39, $0x6ed9eba1)
	ROUND_MAJ(R8, R9, R10, R11, R12, 40)
	ROUND_MAJ(R12, R8, R9, R10, R11, 41)
	ROUND_MAJ(R11, R12, R8, R9, R10, 42)
	ROUND_MAJ(R10, R11, R12, R8, R9, 43)
	ROUND_MAJ(R9, R10, R11, R12, R8, 44)
	ROUND_MAJ(R8, R9, R10, R11, R12, 45)
	ROUND_MAJ(R12, R8, R9, R10, R11, 46)
	ROUND_MAJ(R11, R12, R8, R9, R10, 47)
	ROUND_MAJ(R10, R11, R12, R8, R9, 48)
	ROUND_MAJ(R9, R10, R11, R12, R8, 49)
	ROUND_MAJ(R8, R9, R10, R11, R12, 50)
	ROUND_MAJ(R12, R8, R9, R10, R11, 51)
	ROUND_MAJ(R11, R12, R8, R9, R10, 52)
	ROUND_MAJ(R10, R11, R12, R8, R9, 53)
	ROUND_MAJ(R9, R10, R11, R12, R8, 54)
	ROUND_MAJ(R8, R9, R10, R11, R12, 55)
	ROUND_MAJ(R12, R8, R9, R10, R11, 56)
	ROUND_MAJ(R11, R12, R8, R9, R10, 57)
	SAVECS(R10, R11, R12, R8, R9, 1)
	ROUND_MAJ(R10, R11, R12, R8, R9, 58)
	ROUND_MAJ(R9, R10, R11, R12, R8, 59)
	ROUND_PARITY(R8, R9, R10, R11, R12, 60, $0xca62c1d6)
	ROUND_PARITY(R12, R8, R9, R10, R11, 61, $0xca62c1d6)
	ROUND_PARITY(R11, R12, R8, R9, R10, 62, $0xca62c1d6)
	ROUND_PARITY(R10, R11, R12, R8, R9, 63, $0xca62c1d6)
	ROUND_PARITY(R9, R10, R11, R12, R8, 64, $0xca62c1d6)
	SAVECS(R8, R9, R10, R11, R12, 2)
	ROUND_PARITY(R8, R9, R10, R11, R12, 65, $0xca62c1d6)
	ROUND_PARITY(R12, R8, R9, R10, R11, 66, $0xca62c1d6)
	ROUND_PARITY(R11, R12, R8, R9, R10, 67, $0xca62c1d6)
	ROUND_PARITY(R10, R11, R12, R8, R9, 68, $0xca62c1d6)
	ROUND_PARITY(R9, R10, R11, R12, R8, 69, $0xca62c1d6)
	ROUND_PARITY(R8, R9, R10, R11, R12, 70, $0xca62c1d6)
	ROUND_PARITY(R12, R8, R9, R10, R11, 71, $0xca62c1d6)
	ROUND_PARITY(R11, R12, R8, R9, R10, 72, $0xca62c1d6)
	ROUND_PARITY(R10, R11, R12, R8, R9, 73, $0xca62c1d6)
	ROUND_PARITY(R9, R10, R11, R12, R8, 74, $0xca62c1d6)
	ROUND_PARITY(R8, R9, R10, R11, R12, 75, $0xca62c1d6)
	ROUND_PARITY(R12, R8, R9, R10, R11, 76, $0xca62c1d6)
	ROUND_PARITY(R11, R12, R8, R9, R10, 77, $0xca62c1d6)
	ROUND_PARITY(R10, R11, R12, R8, R9, 78, $0xca62c1d6)
	ROUND_PARITY(R9, R10, R11, R12, R8, 79, $0xca62c1d6)

	ADDL R8, 0(DI)
	ADDL R9, 4(DI)
	ADDL R10, 8(DI)
	ADDL R11, 12(DI)
	ADDL R12, 16(DI)
	RET

// Swaps the bytes of each word, as SHA-1 reads the block big endian.
DATA bswap_mask<>+0(SB)/8, $0x0405060700010203
DATA bswap_mask<>+8(SB)/8, $0x0c0d0e0f08090a0b
GLOBL bswap_mask<>(SB), RODATA|NOPTR, $16
