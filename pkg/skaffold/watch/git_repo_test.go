package watch

import (
	"fmt"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGetCurrentCommit(t *testing.T) {
	var tests = []struct {
		description string
		url         string
		command     util.Command
		expected    string
		shouldErr   bool
	}{
		{
			description: "correct",
			url:         "https://test.git",
			expected:    "a143d3841fa9d981ab242bb2c1b09f27e5da05bc",
			command: testutil.NewFakeCmdOut("git ls-remote https://test.git refs/heads/master", "a143d3841fa9d981ab242bb2c1b09f27e5da05bc	refs/heads/master\n", nil),
		},
		{
			description: "error",
			url:         "https://test.git",
			shouldErr:   true,
			command: testutil.NewFakeCmdOut("git ls-remote https://test.git refs/heads/master", "a143d3841fa9d981ab242bb2c1b09f27e5da05bc	refs/heads/master\n", fmt.Errorf("")),
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			actual, err := getCurrentCommit(test.url)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, actual)
		})
	}
}

func TestGetHeadRef(t *testing.T) {
	var tests = []struct {
		description string
		url         string
		command     util.Command
		expected    string
		shouldErr   bool
	}{
		{
			description: "correct",
			expected:    "a143d3841fa9d981ab242bb2c1b09f27e5da05bc",
			command:     testutil.NewFakeCmdOut("git rev-parse master", "a143d3841fa9d981ab242bb2c1b09f27e5da05bc", nil),
		},
		{
			description: "error",
			shouldErr:   true,
			command:     testutil.NewFakeCmdOut("git rev-parse master", "a143d3841fa9d981ab242bb2c1b09f27e5da05bc", fmt.Errorf("")),
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			actual, err := getHeadRef()
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, actual)
		})
	}
}

const diffOutput = `cmd/skaffold/app/cmd/cmd.go
pkg/skaffold/config/options.go
pkg/skaffold/runner/runner.go
pkg/skaffold/watch/artifacts.go
pkg/skaffold/watch/watch.go
`

var expectedDiffOutput = []string{
	"cmd/skaffold/app/cmd/cmd.go",
	"pkg/skaffold/config/options.go",
	"pkg/skaffold/runner/runner.go",
	"pkg/skaffold/watch/artifacts.go",
	"pkg/skaffold/watch/watch.go",
}

func TestComputeGitDiff(t *testing.T) {
	var tests = []struct {
		description string
		srcRef      string
		targetRef   string
		url         string
		command     util.Command
		expected    []string
		shouldErr   bool
	}{
		{
			description: "correct",
			srcRef:      "a143d3841fa9d981ab242bb2c1b09f27e5da05bc",
			targetRef:   "d779f0e0558998943c9bd45970513e7408fa9c64",
			expected:    expectedDiffOutput,
			command:     testutil.NewFakeCmdOut("git diff --name-only a143d3841fa9d981ab242bb2c1b09f27e5da05bc d779f0e0558998943c9bd45970513e7408fa9c64", diffOutput, nil),
		},
		{
			description: "error",
			srcRef:      "a143d3841fa9d981ab242bb2c1b09f27e5da05bc",
			targetRef:   "d779f0e0558998943c9bd45970513e7408fa9c64",
			shouldErr:   true,
			command:     testutil.NewFakeCmdOut("git diff --name-only a143d3841fa9d981ab242bb2c1b09f27e5da05bc d779f0e0558998943c9bd45970513e7408fa9c64", diffOutput, fmt.Errorf(""))},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}

			actual, err := computeGitDiff(test.srcRef, test.targetRef)
			fmt.Println(actual)
			sort.Strings(actual)
			testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expected, actual)
		})
	}
}
