//go:build !noasm && gc && arm64 && freebsd

#include "textflag.h"

// func getisar0() uint64
TEXT ·getisar0(SB), NOSPLIT, $0-8
	MRS  ID_AA64ISAR0_EL1, R0
	MOVD R0, ret+0(FP)
	RET
