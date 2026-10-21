//go:build !noasm && gc && arm64 && !amd64 && sha1cd_asmtest

#include "textflag.h"

// callBlockARM64DirtyRegs calls blockARM64 with every general-purpose and
// vector register the caller does not own set to all ones. It is only built
// with the sha1cd_asmtest tag, to catch the assembly reading a register before
// writing it.
//
// func callBlockARM64DirtyRegs(h []uint32, p []byte, m1 []uint32, cs [][5]uint32)
TEXT ·callBlockARM64DirtyRegs(SB), NOSPLIT, $104-96
	MOVD	h_base+0(FP), R0
	MOVD	R0, 8(RSP)
	MOVD	h_len+8(FP), R0
	MOVD	R0, 16(RSP)
	MOVD	h_cap+16(FP), R0
	MOVD	R0, 24(RSP)
	MOVD	p_base+24(FP), R0
	MOVD	R0, 32(RSP)
	MOVD	p_len+32(FP), R0
	MOVD	R0, 40(RSP)
	MOVD	p_cap+40(FP), R0
	MOVD	R0, 48(RSP)
	MOVD	m1_base+48(FP), R0
	MOVD	R0, 56(RSP)
	MOVD	m1_len+56(FP), R0
	MOVD	R0, 64(RSP)
	MOVD	m1_cap+64(FP), R0
	MOVD	R0, 72(RSP)
	MOVD	cs_base+72(FP), R0
	MOVD	R0, 80(RSP)
	MOVD	cs_len+80(FP), R0
	MOVD	R0, 88(RSP)
	MOVD	cs_cap+88(FP), R0
	MOVD	R0, 96(RSP)

	// R18 is reserved by the platform, R27 by the assembler, R28 holds g,
	// R29 is the frame pointer and R30 the link register.
	MOVD	$-1, R0
	MOVD	R0, R1
	MOVD	R0, R2
	MOVD	R0, R3
	MOVD	R0, R4
	MOVD	R0, R5
	MOVD	R0, R6
	MOVD	R0, R7
	MOVD	R0, R8
	MOVD	R0, R9
	MOVD	R0, R10
	MOVD	R0, R11
	MOVD	R0, R12
	MOVD	R0, R13
	MOVD	R0, R14
	MOVD	R0, R15
	MOVD	R0, R16
	MOVD	R0, R17
	MOVD	R0, R19
	MOVD	R0, R20
	MOVD	R0, R21
	MOVD	R0, R22
	MOVD	R0, R23
	MOVD	R0, R24
	MOVD	R0, R25
	MOVD	R0, R26

	VMOVI	$0xff, V0.B16
	VMOVI	$0xff, V1.B16
	VMOVI	$0xff, V2.B16
	VMOVI	$0xff, V3.B16
	VMOVI	$0xff, V4.B16
	VMOVI	$0xff, V5.B16
	VMOVI	$0xff, V6.B16
	VMOVI	$0xff, V7.B16
	VMOVI	$0xff, V8.B16
	VMOVI	$0xff, V9.B16
	VMOVI	$0xff, V10.B16
	VMOVI	$0xff, V11.B16
	VMOVI	$0xff, V12.B16
	VMOVI	$0xff, V13.B16
	VMOVI	$0xff, V14.B16
	VMOVI	$0xff, V15.B16
	VMOVI	$0xff, V16.B16
	VMOVI	$0xff, V17.B16
	VMOVI	$0xff, V18.B16
	VMOVI	$0xff, V19.B16
	VMOVI	$0xff, V20.B16
	VMOVI	$0xff, V21.B16
	VMOVI	$0xff, V22.B16
	VMOVI	$0xff, V23.B16
	VMOVI	$0xff, V24.B16
	VMOVI	$0xff, V25.B16
	VMOVI	$0xff, V26.B16
	VMOVI	$0xff, V27.B16
	VMOVI	$0xff, V28.B16
	VMOVI	$0xff, V29.B16
	VMOVI	$0xff, V30.B16
	VMOVI	$0xff, V31.B16

	BL	·blockARM64(SB)
	RET
