package please

import (
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"

	"github.com/GoogleContainerTools/skaffold/testutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func TestDepToPath(t *testing.T) {
	var tests = []struct {
		description string
		dep         string
		expected    string
	}{
		{
			description: "top level target",
			dep:         "//:image",
			expected:    "",
		},
		{
			description: "regular target",
			dep:         "//example:image",
			expected:    "example",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			path := depToPath(test.dep)

			if path != test.expected {
				t.Errorf("Expected %s. Got %s", test.expected, path)
			}
		})
	}
}

func TestContainsPath(t *testing.T) {
	var deps = []string{"dep1", "dep2"}
	var tests = []struct {
		description string
		dep         string
		expected    bool
	}{
		{
			description: "check if dep1 is found in deps",
			dep:         "dep1",
			expected:    true,
		},
		{
			description: "check if dep5 is found in deps",
			dep:         "dep5",
			expected:    false,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.expected != containsPath(deps, test.dep) {
				t.Errorf("Expected %t. Got %t", test.expected, !test.expected)
			}
		})
	}

}

func TestGetInputFiles(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunOut(
		"please query input //test:image",
		"test/Dockerfile\n\ntest/example.py\n",
	)

	deps, err := getInputFiles(context.Background(), ".", &latest.PleaseArtifact{
		BuildTarget: "//test:image",
	})
	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"test/Dockerfile", "test/example.py"}, deps)
}

func TestGetDepTargets(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunOut(
		"please query deps -p -u //example:image",
		`//example:image
//example:base_fqn
//example:example

//example:base
//example:image_fqn

`,
	)

	deps, err := getDepTargets(context.Background(), ".", &latest.PleaseArtifact{
		BuildTarget: "//example:image",
	})
	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"example/BUILD"}, deps)
}

func TestGetDependencies(t *testing.T) {
	defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
	util.DefaultExecCommand = testutil.NewFakeCmd(t).WithRunOut(
		"please query input //example:image",
		`example/example.py
example/Dockerfile
example/Dockerfile-base

`,
	).WithRunOut(
		"please query deps -p -u //example:image",
		`//example:image
//example:base_fqn
//example:example
//example:base
//example:image_fqn

`,
	)
	deps, err := GetDependencies(context.Background(), ".", &latest.PleaseArtifact{
		BuildTarget: "//example:image",
	})
	testutil.CheckErrorAndDeepEqual(t, false, err, []string{"example/example.py", "example/Dockerfile", "example/Dockerfile-base", "example/BUILD"}, deps)
}
