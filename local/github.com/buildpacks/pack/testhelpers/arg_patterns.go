package testhelpers

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/buildpacks/pack/internal/stringset"
)

func AssertIncludeAllExpectedPatterns(t *testing.T, receivedArgs []string, expectedPatterns ...[]string) {
	t.Helper()

	missingPatterns := [][]string{}

	for _, expectedPattern := range expectedPatterns {
		if !patternExists(expectedPattern, receivedArgs) {
			missingPatterns = append(missingPatterns, expectedPattern)
		}
	}

	assertSliceEmpty(t,
		missingPatterns,
		"Expected the patterns %s to exist in [%s]",
		missingPatterns,
		strings.Join(receivedArgs, " "),
	)
}

func patternExists(expectedPattern []string, receivedArgs []string) bool {
	_, missing, _ := stringset.Compare(receivedArgs, expectedPattern)
	if len(missing) > 0 {
		return false
	}

	if len(expectedPattern) == 1 {
		return true
	}

	for _, loc := range matchLocations(expectedPattern[0], receivedArgs) {
		finalElementLoc := intMin(loc+len(expectedPattern), len(receivedArgs))

		receivedSubSlice := receivedArgs[loc:finalElementLoc]

		if reflect.DeepEqual(receivedSubSlice, expectedPattern) {
			return true
		}
	}

	return false
}

func matchLocations(expectedArg string, receivedArgs []string) []int {
	indices := []int{}

	for i, receivedArg := range receivedArgs {
		if receivedArg == expectedArg {
			indices = append(indices, i)
		}
	}

	return indices
}

func assertSliceEmpty(t *testing.T, actual [][]string, msg string, msgArgs ...interface{}) {
	t.Helper()

	empty, err := sliceEmpty(actual)

	if err != nil {
		t.Fatalf("assertSliceNotEmpty error: %s", err.Error())
	}

	if !empty {
		t.Fatalf(msg, msgArgs...)
	}
}

func sliceEmpty(slice [][]string) (bool, error) {
	switch reflect.TypeOf(slice).Kind() {
	case reflect.Slice:
		return reflect.ValueOf(slice).Len() == 0, nil
	default:
		return true, fmt.Errorf("invoked with non slice actual: %v", slice)
	}
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
