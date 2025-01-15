package gotypes

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"strconv"

	"github.com/mmcloughlin/avo/operand"
)

// Signature represents a Go function signature.
type Signature struct {
	pkg     *types.Package
	sig     *types.Signature
	params  *Tuple
	results *Tuple
}

// NewSignature constructs a Signature.
func NewSignature(pkg *types.Package, sig *types.Signature) *Signature {
	s := &Signature{
		pkg: pkg,
		sig: sig,
	}
	s.init()
	return s
}

// NewSignatureVoid builds the void signature "func()".
func NewSignatureVoid() *Signature {
	return NewSignature(nil, types.NewSignatureType(nil, nil, nil, nil, nil, false))
}

// LookupSignature returns the signature of the named function in the provided package.
func LookupSignature(pkg *types.Package, name string) (*Signature, error) {
	scope := pkg.Scope()
	obj := scope.Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("could not find function \"%s\"", name)
	}
	s, ok := obj.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("object \"%s\" does not have signature type", name)
	}
	return NewSignature(pkg, s), nil
}

// ParseSignature builds a Signature by parsing a Go function type expression.
// The function type must reference builtin types only; see
// ParseSignatureInPackage if custom types are required.
func ParseSignature(expr string) (*Signature, error) {
	return ParseSignatureInPackage(nil, expr)
}

// ParseSignatureInPackage builds a Signature by parsing a Go function type
// expression. The expression may reference types in the provided package.
func ParseSignatureInPackage(pkg *types.Package, expr string) (*Signature, error) {
	tv, err := types.Eval(token.NewFileSet(), pkg, token.NoPos, expr)
	if err != nil {
		return nil, err
	}
	if tv.Value != nil {
		return nil, errors.New("signature expression should have nil value")
	}
	s, ok := tv.Type.(*types.Signature)
	if !ok {
		return nil, errors.New("provided type is not a function signature")
	}
	return NewSignature(pkg, s), nil
}

// Params returns the function signature argument types.
func (s *Signature) Params() *Tuple { return s.params }

// Results returns the function return types.
func (s *Signature) Results() *Tuple { return s.results }

// Bytes returns the total size of the function arguments and return values.
func (s *Signature) Bytes() int { return s.Params().Bytes() + s.Results().Bytes() }

// String writes Signature as a string. This does not include the "func" keyword.
func (s *Signature) String() string {
	var buf bytes.Buffer
	types.WriteSignature(&buf, s.sig, func(pkg *types.Package) string {
		if pkg == s.pkg {
			return ""
		}
		return pkg.Name()
	})
	return buf.String()
}

func (s *Signature) init() {
	p := s.sig.Params()
	r := s.sig.Results()

	// Compute parameter offsets. Note that if the function has results,
	// additional padding up to max align is inserted between parameters and
	// results.
	vs := tuplevars(p)
	vs = append(vs, types.NewParam(token.NoPos, nil, "sentinel", types.Typ[types.Uint64]))
	paramsoffsets := Sizes.Offsetsof(vs)
	paramssize := paramsoffsets[p.Len()]
	if r.Len() == 0 {
		paramssize = structsize(vs[:p.Len()])
	}
	s.params = newTuple(p, paramsoffsets, paramssize, "arg")

	// Result offsets.
	vs = tuplevars(r)
	resultsoffsets := Sizes.Offsetsof(vs)
	resultssize := structsize(vs)
	for i := range resultsoffsets {
		resultsoffsets[i] += paramssize
	}
	s.results = newTuple(r, resultsoffsets, resultssize, "ret")
}

// Tuple represents a tuple of variables, such as function arguments or results.
type Tuple struct {
	components []Component
	byname     map[string]Component
	size       int
}

func newTuple(t *types.Tuple, offsets []int64, size int64, defaultprefix string) *Tuple {
	tuple := &Tuple{
		byname: map[string]Component{},
		size:   int(size),
	}
	for i := 0; i < t.Len(); i++ {
		v := t.At(i)
		name := v.Name()
		if name == "" {
			name = defaultprefix
			if i > 0 {
				name += strconv.Itoa(i)
			}
		}
		addr := operand.NewParamAddr(name, int(offsets[i]))
		c := NewComponent(v.Type(), addr)
		tuple.components = append(tuple.components, c)
		if v.Name() != "" {
			tuple.byname[v.Name()] = c
		}
	}
	return tuple
}

// Lookup returns the variable with the given name.
func (t *Tuple) Lookup(name string) Component {
	e := t.byname[name]
	if e == nil {
		return errorf("unknown variable \"%s\"", name)
	}
	return e
}

// At returns the variable at index i.
func (t *Tuple) At(i int) Component {
	if i >= len(t.components) {
		return errorf("index out of range")
	}
	return t.components[i]
}

// Bytes returns the size of the Tuple. This may include additional padding.
func (t *Tuple) Bytes() int { return t.size }

func tuplevars(t *types.Tuple) []*types.Var {
	vs := make([]*types.Var, t.Len())
	for i := 0; i < t.Len(); i++ {
		vs[i] = t.At(i)
	}
	return vs
}

// structsize computes the size of a struct containing the given variables as
// fields. It would be equivalent to calculating the size of types.NewStruct(vs,
// nil), apart from the fact that NewStruct panics if multiple fields have the
// same name, and this happens for example if the variables represent return
// types from a function.
func structsize(vs []*types.Var) int64 {
	n := len(vs)
	if n == 0 {
		return 0
	}
	offsets := Sizes.Offsetsof(vs)
	return offsets[n-1] + Sizes.Sizeof(vs[n-1].Type())
}
