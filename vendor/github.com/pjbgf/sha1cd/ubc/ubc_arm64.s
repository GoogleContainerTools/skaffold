//go:build !noasm && gc && arm64 && !amd64
// +build !noasm,gc,arm64,!amd64

#include "textflag.h"

// func CalculateDvMaskARM64(W [80]uint32) uint32
TEXT ·CalculateDvMaskARM64(SB), NOSPLIT, $0-324
	MOVW $0xffffffff, R0

	// (((((W[44] ^ W[45]) >> 29) & 1) - 1) | ^(DV_I_48_0_bit | DV_I_51_0_bit | DV_I_52_0_bit | DV_II_45_0_bit | DV_II_46_0_bit | DV_II_50_0_bit | DV_II_51_0_bit))
	MOVW W_44+176(FP), R1
	MOVW W_45+180(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xfd7c5f7f, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[49] ^ W[50]) >> 29) & 1) - 1) | ^(DV_I_46_0_bit | DV_II_45_0_bit | DV_II_50_0_bit | DV_II_51_0_bit | DV_II_55_0_bit | DV_II_56_0_bit))
	MOVW W_49+196(FP), R1
	MOVW W_50+200(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0x3d7efff7, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[48] ^ W[49]) >> 29) & 1) - 1) | ^(DV_I_45_0_bit | DV_I_52_0_bit | DV_II_49_0_bit | DV_II_50_0_bit | DV_II_54_0_bit | DV_II_55_0_bit))
	MOVW W_48+192(FP), R1
	MOVW W_49+196(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0x9f5f7ffb, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[47] ^ (W[50] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_I_47_0_bit | DV_I_49_0_bit | DV_I_51_0_bit | DV_II_45_0_bit | DV_II_51_0_bit | DV_II_56_0_bit))
	MOVW W_47+188(FP), R1
	MOVW W_50+200(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0x7dfedddf, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[47] ^ W[48]) >> 29) & 1) - 1) | ^(DV_I_44_0_bit | DV_I_51_0_bit | DV_II_48_0_bit | DV_II_49_0_bit | DV_II_53_0_bit | DV_II_54_0_bit))
	MOVW W_47+188(FP), R1
	MOVW W_48+192(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xcfcfdffd, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[46] >> 4) ^ (W[49] >> 29)) & 1) - 1) | ^(DV_I_46_0_bit | DV_I_48_0_bit | DV_I_50_0_bit | DV_I_52_0_bit | DV_II_50_0_bit | DV_II_55_0_bit))
	MOVW W_46+184(FP), R1
	LSR  $0x04, R1, R1
	MOVW W_49+196(FP), R2
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xbf7f7777, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[46] ^ W[47]) >> 29) & 1) - 1) | ^(DV_I_43_0_bit | DV_I_50_0_bit | DV_II_47_0_bit | DV_II_48_0_bit | DV_II_52_0_bit | DV_II_53_0_bit))
	MOVW W_46+184(FP), R1
	MOVW W_47+188(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xe7e7f7fe, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[45] >> 4) ^ (W[48] >> 29)) & 1) - 1) | ^(DV_I_45_0_bit | DV_I_47_0_bit | DV_I_49_0_bit | DV_I_51_0_bit | DV_II_49_0_bit | DV_II_54_0_bit))
	MOVW W_45+180(FP), R1
	LSR  $0x04, R1, R1
	MOVW W_48+192(FP), R2
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xdfdfdddb, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[45] ^ W[46]) >> 29) & 1) - 1) | ^(DV_I_49_0_bit | DV_I_52_0_bit | DV_II_46_0_bit | DV_II_47_0_bit | DV_II_51_0_bit | DV_II_52_0_bit))
	MOVW W_45+180(FP), R1
	MOVW W_46+184(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xf5f57dff, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[44] >> 4) ^ (W[47] >> 29)) & 1) - 1) | ^(DV_I_44_0_bit | DV_I_46_0_bit | DV_I_48_0_bit | DV_I_50_0_bit | DV_II_48_0_bit | DV_II_53_0_bit))
	MOVW W_44+176(FP), R1
	LSR  $0x04, R1, R1
	MOVW W_47+188(FP), R2
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xefeff775, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[43] >> 4) ^ (W[46] >> 29)) & 1) - 1) | ^(DV_I_43_0_bit | DV_I_45_0_bit | DV_I_47_0_bit | DV_I_49_0_bit | DV_II_47_0_bit | DV_II_52_0_bit))
	MOVW W_43+172(FP), R1
	LSR  $0x04, R1, R1
	MOVW W_46+184(FP), R2
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xf7f7fdda, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[43] ^ W[44]) >> 29) & 1) - 1) | ^(DV_I_47_0_bit | DV_I_50_0_bit | DV_I_51_0_bit | DV_II_45_0_bit | DV_II_49_0_bit | DV_II_50_0_bit))
	MOVW W_43+172(FP), R1
	MOVW W_44+176(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xff5ed7df, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[42] >> 4) ^ (W[45] >> 29)) & 1) - 1) | ^(DV_I_44_0_bit | DV_I_46_0_bit | DV_I_48_0_bit | DV_I_52_0_bit | DV_II_46_0_bit | DV_II_51_0_bit))
	MOVW W_42+168(FP), R1
	LSR  $0x04, R1, R1
	MOVW W_45+180(FP), R2
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xfdfd7f75, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[41] >> 4) ^ (W[44] >> 29)) & 1) - 1) | ^(DV_I_43_0_bit | DV_I_45_0_bit | DV_I_47_0_bit | DV_I_51_0_bit | DV_II_45_0_bit | DV_II_50_0_bit))
	MOVW W_41+164(FP), R1
	LSR  $0x04, R1, R1
	MOVW W_44+176(FP), R2
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xff7edfda, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[40] ^ W[41]) >> 29) & 1) - 1) | ^(DV_I_44_0_bit | DV_I_47_0_bit | DV_I_48_0_bit | DV_II_46_0_bit | DV_II_47_0_bit | DV_II_56_0_bit))
	MOVW W_40+160(FP), R1
	MOVW W_41+164(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0x7ff5ff5d, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[54] ^ W[55]) >> 29) & 1) - 1) | ^(DV_I_51_0_bit | DV_II_47_0_bit | DV_II_50_0_bit | DV_II_55_0_bit | DV_II_56_0_bit))
	MOVW W_54+216(FP), R1
	MOVW W_55+220(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0x3f77dfff, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[53] ^ W[54]) >> 29) & 1) - 1) | ^(DV_I_50_0_bit | DV_II_46_0_bit | DV_II_49_0_bit | DV_II_54_0_bit | DV_II_55_0_bit))
	MOVW W_53+212(FP), R1
	MOVW W_54+216(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0x9fddf7ff, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[52] ^ W[53]) >> 29) & 1) - 1) | ^(DV_I_49_0_bit | DV_II_45_0_bit | DV_II_48_0_bit | DV_II_53_0_bit | DV_II_54_0_bit))
	MOVW W_52+208(FP), R1
	MOVW W_53+212(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xcfeefdff, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[50] ^ (W[53] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_I_50_0_bit | DV_I_52_0_bit | DV_II_46_0_bit | DV_II_48_0_bit | DV_II_54_0_bit))
	MOVW W_50+200(FP), R1
	MOVW W_53+212(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xdfed77ff, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[50] ^ W[51]) >> 29) & 1) - 1) | ^(DV_I_47_0_bit | DV_II_46_0_bit | DV_II_51_0_bit | DV_II_52_0_bit | DV_II_56_0_bit))
	MOVW W_50+200(FP), R1
	MOVW W_51+204(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0x75fdffdf, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[49] ^ (W[52] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_I_49_0_bit | DV_I_51_0_bit | DV_II_45_0_bit | DV_II_47_0_bit | DV_II_53_0_bit))
	MOVW W_49+196(FP), R1
	MOVW W_52+208(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xeff6ddff, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[48] ^ (W[51] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_I_48_0_bit | DV_I_50_0_bit | DV_I_52_0_bit | DV_II_46_0_bit | DV_II_52_0_bit))
	MOVW W_48+192(FP), R1
	MOVW W_51+204(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xf7fd777f, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[42] ^ W[43]) >> 29) & 1) - 1) | ^(DV_I_46_0_bit | DV_I_49_0_bit | DV_I_50_0_bit | DV_II_48_0_bit | DV_II_49_0_bit))
	MOVW W_42+168(FP), R1
	MOVW W_43+172(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xffcff5f7, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[41] ^ W[42]) >> 29) & 1) - 1) | ^(DV_I_45_0_bit | DV_I_48_0_bit | DV_I_49_0_bit | DV_II_47_0_bit | DV_II_48_0_bit))
	MOVW W_41+164(FP), R1
	MOVW W_42+168(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xffe7fd7b, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[40] >> 4) ^ (W[43] >> 29)) & 1) - 1) | ^(DV_I_44_0_bit | DV_I_46_0_bit | DV_I_50_0_bit | DV_II_49_0_bit | DV_II_56_0_bit))
	MOVW W_40+160(FP), R1
	LSR  $0x04, R1, R1
	MOVW W_43+172(FP), R2
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0x7fdff7f5, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[39] >> 4) ^ (W[42] >> 29)) & 1) - 1) | ^(DV_I_43_0_bit | DV_I_45_0_bit | DV_I_49_0_bit | DV_II_48_0_bit | DV_II_55_0_bit))
	MOVW W_39+156(FP), R1
	LSR  $0x04, R1, R1
	MOVW W_42+168(FP), R2
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xbfeffdfa, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_44_0_bit | DV_I_48_0_bit | DV_II_47_0_bit | DV_II_54_0_bit | DV_II_56_0_bit)) != 0 {
	//   mask &= (((((W[38] >> 4) ^ (W[41] >> 29)) & 1) - 1) | ^(DV_I_44_0_bit | DV_I_48_0_bit | DV_II_47_0_bit | DV_II_54_0_bit | DV_II_56_0_bit))
	// }
	TST  $0xa0080082, R0
	BEQ  f1
	MOVW W_38+152(FP), R1
	MOVW W_41+164(FP), R2
	LSR  $0x04, R1, R1
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0x5ff7ff7d, R1, R1
	AND  R1, R0, R0

f1:
	// mask &= (((((W[37] >> 4) ^ (W[40] >> 29)) & 1) - 1) | ^(DV_I_43_0_bit | DV_I_47_0_bit | DV_II_46_0_bit | DV_II_53_0_bit | DV_II_55_0_bit))
	MOVW W_37+148(FP), R1
	MOVW W_40+160(FP), R2
	LSR  $0x04, R1, R1
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xaffdffde, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_52_0_bit | DV_II_48_0_bit | DV_II_51_0_bit | DV_II_56_0_bit)) != 0 {
	//   mask &= (((((W[55] ^ W[56]) >> 29) & 1) - 1) | ^(DV_I_52_0_bit | DV_II_48_0_bit | DV_II_51_0_bit | DV_II_56_0_bit))
	// }
	TST  $0x82108000, R0
	BEQ  f2
	MOVW W_55+220(FP), R1
	MOVW W_56+224(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0x7def7fff, R1, R1
	AND  R1, R0, R0

f2:
	// if (mask & (DV_I_52_0_bit | DV_II_48_0_bit | DV_II_50_0_bit | DV_II_56_0_bit)) != 0 {
	//   mask &= ((((W[52] ^ (W[55] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_I_52_0_bit | DV_II_48_0_bit | DV_II_50_0_bit | DV_II_56_0_bit))
	// }
	TST  $0x80908000, R0
	BEQ  f3
	MOVW W_52+208(FP), R1
	MOVW W_55+220(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0x7f6f7fff, R1, R1
	AND  R1, R0, R0

f3:
	// if (mask & (DV_I_51_0_bit | DV_II_47_0_bit | DV_II_49_0_bit | DV_II_55_0_bit)) != 0 {
	//   mask &= ((((W[51] ^ (W[54] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_I_51_0_bit | DV_II_47_0_bit | DV_II_49_0_bit | DV_II_55_0_bit))
	// }
	TST  $0x40282000, R0
	BEQ  f4
	MOVW W_51+204(FP), R1
	MOVW W_54+216(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xbfd7dfff, R1, R1
	AND  R1, R0, R0

f4:
	// if (mask & (DV_I_48_0_bit | DV_II_47_0_bit | DV_II_52_0_bit | DV_II_53_0_bit)) != 0 {
	//   mask &= (((((W[51] ^ W[52]) >> 29) & 1) - 1) | ^(DV_I_48_0_bit | DV_II_47_0_bit | DV_II_52_0_bit | DV_II_53_0_bit))
	// }
	TST  $0x18080080, R0
	BEQ  f5
	MOVW W_51+204(FP), R1
	MOVW W_52+208(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xe7f7ff7f, R1, R1
	AND  R1, R0, R0

f5:
	// if (mask & (DV_I_46_0_bit | DV_I_49_0_bit | DV_II_45_0_bit | DV_II_48_0_bit)) != 0 {
	//   mask &= (((((W[36] >> 4) ^ (W[40] >> 29)) & 1) - 1) | ^(DV_I_46_0_bit | DV_I_49_0_bit | DV_II_45_0_bit | DV_II_48_0_bit))
	// }
	TST  $0x00110208, R0
	BEQ  f6
	MOVW W_36+144(FP), R1
	LSR  $0x04, R1, R1
	MOVW W_40+160(FP), R2
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xffeefdf7, R1, R1
	AND  R1, R0, R0

f6:
	// if (mask & (DV_I_52_0_bit | DV_II_48_0_bit | DV_II_49_0_bit)) != 0 {
	//   mask &= ((0 - (((W[53] ^ W[56]) >> 29) & 1)) | ^(DV_I_52_0_bit | DV_II_48_0_bit | DV_II_49_0_bit))
	// }
	TST  $0x00308000, R0
	BEQ  f7
	MOVW W_53+212(FP), R1
	MOVW W_56+224(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xffcf7fff, R1, R1
	AND  R1, R0, R0

f7:
	// if (mask & (DV_I_50_0_bit | DV_II_46_0_bit | DV_II_47_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[51] ^ W[54]) >> 29) & 1)) | ^(DV_I_50_0_bit | DV_II_46_0_bit | DV_II_47_0_bit))
	// }
	TST  $0x000a0800, R0
	BEQ  f8
	MOVW W_51+204(FP), R1
	MOVW W_54+216(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xfff5f7ff, R1, R1
	AND  R1, R0, R0

f8:
	// if (mask & (DV_I_49_0_bit | DV_I_51_0_bit | DV_II_45_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[50] ^ W[52]) >> 29) & 1)) | ^(DV_I_49_0_bit | DV_I_51_0_bit | DV_II_45_0_bit))
	// }
	TST  $0x00012200, R0
	BEQ  f9
	MOVW W_50+200(FP), R1
	MOVW W_52+208(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xfffeddff, R1, R1
	AND  R1, R0, R0

f9:
	// if (mask & (DV_I_48_0_bit | DV_I_50_0_bit | DV_I_52_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[49] ^ W[51]) >> 29) & 1)) | ^(DV_I_48_0_bit | DV_I_50_0_bit | DV_I_52_0_bit))
	// }
	TST  $0x00008880, R0
	BEQ  f10
	MOVW W_49+196(FP), R1
	MOVW W_51+204(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xffff777f, R1, R1
	AND  R1, R0, R0

f10:
	// if (mask & (DV_I_47_0_bit | DV_I_49_0_bit | DV_I_51_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[48] ^ W[50]) >> 29) & 1)) | ^(DV_I_47_0_bit | DV_I_49_0_bit | DV_I_51_0_bit))
	// }
	TST  $0x00002220, R0
	BEQ  f11
	MOVW W_48+192(FP), R1
	MOVW W_50+200(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xffffdddf, R1, R1
	AND  R1, R0, R0

f11:
	// if (mask & (DV_I_46_0_bit | DV_I_48_0_bit | DV_I_50_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[47] ^ W[49]) >> 29) & 1)) | ^(DV_I_46_0_bit | DV_I_48_0_bit | DV_I_50_0_bit))
	// }
	TST  $0x00000888, R0
	BEQ  f12
	MOVW W_47+188(FP), R1
	MOVW W_49+196(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xfffff777, R1, R1
	AND  R1, R0, R0

f12:
	// if (mask & (DV_I_45_0_bit | DV_I_47_0_bit | DV_I_49_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[46] ^ W[48]) >> 29) & 1)) | ^(DV_I_45_0_bit | DV_I_47_0_bit | DV_I_49_0_bit))
	// }
	TST  $0x00000224, R0
	BEQ  f13
	MOVW W_46+184(FP), R1
	MOVW W_48+192(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xfffffddb, R1, R1
	AND  R1, R0, R0

f13:
	// mask &= ((((W[45] ^ W[47]) & (1 << 6)) - (1 << 6)) | ^(DV_I_47_2_bit | DV_I_49_2_bit | DV_I_51_2_bit))
	MOVW W_45+180(FP), R1
	MOVW W_47+188(FP), R2
	EOR  R2, R1, R1
	AND  $0x00000040, R1, R1
	SUB  $0x00000040, R1, R1
	ORR  $0xffffbbbf, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_44_0_bit | DV_I_46_0_bit | DV_I_48_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[45] ^ W[47]) >> 29) & 1)) | ^(DV_I_44_0_bit | DV_I_46_0_bit | DV_I_48_0_bit))
	// }
	TST  $0x0000008a, R0
	BEQ  f14
	MOVW W_45+180(FP), R1
	MOVW W_47+188(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xffffff75, R1, R1
	AND  R1, R0, R0

f14:
	// mask &= (((((W[44] ^ W[46]) >> 6) & 1) - 1) | ^(DV_I_46_2_bit | DV_I_48_2_bit | DV_I_50_2_bit))
	MOVW W_44+176(FP), R1
	MOVW W_46+184(FP), R2
	EOR  R2, R1, R1
	LSR  $0x06, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xffffeeef, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_43_0_bit | DV_I_45_0_bit | DV_I_47_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[44] ^ W[46]) >> 29) & 1)) | ^(DV_I_43_0_bit | DV_I_45_0_bit | DV_I_47_0_bit))
	// }
	TST  $0x00000025, R0
	BEQ  f15
	MOVW W_44+176(FP), R1
	MOVW W_46+184(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xffffffda, R1, R1
	AND  R1, R0, R0

f15:
	// mask &= ((0 - ((W[41] ^ (W[42] >> 5)) & (1 << 1))) | ^(DV_I_48_2_bit | DV_II_46_2_bit | DV_II_51_2_bit))
	MOVW W_41+164(FP), R1
	MOVW W_42+168(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	ORR  $0xfbfbfeff, R1, R1
	AND  R1, R0, R0

	// mask &= ((0 - ((W[40] ^ (W[41] >> 5)) & (1 << 1))) | ^(DV_I_47_2_bit | DV_I_51_2_bit | DV_II_50_2_bit))
	MOVW W_40+160(FP), R1
	MOVW W_41+164(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	ORR  $0xfeffbfbf, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_44_0_bit | DV_I_46_0_bit | DV_II_56_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[40] ^ W[42]) >> 4) & 1)) | ^(DV_I_44_0_bit | DV_I_46_0_bit | DV_II_56_0_bit))
	// }
	TST  $0x8000000a, R0
	BEQ  f16
	MOVW W_40+160(FP), R1
	MOVW W_42+168(FP), R2
	EOR  R2, R1, R1
	LSR  $0x04, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0x7ffffff5, R1, R1
	AND  R1, R0, R0

f16:
	// mask &= ((0 - ((W[39] ^ (W[40] >> 5)) & (1 << 1))) | ^(DV_I_46_2_bit | DV_I_50_2_bit | DV_II_49_2_bit))
	MOVW W_39+156(FP), R1
	MOVW W_40+160(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	ORR  $0xffbfefef, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_43_0_bit | DV_I_45_0_bit | DV_II_55_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[39] ^ W[41]) >> 4) & 1)) | ^(DV_I_43_0_bit | DV_I_45_0_bit | DV_II_55_0_bit))
	// }
	TST  $0x40000005, R0
	BEQ  f17
	MOVW W_39+156(FP), R1
	MOVW W_41+164(FP), R2
	EOR  R2, R1, R1
	LSR  $0x04, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xbffffffa, R1, R1
	AND  R1, R0, R0

