package mint

import (
	"os"
	"testing"
)

// Mint (mint.Mint) is wrapper for *testing.T
// blending testing type to omit repeated `t`.
type Mint struct {
	t *testing.T
}

var (
	failToBe     = 0
	failType     = 1
	failIn       = 2
	failToMatch  = 3
	failExitCode = 4
	scolds       = map[int]string{
		failToBe:     "%s:%d\n\tExpected %sto be\t`%+v`\n\tBut actual\t`%+v`",
		failType:     "%s:%d\n\tExpected %stype\t`%+v`\n\tBut actual\t`%T`",
		failIn:       "%s:%d\n\tExpected %sis in\t`%v`\n\tbut it's not",
		failToMatch:  "%s:%d\n\tExpected %v to match\t`%s`\n\tBut actual\t`%+v`",
		failExitCode: "%s:%d\n\tExpected %sto exit with code `%d`\n\tBut actual\t`%d`",
	}
)
var (
	redB     = "\033[1;31m"
	reset    = "\033[0m"
	colorize = map[string]func(string) string{
		"red": func(v string) string {
			return redB + v + reset
		},
	}
)

// Blend provides (blended) *mint.Mint.
// You can save writing "t" repeatedly.
func Blend(t *testing.T) *Mint {
	return &Mint{
		t,
	}
}

// Expect provides "*Testee".
// The blended mint is merely a proxy to instantiate testee.
func (m *Mint) Expect(actual interface{}) *Testee {
	return expect(m.t, actual)
}

// Expect provides "*mint.Testee".
// It has assertion methods such as "ToBe".
func Expect(t *testing.T, actual interface{}) *Testee {
	return expect(t, actual)
}

func expect(t *testing.T, actual interface{}) *Testee {
	return &Testee{t: t, actual: actual, verbose: isVerbose(os.Args), result: MintResult{ok: true}}
}

// Require provides "*mint.Testee",
// which stops execution of goroutine when the assertion failed.
func Require(t *testing.T, actual interface{}) *Testee {
	return require(t, actual)
}

func require(t *testing.T, actual interface{}) *Testee {
	return &Testee{t: t, actual: actual, verbose: isVerbose(os.Args), required: true, result: MintResult{ok: true}}
}

func isVerbose(flags []string) bool {
	for _, f := range flags {
		if f == "-test.v=true" {
			return true
		}
	}
	return false
}
func judge(a, b interface{}, not, deeply bool) bool {
	comparer := getComparer(a, b, deeply)
	if not {
		return !comparer.Compare(a, b)
	}
	return comparer.Compare(a, b)
}
