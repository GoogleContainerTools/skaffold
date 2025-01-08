package mint

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"testing"

	"github.com/otiai10/mint/mquery"
)

// Testee is holder of interfaces which user want to assert
// and also has its result.
type Testee struct {
	t        *testing.T
	actual   interface{}
	expected interface{}
	dry      bool
	not      bool
	deeply   bool
	result   MintResult
	required bool
	verbose  bool

	// origin string // Only used when querying
}

// Query queries the actual value with given query string.
func (testee *Testee) Query(query string) *Testee {
	// testee.origin = fmt.Sprintf("%T", testee.actual)
	testee.actual = mquery.Query(testee.actual, query)
	return testee
}

// ToBe can assert the testee to equal the parameter of this func.
// OS will exit with code 1, when the assertion fail.
// If you don't want to exit, see "Dry()".
func (testee *Testee) ToBe(expected interface{}) MintResult {
	if judge(testee.actual, expected, testee.not, testee.deeply) {
		return testee.result
	}
	testee.expected = expected
	return testee.failed(failToBe)
}

// Match can assert the testee to match with specified regular expression.
// It uses `regexp.MustCompile`, it's due to caller to make sure it's valid regexp.
// OS will exit with code 1, when the assertion fail.
// If you don't want to exit, see "Dry()".
func (testee *Testee) Match(expression string) MintResult {
	exp := regexp.MustCompile(expression)
	matched := exp.MatchString(fmt.Sprintf("%v", testee.actual))
	if judge(matched, true, testee.not, testee.deeply) {
		return testee.result
	}
	testee.expected = expression
	return testee.failed(failToMatch)
}

// In can assert the testee is in given array.
func (testee *Testee) In(expecteds ...interface{}) MintResult {
	for _, expected := range expecteds {
		if judge(testee.actual, expected, testee.not, testee.deeply) {
			return testee.result
		}
	}
	testee.expected = expecteds
	return testee.failed(failIn)
}

// TypeOf can assert the type of testee to equal the parameter of this func.
// OS will exit with code 1, when the assertion fail.
// If you don't want to exit, see "Dry()".
func (testee *Testee) TypeOf(typeName string) MintResult {
	if judge(reflect.TypeOf(testee.actual).String(), typeName, testee.not, testee.deeply) {
		return testee.result
	}
	testee.expected = typeName
	return testee.failed(failType)
}

// Not makes following assertion conversed.
func (testee *Testee) Not() *Testee {
	testee.not = true
	return testee
}

// Dry makes the testee NOT to call "Fail()".
// Use this if you want to fail test in a purpose.
func (testee *Testee) Dry() *Testee {
	testee.dry = true
	return testee
}

// Deeply makes following assertions use `reflect.DeepEqual`.
// You had better use this to compare reference type objects.
func (testee *Testee) Deeply() *Testee {
	testee.deeply = true
	return testee
}

func (testee *Testee) failed(failure int) MintResult {
	message := testee.toText(failure)
	testee.result.ok = false
	testee.result.message = message
	if !testee.dry {
		fmt.Println(colorize["red"](message))
		if testee.required {
			testee.t.FailNow()
		} else {
			testee.t.Fail()
		}
	}
	return testee.result
}

func (testee *Testee) toText(fail int) string {
	not := ""
	if testee.not {
		not = "NOT "
	}
	_, file, line, _ := runtime.Caller(3)
	// if testee.origin != "" {
	// 	testee.origin = fmt.Sprintf("(queried from %s)", testee.origin)
	// }
	return fmt.Sprintf(
		scolds[fail],
		filepath.Base(file), line,
		not,
		testee.expected,
		testee.actual,
	)
}

// Log only output if -v flag is given.
// This is because the standard "t.Testing.Log" method decorates
// its caller: runtime.Caller(3) automatically.
func (testee *Testee) Log(args ...interface{}) {
	if !testee.verbose {
		return
	}
	fmt.Print(args...)
}
