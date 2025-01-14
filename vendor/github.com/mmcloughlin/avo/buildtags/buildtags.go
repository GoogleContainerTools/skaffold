// Package buildtags provides types for representing and manipulating build constraints.
//
// In Go, build constraints are represented as comments in source code together with file naming conventions. For example
//
//	// +build linux,386 darwin,!cgo
//	// +build !purego
//
// Any terms provided in the filename can be thought of as an implicit extra
// constraint comment line. Collectively, these are referred to as
// “constraints”. Each line is a “constraint”. Within each constraint the
// space-separated terms are “options”, and within that the comma-separated
// items are “terms” which may be negated with at most one exclaimation mark.
//
// These represent a boolean formulae. The constraints are evaluated as the AND
// of constraint lines; a constraint is evaluated as the OR of its options and
// an option is evaluated as the AND of its terms. Overall build constraints are
// a boolean formula that is an AND of ORs of ANDs.
//
// This level of complexity is rarely used in Go programs. Therefore this
// package aims to provide access to all these layers of nesting if required,
// but make it easy to forget about for basic use cases too.
package buildtags

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Reference: https://github.com/golang/go/blob/204a8f55dc2e0ac8d27a781dab0da609b98560da/src/go/build/doc.go#L73-L92
//
//	// A build constraint is evaluated as the OR of space-separated options;
//	// each option evaluates as the AND of its comma-separated terms;
//	// and each term is an alphanumeric word or, preceded by !, its negation.
//	// That is, the build constraint:
//	//
//	//	// +build linux,386 darwin,!cgo
//	//
//	// corresponds to the boolean formula:
//	//
//	//	(linux AND 386) OR (darwin AND (NOT cgo))
//	//
//	// A file may have multiple build constraints. The overall constraint is the AND
//	// of the individual constraints. That is, the build constraints:
//	//
//	//	// +build linux darwin
//	//	// +build 386
//	//
//	// corresponds to the boolean formula:
//	//
//	//	(linux OR darwin) AND 386
//

// Interface represents a build constraint.
type Interface interface {
	ConstraintsConvertable
	fmt.GoStringer
	Evaluate(v map[string]bool) bool
	Validate() error
}

// ConstraintsConvertable can be converted to a Constraints object.
type ConstraintsConvertable interface {
	ToConstraints() Constraints
}

// ConstraintConvertable can be converted to a Constraint.
type ConstraintConvertable interface {
	ToConstraint() Constraint
}

// OptionConvertable can be converted to an Option.
type OptionConvertable interface {
	ToOption() Option
}

// Constraints represents the AND of a list of Constraint lines.
type Constraints []Constraint

// And builds Constraints that will be true if all of its constraints are true.
func And(cs ...ConstraintConvertable) Constraints {
	constraints := Constraints{}
	for _, c := range cs {
		constraints = append(constraints, c.ToConstraint())
	}
	return constraints
}

// ToConstraints returns cs.
func (cs Constraints) ToConstraints() Constraints { return cs }

