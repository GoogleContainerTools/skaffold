package cli

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func TestRequiresNoArgs(t *testing.T) {
	testCases := []testCase{
		{
			validateFunc:  NoArgs,
			expectedError: "no error",
		},
		{
			args:          []string{"foo"},
			validateFunc:  NoArgs,
			expectedError: "accepts no arguments.",
		},
	}
	runTestCases(t, testCases)
}

func TestRequiresMinArgs(t *testing.T) {
	testCases := []testCase{
		{
			validateFunc:  RequiresMinArgs(0),
			expectedError: "no error",
		},
		{
			validateFunc:  RequiresMinArgs(1),
			expectedError: "at least 1 argument.",
		},
		{
			args:          []string{"foo"},
			validateFunc:  RequiresMinArgs(2),
			expectedError: "at least 2 arguments.",
		},
	}
	runTestCases(t, testCases)
}

func TestRequiresMaxArgs(t *testing.T) {
	testCases := []testCase{
		{
			validateFunc:  RequiresMaxArgs(0),
			expectedError: "no error",
		},
		{
			args:          []string{"foo", "bar"},
			validateFunc:  RequiresMaxArgs(1),
			expectedError: "at most 1 argument.",
		},
		{
			args:          []string{"foo", "bar", "baz"},
			validateFunc:  RequiresMaxArgs(2),
			expectedError: "at most 2 arguments.",
		},
	}
	runTestCases(t, testCases)
}

func TestRequiresRangeArgs(t *testing.T) {
	testCases := []testCase{
		{
			validateFunc:  RequiresRangeArgs(0, 0),
			expectedError: "no error",
		},
		{
			validateFunc:  RequiresRangeArgs(0, 1),
			expectedError: "no error",
		},
		{
			args:          []string{"foo", "bar"},
			validateFunc:  RequiresRangeArgs(0, 1),
			expectedError: "at most 1 argument.",
		},
		{
			args:          []string{"foo", "bar", "baz"},
			validateFunc:  RequiresRangeArgs(0, 2),
			expectedError: "at most 2 arguments.",
		},
		{
			validateFunc:  RequiresRangeArgs(1, 2),
			expectedError: "at least 1 ",
		},
	}
	runTestCases(t, testCases)
}

func TestExactArgs(t *testing.T) {
	testCases := []testCase{
		{
			validateFunc:  ExactArgs(0),
			expectedError: "no error",
		},
		{
			validateFunc:  ExactArgs(1),
			expectedError: "exactly 1 argument.",
		},
		{
			validateFunc:  ExactArgs(2),
			expectedError: "exactly 2 arguments.",
		},
	}
	runTestCases(t, testCases)
}

type testCase struct {
	args          []string
	validateFunc  cobra.PositionalArgs
	expectedError string
}

func runTestCases(t *testing.T, testCases []testCase) {
	for _, tc := range testCases {
		cmd := newDummyCommand(tc.validateFunc)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)

		err := cmd.Execute()
		assert.ErrorContains(t, err, tc.expectedError)
	}
}

func newDummyCommand(validationFunc cobra.PositionalArgs) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "dummy",
		Args: validationFunc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("no error")
		},
	}
	return cmd
}
