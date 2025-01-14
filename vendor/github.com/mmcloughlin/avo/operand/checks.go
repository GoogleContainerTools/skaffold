package operand

import "github.com/mmcloughlin/avo/reg"

// Pure type assertion checks:

// IsRegister returns whether op has type reg.Register.
func IsRegister(op Op) bool { _, ok := op.(reg.Register); return ok }

// IsMem returns whether op has type Mem.
func IsMem(op Op) bool { _, ok := op.(Mem); return ok }

// IsRel returns whether op has type Rel.
func IsRel(op Op) bool { _, ok := op.(Rel); return ok }

// Checks corresponding to specific operand types in the Intel Manual:

// Is1 returns true if op is the immediate constant 1.
func Is1(op Op) bool {
	i, ok := op.(U8)
	return ok && i == 1
}

// Is3 returns true if op is the immediate constant 3.
func Is3(op Op) bool {
	i, ok := op.(U8)
	return ok && i == 3
}

// IsIMM2U returns true if op is a 2-bit unsigned immediate (less than 4).
func IsIMM2U(op Op) bool {
	i, ok := op.(U8)
	return ok && i < 4
}

// IsIMM8 returns true is op is an 8-bit immediate.
func IsIMM8(op Op) bool {
	_, uok := op.(U8)
	_, iok := op.(I8)
	return uok || iok
}

// IsIMM16 returns true is op is a 16-bit immediate.
func IsIMM16(op Op) bool {
	_, uok := op.(U16)
	_, iok := op.(I16)
	return uok || iok
}

// IsIMM32 returns true is op is a 32-bit immediate.
func IsIMM32(op Op) bool {
	_, uok := op.(U32)
	_, iok := op.(I32)
	return uok || iok
}

// IsIMM64 returns true is op is a 64-bit immediate.
func IsIMM64(op Op) bool {
	_, uok := op.(U64)
	_, iok := op.(I64)
	return uok || iok
}

// IsAL returns true if op is the AL register.
func IsAL(op Op) bool {
	return op == reg.AL
}

// IsCL returns true if op is the CL register.
func IsCL(op Op) bool {
	return op == reg.CL
}

// IsAX returns true if op is the 16-bit AX register.
func IsAX(op Op) bool {
	return op == reg.AX
}

// IsEAX returns true if op is the 32-bit EAX register.
func IsEAX(op Op) bool {
	return op == reg.EAX
}

// IsRAX returns true if op is the 64-bit RAX register.
func IsRAX(op Op) bool {
	return op == reg.RAX
}

// IsR8 returns true if op is an 8-bit general-purpose register.
func IsR8(op Op) bool {
	return IsGP(op, 1)
}

// IsR16 returns true if op is a 16-bit general-purpose register.
func IsR16(op Op) bool {
	return IsGP(op, 2)
}

// IsR32 returns true if op is a 32-bit general-purpose register.
func IsR32(op Op) bool {
	return IsGP(op, 4)
}

// IsR64 returns true if op is a 64-bit general-purpose register.
func IsR64(op Op) bool {
	return IsGP(op, 8)
}

// IsPseudo returns true if op is a pseudo register.
func IsPseudo(op Op) bool {
	return IsRegisterKind(op, reg.KindPseudo)
}

// IsGP returns true if op is a general-purpose register of size n bytes.
func IsGP(op Op, n uint) bool {
	return IsRegisterKindSize(op, reg.KindGP, n)
}

// IsXMM0 returns true if op is the X0 register.
func IsXMM0(op Op) bool {
	return op == reg.X0
}

// IsXMM returns true if op is a 128-bit XMM register.
func IsXMM(op Op) bool {
	return IsRegisterKindSize(op, reg.KindVector, 16)
}

// IsYMM returns true if op is a 256-bit YMM register.
func IsYMM(op Op) bool {
	return IsRegisterKindSize(op, reg.KindVector, 32)
}

// IsZMM returns true if op is a 512-bit ZMM register.
func IsZMM(op Op) bool {
	return IsRegisterKindSize(op, reg.KindVector, 64)
}

// IsK returns true if op is an Opmask register.
func IsK(op Op) bool {
	return IsRegisterKind(op, reg.KindOpmask)
}

// IsRegisterKindSize returns true if op is a register of the given kind and size in bytes.
func IsRegisterKindSize(op Op, k reg.Kind, n uint) bool {
	r, ok := op.(reg.Register)
	return ok && r.Kind() == k && r.Size() == n
}

