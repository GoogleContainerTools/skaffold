package reg

// Register kinds.
const (
	KindPseudo Kind = iota
	KindGP
	KindVector
	KindOpmask
)

// Declare register families.
var (
	Pseudo         = &Family{Kind: KindPseudo}
	GeneralPurpose = &Family{Kind: KindGP}
	Vector         = &Family{Kind: KindVector}
	Opmask         = &Family{Kind: KindOpmask}

	Families = []*Family{
		Pseudo,
		GeneralPurpose,
		Vector,
		Opmask,
	}
)

var familiesByKind = map[Kind]*Family{}

func init() {
	for _, f := range Families {
		familiesByKind[f.Kind] = f
	}
}

// FamilyOfKind returns the Family of registers of the given kind, or nil if not found.
func FamilyOfKind(k Kind) *Family {
	return familiesByKind[k]
}

// Pseudo registers.
var (
	FramePointer   = Pseudo.define(S0, 0, "FP")
	ProgramCounter = Pseudo.define(S0, 0, "PC")
	StaticBase     = Pseudo.define(S0, 0, "SB")
	StackPointer   = Pseudo.define(S0, 0, "SP")
)

// GP provides additional methods for general purpose registers.
type GP interface {
	As8() Register
	As8L() Register
	As8H() Register
	As16() Register
	As32() Register
	As64() Register
}

// GPPhysical is a general-purpose physical register.
type GPPhysical interface {
	Physical
	GP
}

type gpp struct {
	Physical
}

func newgpp(r Physical) GPPhysical { return gpp{Physical: r} }

func (p gpp) As8() Register  { return newgpp(p.as(S8).(Physical)) }
func (p gpp) As8L() Register { return newgpp(p.as(S8L).(Physical)) }
func (p gpp) As8H() Register { return newgpp(p.as(S8H).(Physical)) }
func (p gpp) As16() Register { return newgpp(p.as(S16).(Physical)) }
func (p gpp) As32() Register { return newgpp(p.as(S32).(Physical)) }
func (p gpp) As64() Register { return newgpp(p.as(S64).(Physical)) }

// GPVirtual is a general-purpose virtual register.
type GPVirtual interface {
	Virtual
	GP
}

type gpv struct {
	Virtual
}

func newgpv(v Virtual) GPVirtual { return gpv{Virtual: v} }

func (v gpv) As8() Register  { return newgpv(v.as(S8).(Virtual)) }
func (v gpv) As8L() Register { return newgpv(v.as(S8L).(Virtual)) }
func (v gpv) As8H() Register { return newgpv(v.as(S8H).(Virtual)) }
func (v gpv) As16() Register { return newgpv(v.as(S16).(Virtual)) }
func (v gpv) As32() Register { return newgpv(v.as(S32).(Virtual)) }
func (v gpv) As64() Register { return newgpv(v.as(S64).(Virtual)) }

func gp(s Spec, id Index, name string, flags ...Info) GPPhysical {
	r := newgpp(newregister(GeneralPurpose, s, id, name, flags...))
	GeneralPurpose.add(r)
	return r
}

