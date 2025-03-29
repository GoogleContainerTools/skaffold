package testhelpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v3"

	"github.com/buildpacks/pack/testhelpers/comparehelpers"

	"github.com/google/go-cmp/cmp"
)

type AssertionManager struct {
	testObject *testing.T
}

func NewAssertionManager(testObject *testing.T) AssertionManager {
	return AssertionManager{
		testObject: testObject,
	}
}

func (a AssertionManager) TrimmedEq(actual, expected string) {
	a.testObject.Helper()

	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")
	for lineIdx, line := range actualLines {
		actualLines[lineIdx] = strings.TrimRight(line, "\t \n")
	}

	for lineIdx, line := range expectedLines {
		expectedLines[lineIdx] = strings.TrimRight(line, "\t \n")
	}

	actualTrimmed := strings.Join(actualLines, "\n")
	expectedTrimmed := strings.Join(expectedLines, "\n")

	a.Equal(actualTrimmed, expectedTrimmed)
}

func (a AssertionManager) AssertTrimmedContains(actual, expected string) {
	a.testObject.Helper()

	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")
	for lineIdx, line := range actualLines {
		actualLines[lineIdx] = strings.TrimRight(line, "\t \n")
	}

	for lineIdx, line := range expectedLines {
		expectedLines[lineIdx] = strings.TrimRight(line, "\t \n")
	}

	actualTrimmed := strings.Join(actualLines, "\n")
	expectedTrimmed := strings.Join(expectedLines, "\n")

	a.Contains(actualTrimmed, expectedTrimmed)
}

func (a AssertionManager) Equal(actual, expected interface{}) {
	a.testObject.Helper()

	if diff := cmp.Diff(actual, expected); diff != "" {
		a.testObject.Fatal(diff)
	}
}

func (a AssertionManager) NotEqual(actual, expected interface{}) {
	a.testObject.Helper()

	if diff := cmp.Diff(actual, expected); diff == "" {
		a.testObject.Fatal(diff)
	}
}

func (a AssertionManager) Nil(actual interface{}) {
	a.testObject.Helper()

	if !isNil(actual) {
		a.testObject.Fatalf("expected nil: %v", actual)
	}
}

func (a AssertionManager) Succeeds(actual interface{}) {
	a.testObject.Helper()

	a.Nil(actual)
}

func (a AssertionManager) Fails(actual interface{}) {
	a.testObject.Helper()

	a.NotNil(actual)
}

func (a AssertionManager) NilWithMessage(actual interface{}, message string) {
	a.testObject.Helper()

	if !isNil(actual) {
		a.testObject.Fatalf("expected nil: %s: %s", actual, message)
	}
}

func (a AssertionManager) TrueWithMessage(actual bool, message string) {
	a.testObject.Helper()

	if !actual {
		a.testObject.Fatalf("expected true: %s", message)
	}
}

func (a AssertionManager) NotNil(actual interface{}) {
	a.testObject.Helper()

	if isNil(actual) {
		a.testObject.Fatal("expect not nil")
	}
}

func (a AssertionManager) Contains(actual, expected string) {
	a.testObject.Helper()

	if !strings.Contains(actual, expected) {
		a.testObject.Fatalf(
			"Expected '%s' to contain '%s'\n\nDiff:%s",
			actual,
			expected,
			cmp.Diff(expected, actual),
		)
	}
}

func (a AssertionManager) EqualJSON(actualJSON, expectedJSON string) {
	a.ContainsJSON(actualJSON, expectedJSON)
	a.ContainsJSON(expectedJSON, actualJSON)
}