f17:
	// if (mask & (DV_I_44_0_bit | DV_II_54_0_bit | DV_II_56_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[38] ^ W[40]) >> 4) & 1)) | ^(DV_I_44_0_bit | DV_II_54_0_bit | DV_II_56_0_bit))
	// }
	TST  $0xa0000002, R0
	BEQ  f18
	MOVW W_38+152(FP), R1
	MOVW W_40+160(FP), R2
	EOR  R2, R1, R1
	LSR  $0x04, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0x5ffffffd, R1, R1
	AND  R1, R0, R0

f18:
	// if (mask & (DV_I_43_0_bit | DV_II_53_0_bit | DV_II_55_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[37] ^ W[39]) >> 4) & 1)) | ^(DV_I_43_0_bit | DV_II_53_0_bit | DV_II_55_0_bit))
	// }
	TST  $0x50000001, R0
	BEQ  f19
	MOVW W_37+148(FP), R1
	MOVW W_39+156(FP), R2
	EOR  R2, R1, R1
	LSR  $0x04, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xaffffffe, R1, R1
	AND  R1, R0, R0

f19:
	// mask &= ((0 - ((W[36] ^ (W[37] >> 5)) & (1 << 1))) | ^(DV_I_47_2_bit | DV_I_50_2_bit | DV_II_46_2_bit))
	MOVW W_36+144(FP), R1
	MOVW W_37+148(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	ORR  $0xfffbefbf, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_45_0_bit | DV_I_48_0_bit | DV_II_47_0_bit)) != 0 {
	// 	mask &= (((((W[35] >> 4) ^ (W[39] >> 29)) & 1) - 1) | ^(DV_I_45_0_bit | DV_I_48_0_bit | DV_II_47_0_bit))
	// }
	TST  $0x00080084, R0
	BEQ  f20
	MOVW W_35+140(FP), R1
	MOVW W_39+156(FP), R2
	LSR  $0x04, R1, R1
	LSR  $0x1d, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $0x00000001, R1, R1
	ORR  $0xfff7ff7b, R1, R1
	AND  R1, R0, R0

f20:
	// if (mask & (DV_I_48_0_bit | DV_II_48_0_bit)) != 0 {
	// 	mask &= ((0 - ((W[63] ^ (W[64] >> 5)) & (1 << 0))) | ^(DV_I_48_0_bit | DV_II_48_0_bit))
	// }
	TST  $0x00100080, R0
	BEQ  f21
	MOVW W_63+252(FP), R1
	MOVW W_64+256(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xffefff7f, R1, R1
	AND  R1, R0, R0

f21:
	// if (mask & (DV_I_45_0_bit | DV_II_45_0_bit)) != 0 {
	// 	mask &= ((0 - ((W[63] ^ (W[64] >> 5)) & (1 << 1))) | ^(DV_I_45_0_bit | DV_II_45_0_bit))
	// }
	TST  $0x00010004, R0
	BEQ  f22
	MOVW W_63+252(FP), R1
	MOVW W_64+256(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	ORR  $0xfffefffb, R1, R1
	AND  R1, R0, R0

f22:
	// if (mask & (DV_I_47_0_bit | DV_II_47_0_bit)) != 0 {
	// 	mask &= ((0 - ((W[62] ^ (W[63] >> 5)) & (1 << 0))) | ^(DV_I_47_0_bit | DV_II_47_0_bit))
	// }
	TST  $0x00080020, R0
	BEQ  f23
	MOVW W_62+248(FP), R1
	MOVW W_63+252(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xfff7ffdf, R1, R1
	AND  R1, R0, R0

f23:
	// if (mask & (DV_I_46_0_bit | DV_II_46_0_bit)) != 0 {
	// 	mask &= ((0 - ((W[61] ^ (W[62] >> 5)) & (1 << 0))) | ^(DV_I_46_0_bit | DV_II_46_0_bit))
	// }
	TST  $0x00020008, R0
	BEQ  f24
	MOVW W_61+244(FP), R1
	MOVW W_62+248(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xfffdfff7, R1, R1
	AND  R1, R0, R0

f24:
	// mask &= ((0 - ((W[61] ^ (W[62] >> 5)) & (1 << 2))) | ^(DV_I_46_2_bit | DV_II_46_2_bit))
	MOVW W_61+244(FP), R1
	MOVW W_62+248(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000004, R1, R1
	NEG  R1, R1
	ORR  $0xfffbffef, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_45_0_bit | DV_II_45_0_bit)) != 0 {
	// 	mask &= ((0 - ((W[60] ^ (W[61] >> 5)) & (1 << 0))) | ^(DV_I_45_0_bit | DV_II_45_0_bit))
	// }
	TST  $0x00010004, R0
	BEQ  f25
	MOVW W_60+240(FP), R1
	MOVW W_61+244(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xfffefffb, R1, R1
	AND  R1, R0, R0

f25:
	// if (mask & (DV_II_51_0_bit | DV_II_54_0_bit)) != 0 {
	// 	mask &= (((((W[58] ^ W[59]) >> 29) & 1) - 1) | ^(DV_II_51_0_bit | DV_II_54_0_bit))
	// }
	TST  $0x22000000, R0
	BEQ  f26
	MOVW W_58+232(FP), R1
	MOVW W_59+236(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xddffffff, R1, R1
	AND  R1, R0, R0

f26:
	// if (mask & (DV_II_50_0_bit | DV_II_53_0_bit)) != 0 {
	// 	mask &= (((((W[57] ^ W[58]) >> 29) & 1) - 1) | ^(DV_II_50_0_bit | DV_II_53_0_bit))
	// }
	TST  $0x10800000, R0
	BEQ  f27
	MOVW W_57+228(FP), R1
	MOVW W_58+232(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $0x00000001, R1, R1
	ORR  $0xef7fffff, R1, R1
	AND  R1, R0, R0

f27:
	// if (mask & (DV_II_52_0_bit | DV_II_54_0_bit)) != 0 {
	// 	mask &= ((((W[56] ^ (W[59] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_II_52_0_bit | DV_II_54_0_bit))
	// }
	TST  $0x28000000, R0
	BEQ  f28
	MOVW W_56+224(FP), R1
	MOVW W_59+236(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xd7ffffff, R1, R1
	AND  R1, R0, R0

f28:
	// if (mask & (DV_II_51_0_bit | DV_II_52_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[56] ^ W[59]) >> 29) & 1)) | ^(DV_II_51_0_bit | DV_II_52_0_bit))
	// }
	TST  $0x0a000000, R0
	BEQ  f29
	MOVW W_56+224(FP), R1
	MOVW W_59+236(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xf5ffffff, R1, R1
	AND  R1, R0, R0

f29:
	// if (mask & (DV_II_49_0_bit | DV_II_52_0_bit)) != 0 {
	// 	mask &= (((((W[56] ^ W[57]) >> 29) & 1) - 1) | ^(DV_II_49_0_bit | DV_II_52_0_bit))
	// }
	TST  $0x08200000, R0
	BEQ  f30
	MOVW W_56+224(FP), R1
	MOVW W_57+228(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $0x00000001, R1, R1
	ORR  $0xf7dfffff, R1, R1
	AND  R1, R0, R0

f30:
	// if (mask & (DV_II_51_0_bit | DV_II_53_0_bit)) != 0 {
	// 	mask &= ((((W[55] ^ (W[58] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_II_51_0_bit | DV_II_53_0_bit))
	// }
	TST  $0x12000000, R0
	BEQ  f31
	MOVW W_55+220(FP), R1
	MOVW W_58+232(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xedffffff, R1, R1
	AND  R1, R0, R0

f31:
	// if (mask & (DV_II_50_0_bit | DV_II_52_0_bit)) != 0 {
	// 	mask &= ((((W[54] ^ (W[57] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_II_50_0_bit | DV_II_52_0_bit))
	// }
	TST  $0x08800000, R0
	BEQ  f32
	MOVW W_54+216(FP), R1
	MOVW W_57+228(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xf77fffff, R1, R1
	AND  R1, R0, R0

f32:
	// if (mask & (DV_II_49_0_bit | DV_II_51_0_bit)) != 0 {
	// 	mask &= ((((W[53] ^ (W[56] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_II_49_0_bit | DV_II_51_0_bit))
	// }
	TST  $0x02200000, R0
	BEQ  f33
	MOVW W_53+212(FP), R1
	MOVW W_56+224(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xfddfffff, R1, R1
	AND  R1, R0, R0

f33:
	// mask &= ((((W[51] ^ (W[50] >> 5)) & (1 << 1)) - (1 << 1)) | ^(DV_I_50_2_bit | DV_II_46_2_bit))
	MOVW W_51+204(FP), R1
	MOVW W_50+200(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	SUB  $0x00000002, R1, R1
	ORR  $0xfffbefff, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[48] ^ W[50]) & (1 << 6)) - (1 << 6)) | ^(DV_I_50_2_bit | DV_II_46_2_bit))
	MOVW W_48+192(FP), R1
	MOVW W_50+200(FP), R2
	EOR  R2, R1, R1
	AND  $0x00000040, R1, R1
	SUB  $0x00000040, R1, R1
	ORR  $0xfffbefff, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_51_0_bit | DV_I_52_0_bit)) != 0 {
	// 	mask &= ((0 - (((W[48] ^ W[55]) >> 29) & 1)) | ^(DV_I_51_0_bit | DV_I_52_0_bit))
	// }
	TST  $0x0000a000, R0
	BEQ  f34
	MOVW W_48+192(FP), R1
	MOVW W_55+220(FP), R2
	EOR  R2, R1, R1
	LSR  $0x1d, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	ORR  $0xffff5fff, R1, R1
	AND  R1, R0, R0

f34:
	// mask &= ((((W[47] ^ W[49]) & (1 << 6)) - (1 << 6)) | ^(DV_I_49_2_bit | DV_I_51_2_bit))
	MOVW W_47+188(FP), R1
	MOVW W_49+196(FP), R2
	EOR  R2, R1, R1
	AND  $0x00000040, R1, R1
	SUB  $0x00000040, R1, R1
	ORR  $0xffffbbff, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[48] ^ (W[47] >> 5)) & (1 << 1)) - (1 << 1)) | ^(DV_I_47_2_bit | DV_II_51_2_bit))
	MOVW W_48+192(FP), R1
	MOVW W_47+188(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	SUB  $0x00000002, R1, R1
	ORR  $0xfbffffbf, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[46] ^ W[48]) & (1 << 6)) - (1 << 6)) | ^(DV_I_48_2_bit | DV_I_50_2_bit))
	MOVW W_46+184(FP), R1
	MOVW W_48+192(FP), R2
	EOR  R2, R1, R1
	AND  $0x00000040, R1, R1
	SUB  $0x00000040, R1, R1
	ORR  $0xffffeeff, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[47] ^ (W[46] >> 5)) & (1 << 1)) - (1 << 1)) | ^(DV_I_46_2_bit | DV_II_50_2_bit))
	MOVW W_47+188(FP), R1
	MOVW W_46+184(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	SUB  $0x00000002, R1, R1
	ORR  $0xfeffffef, R1, R1
	AND  R1, R0, R0

	// mask &= ((0 - ((W[44] ^ (W[45] >> 5)) & (1 << 1))) | ^(DV_I_51_2_bit | DV_II_49_2_bit))
	MOVW W_44+176(FP), R1
	MOVW W_45+180(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	ORR  $0xffbfbfff, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[43] ^ W[45]) & (1 << 6)) - (1 << 6)) | ^(DV_I_47_2_bit | DV_I_49_2_bit))
	MOVW W_43+172(FP), R1
	MOVW W_45+180(FP), R2
	EOR  R2, R1, R1
	AND  $0x00000040, R1, R1
	SUB  $0x00000040, R1, R1
	ORR  $0xfffffbbf, R1, R1
	AND  R1, R0, R0

	// mask &= (((((W[42] ^ W[44]) >> 6) & 1) - 1) | ^(DV_I_46_2_bit | DV_I_48_2_bit))
	MOVW W_42+168(FP), R1
	MOVW W_44+176(FP), R2
	EOR  R2, R1, R1
	LSR  $0x06, R1, R1
	AND  $0x00000001, R1, R1
	SUB  $1, R1, R1
	ORR  $0xfffffeef, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[43] ^ (W[42] >> 5)) & (1 << 1)) - (1 << 1)) | ^(DV_II_46_2_bit | DV_II_51_2_bit))
	MOVW W_43+172(FP), R1
	MOVW W_42+168(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	SUB  $0x00000002, R1, R1
	ORR  $0xfbfbffff, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[42] ^ (W[41] >> 5)) & (1 << 1)) - (1 << 1)) | ^(DV_I_51_2_bit | DV_II_50_2_bit))
	MOVW W_42+168(FP), R1
	MOVW W_41+164(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	SUB  $0x00000002, R1, R1
	ORR  $0xfeffbfff, R1, R1
	AND  R1, R0, R0

	// mask &= ((((W[41] ^ (W[40] >> 5)) & (1 << 1)) - (1 << 1)) | ^(DV_I_50_2_bit | DV_II_49_2_bit))
	MOVW W_41+164(FP), R1
	MOVW W_40+160(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	SUB  $0x00000002, R1, R1
	ORR  $0xffbfefff, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_52_0_bit | DV_II_51_0_bit)) != 0 {
	// 	mask &= ((((W[39] ^ (W[43] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_I_52_0_bit | DV_II_51_0_bit))
	// }
	TST  $0x02008000, R0
	BEQ  f35
	MOVW W_39+156(FP), R1
	MOVW W_43+172(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xfdff7fff, R1, R1
	AND  R1, R0, R0

f35:
	// if (mask & (DV_I_51_0_bit | DV_II_50_0_bit)) != 0 {
	// 	mask &= ((((W[38] ^ (W[42] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_I_51_0_bit | DV_II_50_0_bit))
	// }
	TST  $0x00802000, R0
	BEQ  f36
	MOVW W_38+152(FP), R1
	MOVW W_42+168(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xff7fdfff, R1, R1
	AND  R1, R0, R0

f36:
	// if (mask & (DV_I_48_2_bit | DV_I_51_2_bit)) != 0 {
	// 	mask &= ((0 - ((W[37] ^ (W[38] >> 5)) & (1 << 1))) | ^(DV_I_48_2_bit | DV_I_51_2_bit))
	// }
	TST  $0x00004100, R0
	BEQ  f37
	MOVW W_37+148(FP), R1
	MOVW W_38+152(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	ORR  $0xffffbeff, R1, R1
	AND  R1, R0, R0

f37:
	// if (mask & (DV_I_50_0_bit | DV_II_49_0_bit)) != 0 {
	// 	mask &= ((((W[37] ^ (W[41] >> 25)) & (1 << 4)) - (1 << 4)) | ^(DV_I_50_0_bit | DV_II_49_0_bit))
	// }
	TST  $0x00200800, R0
	BEQ  f38
	MOVW W_37+148(FP), R1
	MOVW W_41+164(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	SUB  $0x00000010, R1, R1
	ORR  $0xffdff7ff, R1, R1
	AND  R1, R0, R0

f38:
	// if (mask & (DV_II_52_0_bit | DV_II_54_0_bit)) != 0 {
	// 	mask &= ((0 - ((W[36] ^ W[38]) & (1 << 4))) | ^(DV_II_52_0_bit | DV_II_54_0_bit))
	// }
	TST  $0x28000000, R0
	BEQ  f39
	MOVW W_36+144(FP), R1
	MOVW W_38+152(FP), R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	NEG  R1, R1
	ORR  $0xd7ffffff, R1, R1
	AND  R1, R0, R0

f39:
	// mask &= ((0 - ((W[35] ^ (W[36] >> 5)) & (1 << 1))) | ^(DV_I_46_2_bit | DV_I_49_2_bit))
	MOVW W_35+140(FP), R1
	MOVW W_36+144(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	ORR  $0xfffffbef, R1, R1
	AND  R1, R0, R0

	// if (mask & (DV_I_51_0_bit | DV_II_47_0_bit)) != 0 {
	// 	mask &= ((((W[35] ^ (W[39] >> 25)) & (1 << 3)) - (1 << 3)) | ^(DV_I_51_0_bit | DV_II_47_0_bit))
	// }
	TST  $0x00082000, R0
	BEQ  f40
	MOVW W_35+140(FP), R1
	MOVW W_39+156(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000008, R1, R1
	SUB  $0x00000008, R1, R1
	ORR  $0xfff7dfff, R1, R1
	AND  R1, R0, R0

f40:
	// if mask != 0
	TST  $0x00000000, R0
	BNE  end

	// if (mask & DV_I_43_0_bit) != 0 {
	// 	if not((W[61]^(W[62]>>5))&(1<<1)) != 0 ||
	// 		not(not((W[59]^(W[63]>>25))&(1<<5))) != 0 ||
	// 		not((W[58]^(W[63]>>30))&(1<<0)) != 0 {
	// 		mask &= ^DV_I_43_0_bit
	// 	}
	// }
	TBZ  $0, R0, f41_skip
	MOVW W_61+244(FP), R1
	MOVW W_62+248(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	CBZ R1, f41_in
	MOVW W_59+236(FP), R1
	MOVW W_63+252(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000020, R1, R1
	CBNZ R1, f41_in
	MOVW W_58+232(FP), R1
	MOVW W_63+252(FP), R2
	LSR  $0x1e, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	CBZ R1, f41_in
	B    f41_skip

f41_in:
	AND  $0xfffffffe, R0, R0

f41_skip:
	// if (mask & DV_I_44_0_bit) != 0 {
	// 	if not((W[62]^(W[63]>>5))&(1<<1)) != 0 ||
	// 		not(not((W[60]^(W[64]>>25))&(1<<5))) != 0 ||
	// 		not((W[59]^(W[64]>>30))&(1<<0)) != 0 {
	// 		mask &= ^DV_I_44_0_bit
	// 	}
	// }
	TBZ  $0x01, R0, f42_skip
	MOVW W_62+248(FP), R1
	MOVW W_63+252(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000002, R1, R1
	NEG  R1, R1
	CBZ  R1, f42_in
	MOVW W_60+240(FP), R1
	MOVW W_64+256(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000020, R1, R1
	CBNZ R1, f42_in
	MOVW W_59+236(FP), R1
	MOVW W_64+256(FP), R2
	LSR  $0x1e, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000001, R1, R1
	NEG  R1, R1
	CBZ  R1, f42_in
	B    f42_skip

f42_in:
	AND  $0xfffffffd, R0, R0

f42_skip:
	// if (mask & DV_I_46_2_bit) != 0 {
	// 	mask &= ((^((W[40] ^ W[42]) >> 2)) | ^DV_I_46_2_bit)
	// }
	TBZ  $0x04, R0, f43
	MOVW W_40+160(FP), R1
	MOVW W_42+168(FP), R2
	EOR  R2, R1, R1
	LSR  $0x02, R1, R1
	MVN  R1, R1
	ORR  $0xffffffef, R1, R1
	AND  R1, R0, R0

f43:
	// if (mask & DV_I_47_2_bit) != 0 {
	// 	if not((W[62]^(W[63]>>5))&(1<<2)) != 0 ||
	// 		not(not((W[41]^W[43])&(1<<6))) != 0 {
	// 		mask &= ^DV_I_47_2_bit
	// 	}
	// }
	TBZ  $6, R0, f44_skip
	MOVW W_62+248(FP), R1
	MOVW W_63+252(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000004, R1, R1
	NEG  R1, R1
	CBZ  R1, f44_in
	MOVW W_41+164(FP), R1
	MOVW W_43+172(FP), R2
	EOR  R2, R1, R1
	AND  $0x00000040, R1, R1
	CBNZ R1, f44_in
	B    f44_skip

f44_in:
	AND  $0xffffffbf, R0, R0

f44_skip:
	// if (mask & DV_I_48_2_bit) != 0 {
	// 	if not((W[63]^(W[64]>>5))&(1<<2)) != 0 ||
	// 		not(not((W[48]^(W[49]<<5))&(1<<6))) != 0 {
	// 		mask &= ^DV_I_48_2_bit
	// 	}
	// }
	TBZ  $0x08, R0, f45_skip
	MOVW W_63+252(FP), R1
	MOVW W_64+256(FP), R2
	LSR  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000004, R1, R1
	NEG  R1, R1
	CBZ  R1, f45_in
	MOVW W_48+192(FP), R1
	MOVW W_49+196(FP), R2
	LSL  $0x05, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000040, R1, R1
	CBNZ R1, f45_in
	B    f45_skip

f45_in:
	AND  $0xfffffeff, R0, R0

f45_skip:
	// if (mask & DV_I_49_2_bit) != 0 {
	// 	if not(not((W[49]^(W[50]<<5))&(1<<6))) != 0 ||
	// 		not((W[42]^W[50])&(1<<1)) != 0 ||
	// 		not(not((W[39]^(W[40]<<5))&(1<<6))) != 0 ||
	// 		not((W[38]^W[40])&(1<<1)) != 0 {
	// 		mask &= ^DV_I_49_2_bit
	// 	}
	// }
	TBZ   $0x0a, R0, f46_skip             // Test bit 10 of R0, skip if clear
	MOVW  W_49+196(FP), R1              // R1 = W_49
	MOVW  W_50+200(FP), R2              // R2 = W_50
	LSL   $0x05, R2, R2                    // R2 = W_50 << 5
	EOR   R2, R1, R1                    // R1 ^= R2
	AND   $0x00000040, R1, R1           // R1 &= 0x40
	CBNZ  R1, f46_in                    // If non-zero, jump to f46_in

	MOVW  W_42+168(FP), R1
	MOVW  W_50+200(FP), R2
	EOR   R2, R1, R1
	AND   $0x00000002, R1, R1
	CBZ   R1, f46_in

	MOVW  W_39+156(FP), R1
	MOVW  W_40+160(FP), R2
	LSL   $0x05, R2, R2
	EOR   R2, R1, R1
	AND   $0x00000040, R1, R1
	CBNZ  R1, f46_in

	MOVW  W_38+152(FP), R1
	MOVW  W_40+160(FP), R2
	EOR   R2, R1, R1
	AND   $0x00000002, R1, R1
	CBZ   R1, f46_in
	B     f46_skip

f46_in:
	AND  $0xfffffbff, R0, R0

f46_skip:
    // if (mask & DV_I_50_0_bit) != 0 {
    //     mask &= (((W[36] ^ W[37]) << 7) | ^DV_I_50_0_bit)
    // }
    TBZ   $0x0b, R0, f47                   // Test bit 11 (DV_I_50_0_bit)
    MOVW  W_36+144(FP), R1
    MOVW  W_37+148(FP), R2
    EOR   R2, R1, R1                     // R1 = W[36] ^ W[37]
    LSL   $0x07, R1, R1                     // R1 <<= 7
    ORR   $0xfffff7ff, R1, R1            // R1 |= ~DV_I_50_0_bit
    AND   R1, R0, R0                     // mask &= R1

f47:
    // if (mask & DV_I_50_2_bit) != 0 {
    //     mask &= (((W[43] ^ W[51]) << 11) | ^DV_I_50_2_bit)
    // }
    TBZ   $0x0c, R0, f48                   // Test bit 12 (DV_I_50_2_bit)
    MOVW  W_43+172(FP), R1
    MOVW  W_51+204(FP), R2
    EOR   R2, R1, R1                     // R1 = W[43] ^ W[51]
    LSL   $0x0b, R1, R1                    // R1 <<= 11
    ORR   $0xffffefff, R1, R1            // R1 |= ~DV_I_50_2_bit
    AND   R1, R0, R0                     // mask &= R1

f48:
    // if (mask & DV_I_51_0_bit) != 0 {
    //     mask &= (((W[37] ^ W[38]) << 9) | ^DV_I_51_0_bit)
    // }
    TBZ   $0x0d, R0, f49                   // Test bit 13 (DV_I_51_0_bit)
    MOVW  W_37+148(FP), R1
    MOVW  W_38+152(FP), R2
    EOR   R2, R1, R1                     // R1 = W[37] ^ W[38]
    LSL   $0x09, R1, R1                     // R1 <<= 9
    ORR   $0xffffdfff, R1, R1            // R1 |= ~DV_I_51_0_bit
    AND   R1, R0, R0                     // mask &= R1

f49:
    // if (mask & DV_I_51_2_bit) != 0 {
    //     if not(not((W[51]^(W[52]<<5))&(1<<6))) != 0 ||
    //         not(not((W[49]^W[51])&(1<<6))) != 0 ||
    //         not(not((W[37]^(W[37]>>5))&(1<<1))) != 0 ||
    //         not(not((W[35]^(W[39]>>25))&(1<<5))) != 0 {
    //         mask &= ^DV_I_51_2_bit
    //     }
    // }
    TBZ   $0x0e, R0, f50_skip                    // Test bit 14 (DV_I_51_2_bit)

    MOVW  W_51+204(FP), R1
    MOVW  W_52+208(FP), R2
    LSL   $0x05, R2, R2
    EOR   R2, R1, R1
    AND   $0x00000040, R1, R1
    CBNZ  R1, f50_in

    MOVW  W_49+196(FP), R1
    MOVW  W_51+204(FP), R2
    EOR   R2, R1, R1
    AND   $0x00000040, R1, R1
    CBNZ  R1, f50_in

    MOVW  W_37+148(FP), R1
    MOVW  W_37+148(FP), R2
    LSR   $0x05, R2, R2
    EOR   R2, R1, R1
    AND   $0x00000002, R1, R1
    CBNZ  R1, f50_in

    MOVW  W_35+140(FP), R1
    MOVW  W_39+156(FP), R2
    LSR   $0x19, R2, R2
    EOR   R2, R1, R1
    AND   $0x00000020, R1, R1
    CBNZ  R1, f50_in

    B     f50_skip

f50_in:
    AND   $0xffffbfff, R0, R0                  // mask &= ~DV_I_51_2_bit

f50_skip:
    // if (mask & DV_I_52_0_bit) != 0 {
    //     mask &= (((W[38] ^ W[39]) << 11) | ^DV_I_52_0_bit)
    // }
    TBZ   $0x0f, R0, f51                         // Test bit 15 (DV_I_52_0_bit)
    MOVW  W_38+152(FP), R1
    MOVW  W_39+156(FP), R2
    EOR   R2, R1, R1
    LSL   $0x0b, R1, R1
    ORR   $0xffff7fff, R1, R1
    AND   R1, R0, R0

f51:
    // if (mask & DV_II_46_2_bit) != 0 {
    //     mask &= (((W[47] ^ W[51]) << 17) | ^DV_II_46_2_bit)
    // }
	TST   $0x00040000, R0
	TBZ   $0x12, R0, f52
	MOVW  W_47+188(FP), R1
	MOVW  W_51+204(FP), R2
	EOR   R2, R1, R1
	LSL   $0x11, R1, R1
	ORR   $0xfffbffff, R1, R1
	AND   R1, R0, R0

f52:
    // if (mask & DV_II_48_0_bit) != 0 {
    //     if not(not((W[36]^(W[40]>>25))&(1<<3))) != 0 ||
    //         not((W[35]^(W[40]<<2))&(1<<30)) != 0 {
    //         mask &= ^DV_II_48_0_bit
    //     }
    // }
    TBZ   $0x14, R0, f53_skip                  // Test bit 20 (DV_II_48_0_bit)

    MOVW  W_36+144(FP), R1
    MOVW  W_40+160(FP), R2
    LSR   $0x19, R2, R2
    EOR   R2, R1, R1
    AND   $0x00000008, R1, R1
    CBNZ  R1, f53_in

    MOVW  W_35+140(FP), R1
    MOVW  W_40+160(FP), R2
    LSL   $0x02, R2, R2
    EOR   R2, R1, R1
    AND   $0x40000000, R1, R1
    CBNZ  R1, f53_in

    B     f53_skip

f53_in:
	AND  $0xffefffff, R0, R0

f53_skip:
	// if (mask & DV_II_49_0_bit) != 0 {
	// 	if not(not((W[37]^(W[41]>>25))&(1<<3))) != 0 ||
	// 		not((W[36]^(W[41]<<2))&(1<<30)) != 0 {
	// 		mask &= ^DV_II_49_0_bit
	// 	}
	// }
    TBZ   $0x15, R0, f54_skip                  // Test bit 21 (DV_II_49_0_bit)

    MOVW  W_37+148(FP), R1
    MOVW  W_41+164(FP), R2
    LSR   $0x19, R2, R2
    EOR   R2, R1, R1
    AND   $0x00000008, R1, R1
    CBNZ  R1, f54_in

    MOVW  W_36+144(FP), R1
    MOVW  W_41+164(FP), R2
    LSL   $0x02, R2, R2
    EOR   R2, R1, R1
    AND   $0x40000000, R1, R1
    CBNZ  R1, f54_in

    B     f54_skip

f54_in:
    AND   $0xffdfffff, R0, R0               // mask &= ~DV_II_49_0_bit

f54_skip:
	// if (mask & DV_II_49_2_bit) != 0 {
	// 	if not(not((W[53]^(W[54]<<5))&(1<<6))) != 0 ||
	// 		not(not((W[51]^W[53])&(1<<6))) != 0 ||
	// 		not((W[50]^W[54])&(1<<1)) != 0 ||
	// 		not(not((W[45]^(W[46]<<5))&(1<<6))) != 0 ||
	// 		not(not((W[37]^(W[41]>>25))&(1<<5))) != 0 ||
	// 		not((W[36]^(W[41]>>30))&(1<<0)) != 0 {
	// 		mask &= ^DV_II_49_2_bit
	// 	}
	// }
	TBZ $0x16, R0, f55_skip

	MOVW W_53+212(FP), R1
	MOVW W_54+216(FP), R2
	LSL  $0x05, R2
	EOR  R2, R1
	AND  $0x00000040, R1
	CMP  $0x00000000, R1
	BNE  f55_in
	MOVW W_51+204(FP), R1
	MOVW W_53+212(FP), R2
	EOR  R2, R1
	AND  $0x00000040, R1
	CMP  $0x00000000, R1
	BNE  f55_in
	MOVW W_50+200(FP), R1
	MOVW W_54+216(FP), R2
	EOR  R2, R1
	AND  $0x00000002, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f55_in
	MOVW W_45+180(FP), R1
	MOVW W_46+184(FP), R2
	LSL  $0x05, R2
	EOR  R2, R1
	AND  $0x00000040, R1
	CMP  $0x00000000, R1
	BNE  f55_in
	MOVW W_37+148(FP), R1
	MOVW W_41+164(FP), R2
	LSR  $0x19, R2
	EOR  R2, R1
	AND  $0x00000020, R1
	CMP  $0x00000000, R1
	BNE  f55_in
	MOVW W_36+144(FP), R1
	MOVW W_41+164(FP), R2
	LSR  $0x1e, R2
	EOR  R2, R1
	AND  $0x00000001, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f55_in
	JMP  f55_skip

f55_in:
	AND  $0xffbfffff, R0

f55_skip:
	// if (mask & DV_II_50_0_bit) != 0 {
	// 	if not((W[55]^W[58])&(1<<29)) != 0 ||
	// 		not(not((W[38]^(W[42]>>25))&(1<<3))) != 0 ||
	// 		not((W[37]^(W[42]<<2))&(1<<30)) != 0 {
	// 		mask &= ^DV_II_50_0_bit
	// 	}
	// }
	TBZ $0x17, R0, f56_skip

	MOVW W_55+220(FP), R1
	MOVW W_58+232(FP), R2
	EOR  R2, R1
	AND  $0x20000000, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f56_in
	MOVW W_38+152(FP), R1
	MOVW W_42+168(FP), R2
	LSR  $0x19, R2
	EOR  R2, R1
	AND  $0x00000008, R1
	CMP  $0x00000000, R1
	BNE  f56_in
	MOVW W_37+148(FP), R1
	MOVW W_42+168(FP), R2
	LSR  $0x02, R2
	EOR  R2, R1
	AND  $0x40000000, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f56_in
	JMP  f56_skip

f56_in:
	AND  $0xff7fffff, R0

f56_skip:
	// if (mask & DV_II_50_2_bit) != 0 {
	// 	if not(not((W[54]^(W[55]<<5))&(1<<6))) != 0 ||
	// 		not(not((W[52]^W[54])&(1<<6))) != 0 ||
	// 		not((W[51]^W[55])&(1<<1)) != 0 ||
	// 		not((W[45]^W[47])&(1<<1)) != 0 ||
	// 		not(not((W[38]^(W[42]>>25))&(1<<5))) != 0 ||
	// 		not((W[37]^(W[42]>>30))&(1<<0)) != 0 {
	// 		mask &= ^DV_II_50_2_bit
	// 	}
	// }
	TBZ $0x18, R0, f57_skip

	MOVW W_54+216(FP), R1
	MOVW W_55+220(FP), R2
	LSL  $0x05, R2
	EOR  R2, R1
	AND  $0x00000040, R1
	CMP  $0x00000000, R1
	BNE  f57_in
	MOVW W_52+208(FP), R1
	MOVW W_54+216(FP), R2
	EOR  R2, R1
	AND  $0x00000040, R1
	CMP  $0x00000000, R1
	BNE  f57_in
	MOVW W_51+204(FP), R1
	MOVW W_55+220(FP), R2
	EOR  R2, R1
	AND  $0x00000002, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f57_in
	MOVW W_45+180(FP), R1
	MOVW W_47+188(FP), R2
	EOR  R2, R1
	AND  $0x00000002, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f57_in
	MOVW W_38+152(FP), R1
	MOVW W_42+168(FP), R2
	LSR  $0x19, R2
	EOR  R2, R1
	AND  $0x00000020, R1
	CMP  $0x00000000, R1
	BNE  f57_in
	MOVW W_37+148(FP), R1
	MOVW W_42+168(FP), R2
	LSR  $0x1e, R2
	EOR  R2, R1
	AND  $0x00000001, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f57_in
	JMP  f57_skip

f57_in:
	AND  $0xfeffffff, R0

f57_skip:
	// if (mask & DV_II_51_0_bit) != 0 {
	// 	if not(not((W[39]^(W[43]>>25))&(1<<3))) != 0 ||
	// 		not((W[38]^(W[43]<<2))&(1<<30)) != 0 {
	// 		mask &= ^DV_II_51_0_bit
	// 	}
	// }
	TBZ $0x19, R0, f58_skip

	MOVW W_39+156(FP), R1
	MOVW W_43+172(FP), R2
	LSR  $0x19, R2
	EOR  R2, R1
	AND  $0x00000008, R1
	CMP  $0x00000000, R1
	BNE  f58_in
	MOVW W_38+152(FP), R1
	MOVW W_43+172(FP), R2
	LSL  $0x02, R2
	EOR  R2, R1
	AND  $0x40000000, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f58_in
	JMP  f58_skip

f58_in:
	AND  $0xfdffffff, R0

f58_skip:
	// if (mask & DV_II_51_2_bit) != 0 {
	// 	if not(not((W[55]^(W[56]<<5))&(1<<6))) != 0 ||
	// 		not(not((W[53]^W[55])&(1<<6))) != 0 ||
	// 		not((W[52]^W[56])&(1<<1)) != 0 ||
	// 		not((W[46]^W[48])&(1<<1)) != 0 ||
	// 		not(not((W[39]^(W[43]>>25))&(1<<5))) != 0 ||
	// 		not((W[38]^(W[43]>>30))&(1<<0)) != 0 {
	// 		mask &= ^DV_II_51_2_bit
	// 	}
	// }	
	TBZ $0x1a, R0, f59_skip

	MOVW W_55+220(FP), R1
	MOVW W_56+224(FP), R2
	LSL  $0x05, R2
	EOR  R2, R1
	AND  $0x00000040, R1
	CMP  $0x00000000, R1
	BNE  f59_in
	MOVW W_53+212(FP), R1
	MOVW W_55+220(FP), R2
	EOR  R2, R1
	AND  $0x00000040, R1
	CMP  $0x00000000, R1
	BNE  f59_in
	MOVW W_52+208(FP), R1
	MOVW W_56+224(FP), R2
	EOR  R2, R1
	AND  $0x00000002, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f59_in
	MOVW W_46+184(FP), R1
	MOVW W_48+192(FP), R2
	EOR  R2, R1
	AND  $0x00000002, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f59_in
	MOVW W_39+156(FP), R1
	MOVW W_43+172(FP), R2
	LSR  $0x19, R2
	EOR  R2, R1
	AND  $0x00000020, R1
	CMP  $0x00000000, R1
	BNE  f59_in
	MOVW W_38+152(FP), R1
	MOVW W_43+172(FP), R2
	LSR  $0x1e, R2
	EOR  R2, R1
	AND  $0x00000001, R1
	NEG  R1, R1
	CMP  $0x00000000, R1
	BEQ  f59_in
	JMP  f59_skip

f59_in:
	AND  $0xfbffffff, R0

f59_skip:
	// if (mask & DV_II_52_0_bit) != 0 {
	// 	if not(not((W[59]^W[60])&(1<<29))) != 0 ||
	// 		not(not((W[40]^(W[44]>>25))&(1<<3))) != 0 ||
	// 		not(not((W[40]^(W[44]>>25))&(1<<4))) != 0 ||
	// 		not((W[39]^(W[44]<<2))&(1<<30)) != 0 {
	// 		mask &= ^DV_II_52_0_bit
	// 	}
	// }
	TBZ  $0x1b, R0, f60_skip
	MOVW W_59+236(FP), R1
	MOVW W_60+240(FP), R2
	EOR  R2, R1, R1
	AND  $0x20000000, R1, R1
	CBNZ R1, f60_in
	MOVW W_40+160(FP), R1
	MOVW W_44+176(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000008, R1, R1
	CBNZ R1, f60_in
	MOVW W_40+160(FP), R1
	MOVW W_44+176(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f60_in
	MOVW W_39+156(FP), R1
	MOVW W_44+176(FP), R2
	LSL  $0x02, R2, R2
	EOR  R2, R1, R1
	AND  $0x40000000, R1, R1
	NEG  R1, R1
	CBZ R1, f60_in
	B    f60_skip

f60_in:
	AND  $0xf7ffffff, R0, R0

f60_skip:
	// if (mask & DV_II_53_0_bit) != 0 {
	// 	if not((W[58]^W[61])&(1<<29)) != 0 ||
	// 		not(not((W[57]^(W[61]>>25))&(1<<4))) != 0 ||
	// 		not(not((W[41]^(W[45]>>25))&(1<<3))) != 0 ||
	// 		not(not((W[41]^(W[45]>>25))&(1<<4))) != 0 {
	// 		mask &= ^DV_II_53_0_bit
	// 	}
	// }
	TBZ  $0x1c, R0, f61_skip
	MOVW W_58+232(FP), R1
	MOVW W_61+244(FP), R2
	EOR  R2, R1, R1
	AND  $0x20000000, R1, R1
	NEG  R1, R1
	CBZ R1, f61_in
	MOVW W_57+228(FP), R1
	MOVW W_61+244(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f61_in
	MOVW W_41+164(FP), R1
	MOVW W_45+180(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000008, R1, R1
	CBNZ R1, f61_in
	MOVW W_41+164(FP), R1
	MOVW W_45+180(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f61_in
	B    f61_skip

f61_in:
	AND  $0xefffffff, R0, R0

f61_skip:
	// if (mask & DV_II_54_0_bit) != 0 {
	// 	if not(not((W[58]^(W[62]>>25))&(1<<4))) != 0 ||
	// 		not(not((W[42]^(W[46]>>25))&(1<<3))) != 0 ||
	// 		not(not((W[42]^(W[46]>>25))&(1<<4))) != 0 {
	// 		mask &= ^DV_II_54_0_bit
	// 	}
	// }
	TBZ  $0x1d, R0, f62_skip
	MOVW W_58+232(FP), R1
	MOVW W_62+248(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f62_in
	MOVW W_42+168(FP), R1
	MOVW W_46+184(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000008, R1, R1
	CBNZ R1, f62_in
	MOVW W_42+168(FP), R1
	MOVW W_46+184(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f62_in
	B    f62_skip

f62_in:
	AND  $0xdfffffff, R0, R0

f62_skip:
	// if (mask & DV_II_55_0_bit) != 0 {
	// 	if not(not((W[59]^(W[63]>>25))&(1<<4))) != 0 ||
	// 		not(not((W[57]^(W[59]>>25))&(1<<4))) != 0 ||
	// 		not(not((W[43]^(W[47]>>25))&(1<<3))) != 0 ||
	// 		not(not((W[43]^(W[47]>>25))&(1<<4))) != 0 {
	// 		mask &= ^DV_II_55_0_bit
	// 	}
	// }
	TBZ  $0x1e, R0, f63_skip
	MOVW W_59+236(FP), R1
	MOVW W_63+252(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f63_in
	MOVW W_57+228(FP), R1
	MOVW W_59+236(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f63_in
	MOVW W_43+172(FP), R1
	MOVW W_47+188(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000008, R1, R1
	CBNZ R1, f63_in
	MOVW W_43+172(FP), R1
	MOVW W_47+188(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f63_in
	B    f63_skip

f63_in:
	AND  $0xbfffffff, R0, R0

f63_skip:
	// if (mask & DV_II_56_0_bit) != 0 {
	// 	if not(not((W[60]^(W[64]>>25))&(1<<4))) != 0 ||
	// 		not(not((W[44]^(W[48]>>25))&(1<<3))) != 0 ||
	// 		not(not((W[44]^(W[48]>>25))&(1<<4))) != 0 {
	// 		mask &= ^DV_II_56_0_bit
	// 	}
	// }
	TBZ  $0x1f, R0, f64_skip
	MOVW W_60+240(FP), R1
	MOVW W_64+256(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f64_in
	MOVW W_44+176(FP), R1
	MOVW W_48+192(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000008, R1, R1
	CBNZ R1, f64_in
	MOVW W_44+176(FP), R1
	MOVW W_48+192(FP), R2
	LSR  $0x19, R2, R2
	EOR  R2, R1, R1
	AND  $0x00000010, R1, R1
	CBNZ R1, f64_in
	B    f64_skip

f64_in:
	AND $0x7fffffff, R0, R0

f64_skip:
end:
	MOVW R0, ret+320(FP)
	RET