// General purpose registers.
var (
	// Low byte.
	AL = gp(S8L, 0, "AL")
	CL = gp(S8L, 1, "CL")
	DL = gp(S8L, 2, "DL")
	BL = gp(S8L, 3, "BL")

	// High byte.
	AH = gp(S8H, 0, "AH")
	CH = gp(S8H, 1, "CH")
	DH = gp(S8H, 2, "DH")
	BH = gp(S8H, 3, "BH")

	// 8-bit.
	SPB  = gp(S8, 4, "SP", Restricted)
	BPB  = gp(S8, 5, "BP", BasePointer)
	SIB  = gp(S8, 6, "SI")
	DIB  = gp(S8, 7, "DI")
	R8B  = gp(S8, 8, "R8")
	R9B  = gp(S8, 9, "R9")
	R10B = gp(S8, 10, "R10")
	R11B = gp(S8, 11, "R11")
	R12B = gp(S8, 12, "R12")
	R13B = gp(S8, 13, "R13")
	R14B = gp(S8, 14, "R14")
	R15B = gp(S8, 15, "R15")

	// 16-bit.
	AX   = gp(S16, 0, "AX")
	CX   = gp(S16, 1, "CX")
	DX   = gp(S16, 2, "DX")
	BX   = gp(S16, 3, "BX")
	SP   = gp(S16, 4, "SP", Restricted)
	BP   = gp(S16, 5, "BP", BasePointer)
	SI   = gp(S16, 6, "SI")
	DI   = gp(S16, 7, "DI")
	R8W  = gp(S16, 8, "R8")
	R9W  = gp(S16, 9, "R9")
	R10W = gp(S16, 10, "R10")
	R11W = gp(S16, 11, "R11")
	R12W = gp(S16, 12, "R12")
	R13W = gp(S16, 13, "R13")
	R14W = gp(S16, 14, "R14")
	R15W = gp(S16, 15, "R15")

	// 32-bit.
	EAX  = gp(S32, 0, "AX")
	ECX  = gp(S32, 1, "CX")
	EDX  = gp(S32, 2, "DX")
	EBX  = gp(S32, 3, "BX")
	ESP  = gp(S32, 4, "SP", Restricted)
	EBP  = gp(S32, 5, "BP", BasePointer)
	ESI  = gp(S32, 6, "SI")
	EDI  = gp(S32, 7, "DI")
	R8L  = gp(S32, 8, "R8")
	R9L  = gp(S32, 9, "R9")
	R10L = gp(S32, 10, "R10")
	R11L = gp(S32, 11, "R11")
	R12L = gp(S32, 12, "R12")
	R13L = gp(S32, 13, "R13")
	R14L = gp(S32, 14, "R14")
	R15L = gp(S32, 15, "R15")

	// 64-bit.
	RAX = gp(S64, 0, "AX")
	RCX = gp(S64, 1, "CX")
	RDX = gp(S64, 2, "DX")
	RBX = gp(S64, 3, "BX")
	RSP = gp(S64, 4, "SP", Restricted)
	RBP = gp(S64, 5, "BP", BasePointer)
	RSI = gp(S64, 6, "SI")
	RDI = gp(S64, 7, "DI")
	R8  = gp(S64, 8, "R8")
	R9  = gp(S64, 9, "R9")
	R10 = gp(S64, 10, "R10")
	R11 = gp(S64, 11, "R11")
	R12 = gp(S64, 12, "R12")
	R13 = gp(S64, 13, "R13")
	R14 = gp(S64, 14, "R14")
	R15 = gp(S64, 15, "R15")
)

// Vec provides methods for vector registers.
type Vec interface {
	AsX() Register
	AsY() Register
	AsZ() Register
}

// VecPhysical is a physical vector register.
type VecPhysical interface {
	Physical
	Vec
}

type vecp struct {
	Physical
	Vec
}

func newvecp(r Physical) VecPhysical { return vecp{Physical: r} }

func (p vecp) AsX() Register { return newvecp(p.as(S128).(Physical)) }
func (p vecp) AsY() Register { return newvecp(p.as(S256).(Physical)) }
func (p vecp) AsZ() Register { return newvecp(p.as(S512).(Physical)) }

// VecVirtual is a virtual vector register.
type VecVirtual interface {
	Virtual
	Vec
}

type vecv struct {
	Virtual
	Vec
}

func newvecv(v Virtual) VecVirtual { return vecv{Virtual: v} }

func (v vecv) AsX() Register { return newvecv(v.as(S128).(Virtual)) }
func (v vecv) AsY() Register { return newvecv(v.as(S256).(Virtual)) }
func (v vecv) AsZ() Register { return newvecv(v.as(S512).(Virtual)) }

func vec(s Spec, id Index, name string, flags ...Info) VecPhysical {
	r := newvecp(newregister(Vector, s, id, name, flags...))
	Vector.add(r)
	return r
}