func (a AssertionManager) ContainsJSON(actualJSON, expectedJSON string) {
	a.testObject.Helper()

	var actual interface{}
	err := json.Unmarshal([]byte(actualJSON), &actual)
	if err != nil {
		a.testObject.Fatalf(
			"Unable to unmarshal 'actualJSON': %q", err,
		)
	}

	var expected interface{}
	err = json.Unmarshal([]byte(expectedJSON), &expected)
	if err != nil {
		a.testObject.Fatalf(
			"Unable to unmarshal 'expectedJSON': %q", err,
		)
	}

	if !comparehelpers.DeepContains(actual, expected) {
		expectedJSONDebug, err := json.Marshal(expected)
		if err != nil {
			a.testObject.Fatalf("unable to render expected failure expectation: %q", err)
		}

		actualJSONDebug, err := json.Marshal(actual)
		if err != nil {
			a.testObject.Fatalf("unable to render actual failure expectation: %q", err)
		}

		var prettifiedExpected bytes.Buffer
		err = json.Indent(&prettifiedExpected, expectedJSONDebug, "", "  ")
		if err != nil {
			a.testObject.Fatal("failed to format expected TOML output as JSON")
		}

		var prettifiedActual bytes.Buffer
		err = json.Indent(&prettifiedActual, actualJSONDebug, "", "  ")
		if err != nil {
			a.testObject.Fatal("failed to format actual TOML output as JSON")
		}

		actualJSONDiffArray := strings.Split(prettifiedActual.String(), "\n")
		expectedJSONDiffArray := strings.Split(prettifiedExpected.String(), "\n")

		a.testObject.Fatalf(
			"Expected '%s' to contain '%s'\n\nJSON Diff:%s",
			prettifiedActual.String(),
			prettifiedExpected.String(),
			cmp.Diff(actualJSONDiffArray, expectedJSONDiffArray),
		)
	}
}

func (a AssertionManager) EqualYAML(actualYAML, expectedYAML string) {
	a.ContainsYAML(actualYAML, expectedYAML)
	a.ContainsYAML(expectedYAML, actualYAML)
}

func (a AssertionManager) ContainsYAML(actualYAML, expectedYAML string) {
	a.testObject.Helper()

	var actual interface{}
	err := yaml.Unmarshal([]byte(actualYAML), &actual)
	if err != nil {
		a.testObject.Fatalf(
			"Unable to unmarshal 'actualJSON': %q", err,
		)
	}

	var expected interface{}
	err = yaml.Unmarshal([]byte(expectedYAML), &expected)
	if err != nil {
		a.testObject.Fatalf(
			"Unable to unmarshal 'expectedYAML': %q", err,
		)
	}

	if !comparehelpers.DeepContains(actual, expected) {
		expectedYAMLDebug, err := yaml.Marshal(expected)
		if err != nil {
			a.testObject.Fatalf("unable to render expected failure expectation: %q", err)
		}

		actualYAMLDebug, err := yaml.Marshal(actual)
		if err != nil {
			a.testObject.Fatalf("unable to render actual failure expectation: %q", err)
		}

		actualYAMLDiffArray := strings.Split(string(actualYAMLDebug), "\n")
		expectedYAMLDiffArray := strings.Split(string(expectedYAMLDebug), "\n")

		a.testObject.Fatalf(
			"Expected '%s' to contain '%s'\n\nDiff:%s",
			string(actualYAMLDebug),
			string(expectedYAMLDebug),
			cmp.Diff(actualYAMLDiffArray, expectedYAMLDiffArray),
		)
	}
}

func (a AssertionManager) EqualTOML(actualTOML, expectedTOML string) {
	a.ContainsTOML(actualTOML, expectedTOML)
	a.ContainsTOML(expectedTOML, actualTOML)
}

