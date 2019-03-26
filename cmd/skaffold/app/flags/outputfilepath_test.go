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
)

func TestNewOutputFilepathType(t *testing.T) {
	flag := NewOutputFilepath("test.out", "test output file")
	expectedFlag := OutputFilepath{
		filepathFlag: filepathFlag{
			path: "test.out",
		},
		usage: "test output file",
	}
	if *flag != expectedFlag {
		t.Errorf("expected %s, actual %s", &expectedFlag, flag)
	}
}

func TestOutputFileFlagSet(t *testing.T) {
	flag := NewOutputFilepath("", "")
	if err := flag.Set("test.out"); err != nil {
		t.Errorf("Error setting flag value: %s", err)
	}
}
