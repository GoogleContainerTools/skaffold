package termui

type Symbol int

func (s Symbol) String() string {
	switch s {
	case NoBranchSymbol:
		return "    "
	case BranchSymbol:
		return "│   "
	case MiddleBranchSymbol:
		return "├── "
	case LastBranchSymbol:
		return "└── "
	default:
		return ""
	}
}

const (
	NoBranchSymbol Symbol = iota
	BranchSymbol
	MiddleBranchSymbol
	LastBranchSymbol
)

type Branches []Symbol

func (b Branches) Add(s Symbol) Branches {
	bCopy := make(Branches, len(b))
	copy(bCopy, b)
	return append(bCopy, s)
}

func (b Branches) String() string {
	var s string
	for _, branch := range b[:len(b)-1] {
		s += branch.String()
	}
	return s
}