func (a AssertionManager) ContainsTOML(actualTOML, expectedTOML string) {
	a.testObject.Helper()

	var actual interface{}
	err := toml.Unmarshal([]byte(actualTOML), &actual)
	if err != nil {
		a.testObject.Fatalf(
			"Unable to unmarshal 'actualTOML': %q", err,
		)
	}

	var expected interface{}
	err = toml.Unmarshal([]byte(expectedTOML), &expected)
	if err != nil {
		a.testObject.Fatalf(
			"Unable to unmarshal 'expectedTOML': %q", err,
		)
	}

	if !comparehelpers.DeepContains(actual, expected) {
		expectedJSONDebug, err := json.Marshal(expected)
		if err != nil {
			a.testObject.Fatalf("unable to render expected failure expectation: %q", err)
		}

		actualJSONDebug, err := json.Marshal(actual)
		if err != nil {
			a.testObject.Fatalf("unable to render actual failure expectation: %q", err)
		}

		var prettifiedExpected bytes.Buffer
		err = json.Indent(&prettifiedExpected, expectedJSONDebug, "", "  ")
		if err != nil {
			a.testObject.Fatal("failed to format expected TOML output as JSON")
		}

		var prettifiedActual bytes.Buffer
		err = json.Indent(&prettifiedActual, actualJSONDebug, "", "  ")
		if err != nil {
			a.testObject.Fatal("failed to format actual TOML output as JSON")
		}

		a.testObject.Fatalf(
			"Expected '%s' to contain '%s'\n\nJSON Diff:%s",
			prettifiedActual.String(),
			prettifiedExpected.String(),
			cmp.Diff(prettifiedActual.String(), prettifiedExpected.String()),
		)
	}
}

func (a AssertionManager) ContainsF(actual, expected string, formatArgs ...interface{}) {
	a.testObject.Helper()

	a.Contains(actual, fmt.Sprintf(expected, formatArgs...))
}

// ContainsWithMessage will fail if expected is not contained within actual, messageFormat will be printed as the
// failure message, with actual interpolated in the message
func (a AssertionManager) ContainsWithMessage(actual, expected, messageFormat string) {
	a.testObject.Helper()

	if !strings.Contains(actual, expected) {
		a.testObject.Fatalf(messageFormat, actual)
	}
}

func (a AssertionManager) ContainsAll(actual string, expected ...string) {
	a.testObject.Helper()

	for _, e := range expected {
		a.Contains(actual, e)
	}
}

func (a AssertionManager) Matches(actual string, pattern *regexp.Regexp) {
	a.testObject.Helper()

	if !pattern.MatchString(actual) {
		a.testObject.Fatalf("Expected '%s' to match regex '%s'", actual, pattern)
	}
}

func (a AssertionManager) NoMatches(actual string, pattern *regexp.Regexp) {
	a.testObject.Helper()

	if pattern.MatchString(actual) {
		a.testObject.Fatalf("Expected '%s' not to match regex '%s'", actual, pattern)
	}
}

func (a AssertionManager) MatchesAll(actual string, patterns ...*regexp.Regexp) {
	a.testObject.Helper()

	for _, pattern := range patterns {
		a.Matches(actual, pattern)
	}
}

func (a AssertionManager) NotContains(actual, expected string) {
	a.testObject.Helper()

	if strings.Contains(actual, expected) {
		a.testObject.Fatalf("Expected '%s' not to be in '%s'", expected, actual)
	}
}

// NotContainWithMessage will fail if expected is contained within actual, messageFormat will be printed as the failure
// message, with actual interpolated in the message
func (a AssertionManager) NotContainWithMessage(actual, expected, messageFormat string) {
	a.testObject.Helper()

	if strings.Contains(actual, expected) {
		a.testObject.Fatalf(messageFormat, actual)
	}
}

// Error checks that the provided value is an error (non-nil)
func (a AssertionManager) Error(actual error) {
	a.testObject.Helper()

	if actual == nil {
		a.testObject.Fatal("Expected an error but got nil")
	}
}

func (a AssertionManager) ErrorContains(actual error, expected string) {
	a.testObject.Helper()

	if actual == nil {
		a.testObject.Fatalf("Expected %q an error but got nil", expected)
	}

	a.Contains(actual.Error(), expected)
}

func (a AssertionManager) ErrorWithMessage(actual error, message string) {
	a.testObject.Helper()

	a.Error(actual)
	a.Equal(actual.Error(), message)
}

func (a AssertionManager) ErrorWithMessageF(actual error, format string, args ...interface{}) {
	a.testObject.Helper()

	a.ErrorWithMessage(actual, fmt.Sprintf(format, args...))
}