// IsRegisterKind returns true if op is a register of the given kind.
func IsRegisterKind(op Op, k reg.Kind) bool {
	r, ok := op.(reg.Register)
	return ok && r.Kind() == k
}

// IsM returns true if op is a 16-, 32- or 64-bit memory operand.
func IsM(op Op) bool {
	// TODO(mbm): confirm "m" check is defined correctly
	// Intel manual: "A 16-, 32- or 64-bit operand in memory."
	return IsM16(op) || IsM32(op) || IsM64(op)
}

// IsM8 returns true if op is an 8-bit memory operand.
func IsM8(op Op) bool {
	// TODO(mbm): confirm "m8" check is defined correctly
	// Intel manual: "A byte operand in memory, usually expressed as a variable or
	// array name, but pointed to by the DS:(E)SI or ES:(E)DI registers. In 64-bit
	// mode, it is pointed to by the RSI or RDI registers."
	return IsMSize(op, 1)
}

// IsM16 returns true if op is a 16-bit memory operand.
func IsM16(op Op) bool {
	return IsMSize(op, 2)
}

// IsM32 returns true if op is a 16-bit memory operand.
func IsM32(op Op) bool {
	return IsMSize(op, 4)
}

// IsM64 returns true if op is a 64-bit memory operand.
func IsM64(op Op) bool {
	return IsMSize(op, 8)
}

// IsMSize returns true if op is a memory operand using general-purpose address
// registers of the given size in bytes.
func IsMSize(op Op, n uint) bool {
	// TODO(mbm): should memory operands have a size attribute as well?
	// TODO(mbm): m8,m16,m32,m64 checks do not actually check size
	m, ok := op.(Mem)
	return ok && IsMReg(m.Base) && (m.Index == nil || IsMReg(m.Index))
}

// IsMReg returns true if op is a register that can be used in a memory operand.
func IsMReg(op Op) bool {
	return IsPseudo(op) || IsRegisterKind(op, reg.KindGP)
}

// IsM128 returns true if op is a 128-bit memory operand.
func IsM128(op Op) bool {
	// TODO(mbm): should "m128" be the same as "m64"?
	return IsM64(op)
}

// IsM256 returns true if op is a 256-bit memory operand.
func IsM256(op Op) bool {
	// TODO(mbm): should "m256" be the same as "m64"?
	return IsM64(op)
}

// IsM512 returns true if op is a 512-bit memory operand.
func IsM512(op Op) bool {
	// TODO(mbm): should "m512" be the same as "m64"?
	return IsM64(op)
}

// IsVM32X returns true if op is a vector memory operand with 32-bit XMM index.
func IsVM32X(op Op) bool {
	return IsVmx(op)
}

// IsVM64X returns true if op is a vector memory operand with 64-bit XMM index.
func IsVM64X(op Op) bool {
	return IsVmx(op)
}

// IsVmx returns true if op is a vector memory operand with XMM index.
func IsVmx(op Op) bool {
	return isvm(op, IsXMM)
}

// IsVM32Y returns true if op is a vector memory operand with 32-bit YMM index.
func IsVM32Y(op Op) bool {
	return IsVmy(op)
}

// IsVM64Y returns true if op is a vector memory operand with 64-bit YMM index.
func IsVM64Y(op Op) bool {
	return IsVmy(op)
}

// IsVmy returns true if op is a vector memory operand with YMM index.
func IsVmy(op Op) bool {
	return isvm(op, IsYMM)
}

// IsVM32Z returns true if op is a vector memory operand with 32-bit ZMM index.
func IsVM32Z(op Op) bool {
	return IsVmz(op)
}

// IsVM64Z returns true if op is a vector memory operand with 64-bit ZMM index.
func IsVM64Z(op Op) bool {
	return IsVmz(op)
}

// IsVmz returns true if op is a vector memory operand with ZMM index.
func IsVmz(op Op) bool {
	return isvm(op, IsZMM)
}

func isvm(op Op, idx func(Op) bool) bool {
	m, ok := op.(Mem)
	return ok && IsR64(m.Base) && idx(m.Index)
}

// IsREL8 returns true if op is an 8-bit offset relative to instruction pointer.
func IsREL8(op Op) bool {
	r, ok := op.(Rel)
	return ok && r == Rel(int8(r))
}

// IsREL32 returns true if op is an offset relative to instruction pointer, or a
// label reference.
func IsREL32(op Op) bool {
	// TODO(mbm): should labels be considered separately?
	_, rel := op.(Rel)
	_, label := op.(LabelRef)
	return rel || label
}
