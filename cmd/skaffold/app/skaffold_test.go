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
	"io"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
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

		err := Run(io.Discard, io.Discard)

		t.CheckError(true, err)
	})
}

func TestSkaffoldCmdline_MainHelp(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		var (
			output    bytes.Buffer
			errOutput bytes.Buffer
		)

		t.Setenv("SKAFFOLD_CMDLINE", "help")
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
		t.Setenv("SKAFFOLD_CMDLINE", "unknown")

		err := Run(io.Discard, io.Discard)

		t.CheckError(true, err)
	})
}

func TestMain_InvalidUsageExitCode(t *testing.T) {
	testutil.Run(t, "unknown command", func(t *testutil.T) {
		// --interactive=false removes the update check and survey prompt.
		t.Override(&os.Args, []string{"skaffold", "unknown", "--interactive=false"})
		err := Run(io.Discard, io.Discard)
		t.CheckErrorAndExitCode(127, err)
	})
	testutil.Run(t, "unknown flag", func(t *testutil.T) {
		// --interactive=false removes the update check and survey prompt.
		t.Override(&os.Args, []string{"skaffold", "--help2", "--interactive=false"})
		err := Run(io.Discard, io.Discard)
		t.CheckErrorAndExitCode(127, err)
	})
	testutil.Run(t, "exactargs error", func(t *testutil.T) {
		// --interactive=false removes the update check and survey prompt.
		t.Override(&os.Args, []string{"skaffold", "config", "set", "a", "b", "c"})
		err := Run(io.Discard, io.Discard)
		t.CheckErrorAndExitCode(127, err)
	})
}

func TestMain_SuppressedErrorReporing(t *testing.T) {
	testutil.Run(t, "inspect should suppress error output", func(t *testutil.T) {
		var (
			output    bytes.Buffer
			errOutput bytes.Buffer
		)
		// non-existent profile should report an error
		t.Override(&os.Args, []string{"skaffold", "inspect", "build-env", "list", "--profile", "non-existent"})
		err := Run(&output, &errOutput)
		t.CheckError(true, err)
		t.CheckContains(`{"errorCode":`, output.String())
		t.CheckEmpty(errOutput.String())
	})

	testutil.Run(t, "diagnose should report error output", func(t *testutil.T) {
		var (
			output    bytes.Buffer
			errOutput bytes.Buffer
		)
		// non-existent profile should report an error
		t.Override(&os.Args, []string{"skaffold", "diagnose", "--yaml-only", "--profile", "non-existent"})
		err := Run(&output, &errOutput)
		t.CheckError(true, err)
		t.CheckEmpty(output.String())
		// checking quoted filename ensures there are no JSON errors too
		t.CheckContains(`unable to find configuration file "skaffold.yaml"`, errOutput.String())
	})
}