// Vector registers.
var (
	// 128-bit.
	X0  = vec(S128, 0, "X0")
	X1  = vec(S128, 1, "X1")
	X2  = vec(S128, 2, "X2")
	X3  = vec(S128, 3, "X3")
	X4  = vec(S128, 4, "X4")
	X5  = vec(S128, 5, "X5")
	X6  = vec(S128, 6, "X6")
	X7  = vec(S128, 7, "X7")
	X8  = vec(S128, 8, "X8")
	X9  = vec(S128, 9, "X9")
	X10 = vec(S128, 10, "X10")
	X11 = vec(S128, 11, "X11")
	X12 = vec(S128, 12, "X12")
	X13 = vec(S128, 13, "X13")
	X14 = vec(S128, 14, "X14")
	X15 = vec(S128, 15, "X15")
	X16 = vec(S128, 16, "X16")
	X17 = vec(S128, 17, "X17")
	X18 = vec(S128, 18, "X18")
	X19 = vec(S128, 19, "X19")
	X20 = vec(S128, 20, "X20")
	X21 = vec(S128, 21, "X21")
	X22 = vec(S128, 22, "X22")
	X23 = vec(S128, 23, "X23")
	X24 = vec(S128, 24, "X24")
	X25 = vec(S128, 25, "X25")
	X26 = vec(S128, 26, "X26")
	X27 = vec(S128, 27, "X27")
	X28 = vec(S128, 28, "X28")
	X29 = vec(S128, 29, "X29")
	X30 = vec(S128, 30, "X30")
	X31 = vec(S128, 31, "X31")

	// 256-bit.
	Y0  = vec(S256, 0, "Y0")
	Y1  = vec(S256, 1, "Y1")
	Y2  = vec(S256, 2, "Y2")
	Y3  = vec(S256, 3, "Y3")
	Y4  = vec(S256, 4, "Y4")
	Y5  = vec(S256, 5, "Y5")
	Y6  = vec(S256, 6, "Y6")
	Y7  = vec(S256, 7, "Y7")
	Y8  = vec(S256, 8, "Y8")
	Y9  = vec(S256, 9, "Y9")
	Y10 = vec(S256, 10, "Y10")
	Y11 = vec(S256, 11, "Y11")
	Y12 = vec(S256, 12, "Y12")
	Y13 = vec(S256, 13, "Y13")
	Y14 = vec(S256, 14, "Y14")
	Y15 = vec(S256, 15, "Y15")
	Y16 = vec(S256, 16, "Y16")
	Y17 = vec(S256, 17, "Y17")
	Y18 = vec(S256, 18, "Y18")
	Y19 = vec(S256, 19, "Y19")
	Y20 = vec(S256, 20, "Y20")
	Y21 = vec(S256, 21, "Y21")
	Y22 = vec(S256, 22, "Y22")
	Y23 = vec(S256, 23, "Y23")
	Y24 = vec(S256, 24, "Y24")
	Y25 = vec(S256, 25, "Y25")
	Y26 = vec(S256, 26, "Y26")
	Y27 = vec(S256, 27, "Y27")
	Y28 = vec(S256, 28, "Y28")
	Y29 = vec(S256, 29, "Y29")
	Y30 = vec(S256, 30, "Y30")
	Y31 = vec(S256, 31, "Y31")

	// 512-bit.
	Z0  = vec(S512, 0, "Z0")
	Z1  = vec(S512, 1, "Z1")
	Z2  = vec(S512, 2, "Z2")
	Z3  = vec(S512, 3, "Z3")
	Z4  = vec(S512, 4, "Z4")
	Z5  = vec(S512, 5, "Z5")
	Z6  = vec(S512, 6, "Z6")
	Z7  = vec(S512, 7, "Z7")
	Z8  = vec(S512, 8, "Z8")
	Z9  = vec(S512, 9, "Z9")
	Z10 = vec(S512, 10, "Z10")
	Z11 = vec(S512, 11, "Z11")
	Z12 = vec(S512, 12, "Z12")
	Z13 = vec(S512, 13, "Z13")
	Z14 = vec(S512, 14, "Z14")
	Z15 = vec(S512, 15, "Z15")
	Z16 = vec(S512, 16, "Z16")
	Z17 = vec(S512, 17, "Z17")
	Z18 = vec(S512, 18, "Z18")
	Z19 = vec(S512, 19, "Z19")
	Z20 = vec(S512, 20, "Z20")
	Z21 = vec(S512, 21, "Z21")
	Z22 = vec(S512, 22, "Z22")
	Z23 = vec(S512, 23, "Z23")
	Z24 = vec(S512, 24, "Z24")
	Z25 = vec(S512, 25, "Z25")
	Z26 = vec(S512, 26, "Z26")
	Z27 = vec(S512, 27, "Z27")
	Z28 = vec(S512, 28, "Z28")
	Z29 = vec(S512, 29, "Z29")
	Z30 = vec(S512, 30, "Z30")
	Z31 = vec(S512, 31, "Z31")
)

// OpmaskPhysical is a opmask physical register.
type OpmaskPhysical interface {
	Physical
}

type opmaskp struct {
	Physical
}

func newopmaskp(r Physical) OpmaskPhysical { return opmaskp{Physical: r} }

// OpmaskVirtual is a virtual opmask register.
type OpmaskVirtual interface {
	Virtual
}

type opmaskv struct {
	Virtual
}

func newopmaskv(v Virtual) OpmaskVirtual { return opmaskv{Virtual: v} }

func opmask(s Spec, id Index, name string, flags ...Info) OpmaskPhysical {
	r := newopmaskp(newregister(Opmask, s, id, name, flags...))
	Opmask.add(r)
	return r
}

// Opmask registers.
//
// Note that while K0 is a physical opmask register (it is a valid opmask source
// and destination operand), it cannot be used as an opmask predicate value
// because in that context K0 means "all true" or "no mask" regardless of the
// actual contents of the physical register. For that reason, K0 should never be
// assigned as a "general purpose" opmask register. However, it can be
// explicitly operated upon by name as non-predicate operand, for example to
// hold a constant or temporary value during calculations on other opmask
// registers.
var (
	K0 = opmask(S64, 0, "K0", Restricted)
	K1 = opmask(S64, 1, "K1")
	K2 = opmask(S64, 2, "K2")
	K3 = opmask(S64, 3, "K3")
	K4 = opmask(S64, 4, "K4")
	K5 = opmask(S64, 5, "K5")
	K6 = opmask(S64, 6, "K6")
	K7 = opmask(S64, 7, "K7")
)
