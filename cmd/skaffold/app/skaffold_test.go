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

package app

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestMainHelp(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		// --interactive=false removes the update check and survey prompt.
		t.Override(&os.Args, []string{"skaffold", "help", "--interactive=false"})

		var (
			output    bytes.Buffer
			errOutput bytes.Buffer
		)
		err := Run(&output, &errOutput)

		t.CheckNoError(err)
		t.CheckContains("End-to-end Pipelines", output.String())
		t.CheckContains("Getting Started With a New Project", output.String())
		t.CheckEmpty(errOutput.String())
	})
}

func TestMainUnknownCommand(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		// --interactive=false removes the update check and survey prompt.
		t.Override(&os.Args, []string{"skaffold", "unknown", "--interactive=false"})

		err := Run(ioutil.Discard, ioutil.Discard)

		t.CheckError(true, err)
	})
}

func TestSkaffoldCmdline_MainHelp(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		var (
			output    bytes.Buffer
			errOutput bytes.Buffer
		)

		t.SetEnvs(map[string]string{"SKAFFOLD_CMDLINE": "help"})
		t.Override(&os.Args, []string{"skaffold"})

		err := Run(&output, &errOutput)

		t.CheckNoError(err)
		t.CheckContains("End-to-end Pipelines", output.String())
		t.CheckContains("Getting Started With a New Project", output.String())
		t.CheckEmpty(errOutput.String())
	})
}

func TestSkaffoldCmdline_MainUnknownCommand(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&os.Args, []string{"skaffold"})
		t.SetEnvs(map[string]string{"SKAFFOLD_CMDLINE": "unknown"})

		err := Run(ioutil.Discard, ioutil.Discard)

		t.CheckError(true, err)
	})
}

func TestMain_InvalidUsageExitCode(t *testing.T) {
	testutil.Run(t, "unknown command", func(t *testutil.T) {
		// --interactive=false removes the update check and survey prompt.
		t.Override(&os.Args, []string{"skaffold", "unknown", "--interactive=false"})
		err := Run(ioutil.Discard, ioutil.Discard)
		t.CheckErrorAndExitCode(127, err)
	})
	testutil.Run(t, "unknown flag", func(t *testutil.T) {
		// --interactive=false removes the update check and survey prompt.
		t.Override(&os.Args, []string{"skaffold", "--help2", "--interactive=false"})
		err := Run(ioutil.Discard, ioutil.Discard)
		t.CheckErrorAndExitCode(127, err)
	})
	testutil.Run(t, "exactargs error", func(t *testutil.T) {
		// --interactive=false removes the update check and survey prompt.
		t.Override(&os.Args, []string{"skaffold", "config", "set", "a", "b", "c"})
		err := Run(ioutil.Discard, ioutil.Discard)
		t.CheckErrorAndExitCode(127, err)
	})
}
