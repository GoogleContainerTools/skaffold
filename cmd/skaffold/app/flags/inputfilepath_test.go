/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flags

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewInputFile(t *testing.T) {
	flag := NewInputFilepath("test.in", "test input file")
	expectedFlag := InputFilepath{
		filepathFlag: filepathFlag{
			path:        "test.in",
			shouldExist: true,
		},
		usage: "test input file",
	}
	if *flag != expectedFlag {
		t.Errorf("expected %s, actual %s", &expectedFlag, flag)
	}
}

func TestInputFileFlagSet(t *testing.T) {
	dir, cleanUp := testutil.NewTempDir(t)
	defer cleanUp()
	filename := "exists.in"
	dir.Write(filename, "some input")

	var tests = []struct {
		description string
		setValue    string
		shouldErr   bool
	}{
		{
			description: "set should not error when file is present and new flag set",
			setValue:    dir.Path(filename),
		},
		{
			description: "set should error when file is not present with original flag value",
			setValue:    "does_not_exist.in",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		flag := NewInputFilepath("", "")
		err := flag.Set(test.setValue)
		expectedFlag := flag
		if !test.shouldErr {
			expectedFlag = NewInputFilepath(test.setValue, "")
		}
		testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, expectedFlag.String(), flag.String())
	}
}

func TestInputFilepathType(t *testing.T) {
	flag := NewInputFilepath("test.in", "")
	expectedFlagType := "*flags.InputFilepath"
	if flag.Type() != expectedFlagType {
		t.Errorf("Flag returned wrong type. Expected %s, Actual %s", expectedFlagType, flag.Type())
	}
}