// Validate validates the constraints set.
func (cs Constraints) Validate() error {
	for _, c := range cs {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Evaluate the boolean formula represented by cs under the given assignment of
// tag values. This is the AND of the values of the constituent Constraints.
func (cs Constraints) Evaluate(v map[string]bool) bool {
	r := true
	for _, c := range cs {
		r = r && c.Evaluate(v)
	}
	return r
}

// GoString represents Constraints as +build comment lines.
func (cs Constraints) GoString() string {
	s := ""
	for _, c := range cs {
		s += c.GoString()
	}
	return s
}

// Constraint represents the OR of a list of Options.
type Constraint []Option

// Any builds a Constraint that will be true if any of its options are true.
func Any(opts ...OptionConvertable) Constraint {
	c := Constraint{}
	for _, opt := range opts {
		c = append(c, opt.ToOption())
	}
	return c
}

// ParseConstraint parses a space-separated list of options.
func ParseConstraint(expr string) (Constraint, error) {
	c := Constraint{}
	for _, field := range strings.Fields(expr) {
		opt, err := ParseOption(field)
		if err != nil {
			return c, err
		}
		c = append(c, opt)
	}
	return c, nil
}

// ToConstraints returns the list of constraints containing just c.
func (c Constraint) ToConstraints() Constraints { return Constraints{c} }

// ToConstraint returns c.
func (c Constraint) ToConstraint() Constraint { return c }

// Validate validates the constraint.
func (c Constraint) Validate() error {
	for _, o := range c {
		if err := o.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Evaluate the boolean formula represented by c under the given assignment of
// tag values. This is the OR of the values of the constituent Options.
func (c Constraint) Evaluate(v map[string]bool) bool {
	r := false
	for _, o := range c {
		r = r || o.Evaluate(v)
	}
	return r
}

// GoString represents the Constraint as one +build comment line.
func (c Constraint) GoString() string {
	s := "// +build"
	for _, o := range c {
		s += " " + o.GoString()
	}
	return s + "\n"
}

// Option represents the AND of a list of Terms.
type Option []Term

// Opt builds an Option from the list of Terms.
func Opt(terms ...Term) Option {
	return Option(terms)
}

// ParseOption parses a comma-separated list of terms.
func ParseOption(expr string) (Option, error) {
	opt := Option{}
	for _, t := range strings.Split(expr, ",") {
		opt = append(opt, Term(t))
	}
	return opt, opt.Validate()
}

// ToConstraints returns Constraints containing just this option.
func (o Option) ToConstraints() Constraints { return o.ToConstraint().ToConstraints() }

// ToConstraint returns a Constraint containing just this option.
func (o Option) ToConstraint() Constraint { return Constraint{o} }

// ToOption returns o.
func (o Option) ToOption() Option { return o }

// Validate validates o.
func (o Option) Validate() error {
	for _, t := range o {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("invalid term %q: %w", t, err)
		}
	}
	return nil
}

// Evaluate the boolean formula represented by o under the given assignment of
// tag values. This is the AND of the values of the constituent Terms.
func (o Option) Evaluate(v map[string]bool) bool {
	r := true
	for _, t := range o {
		r = r && t.Evaluate(v)
	}
	return r
}

// GoString represents the Option as a comma-separated list of terms.
func (o Option) GoString() string {
	var ts []string
	for _, t := range o {
		ts = append(ts, t.GoString())
	}
	return strings.Join(ts, ",")
}

// Term is an atomic term in a build constraint: an identifier or its negation.
type Term string

// Not returns a term for the negation of ident.
func Not(ident string) Term {
	return Term("!" + ident)
}

// ToConstraints returns Constraints containing just this term.
func (t Term) ToConstraints() Constraints { return t.ToOption().ToConstraints() }

// ToConstraint returns a Constraint containing just this term.
func (t Term) ToConstraint() Constraint { return t.ToOption().ToConstraint() }

// ToOption returns an Option containing just this term.
func (t Term) ToOption() Option { return Option{t} }

// IsNegated reports whether t is the negation of an identifier.
func (t Term) IsNegated() bool { return strings.HasPrefix(string(t), "!") }

// Name returns the identifier for this term.
func (t Term) Name() string {
	return strings.TrimPrefix(string(t), "!")
}

// Validate the term.
func (t Term) Validate() error {
	// Reference: https://github.com/golang/go/blob/204a8f55dc2e0ac8d27a781dab0da609b98560da/src/cmd/go/internal/imports/build.go#L110-L112
	//
	//		if strings.HasPrefix(name, "!!") { // bad syntax, reject always
	//			return false
	//		}
	//
	if strings.HasPrefix(string(t), "!!") {
		return errors.New("at most one '!' allowed")
	}

	if len(t.Name()) == 0 {
		return errors.New("empty tag name")
	}

	// Reference: https://github.com/golang/go/blob/204a8f55dc2e0ac8d27a781dab0da609b98560da/src/cmd/go/internal/imports/build.go#L121-L127
	//
	//		// Tags must be letters, digits, underscores or dots.
	//		// Unlike in Go identifiers, all digits are fine (e.g., "386").
	//		for _, c := range name {
	//			if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '.' {
	//				return false
	//			}
	//		}
	//
	for _, c := range t.Name() {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '.' {
			return fmt.Errorf("character '%c' disallowed in tags", c)
		}
	}

	return nil
}

// Evaluate the term under the given set of identifier values.
func (t Term) Evaluate(v map[string]bool) bool {
	return (t.Validate() == nil) && (v[t.Name()] == !t.IsNegated())
}

// GoString returns t.
func (t Term) GoString() string { return string(t) }

// SetTags builds a set where the given list of identifiers are true.
func SetTags(idents ...string) map[string]bool {
	v := map[string]bool{}
	for _, ident := range idents {
		v[ident] = true
	}
	return v
}
