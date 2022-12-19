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

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/GoogleContainerTools/skaffold/v2/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestVerifyPassingTestsWithEnvVar(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	t.Setenv("FOO", "foo-var")
	tmp := t.TempDir()
	logFile := filepath.Join(tmp, uuid.New().String()+"logs.json")

	rpcPort := randomPort()
	// `--default-repo=` is used to cancel the default repo that is set by default.
	out, err := skaffold.Verify("--default-repo=", "--rpc-port", rpcPort, "--event-log-file", logFile).InDir("testdata/verify-succeed").RunWithCombinedOutput(t)
	logs := string(out)

	testutil.CheckError(t, false, err)
	testutil.CheckContains(t, "Hello from Docker!", logs)
	testutil.CheckContains(t, "foo-var", logs)
	testutil.CheckContains(t, "alpine-1", logs)
	testutil.CheckContains(t, "alpine-2", logs)

	// verify logs are in the event output as well
	b, err := os.ReadFile(logFile + ".v2")
	if err != nil {
		t.Fatalf("error reading %s", logFile+".v2")
	}
	v2EventLogs := string(b)
	testutil.CheckContains(t, "Hello from Docker!", v2EventLogs)
	testutil.CheckContains(t, "foo-var", v2EventLogs)

	// TODO(aaron-prindle) verify that SUCCEEDED event is found where expected
}

func TestVerifyOneTestFailsWithEnvVar(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	t.Setenv("FOO", "foo-var")
	tmp := t.TempDir()
	logFile := filepath.Join(tmp, uuid.New().String()+"logs.json")

	rpcPort := randomPort()
	// `--default-repo=` is used to cancel the default repo that is set by default.
	out, err := skaffold.Verify("--default-repo=", "--rpc-port", rpcPort, "--event-log-file", logFile).InDir("testdata/verify-fail").RunWithCombinedOutput(t)
	logs := string(out)

	testutil.CheckError(t, true, err)
	testutil.CheckContains(t, "Hello from Docker!", logs)
	testutil.CheckContains(t, "foo-var", logs)

	// verify logs are in the event output as well
	b, err := os.ReadFile(logFile + ".v2")
	if err != nil {
		t.Fatalf("error reading %s", logFile+".v2")
	}
	v2EventLogs := string(b)
	testutil.CheckContains(t, "Hello from Docker!", v2EventLogs)
	testutil.CheckContains(t, "foo-var", v2EventLogs)

	// TODO(aaron-prindle) verify that FAILED event is found where expected
}

func TestVerifyNoTestsFails(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)
	// `--default-repo=` is used to cancel the default repo that is set by default.
	out, err := skaffold.Verify("--default-repo=").InDir("testdata/verify-no-tests").RunWithCombinedOutput(t)
	logs := string(out)

	testutil.CheckError(t, true, err)
	testutil.CheckContains(t, "verify command expects non-zero number of test cases", logs)
}
