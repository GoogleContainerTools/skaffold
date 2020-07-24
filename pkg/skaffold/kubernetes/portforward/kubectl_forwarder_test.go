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

package portforward

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUnavailablePort(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&waitPortNotFree, 100*time.Millisecond)

		// Return that the port is false, while also
		// adding a sync group so we know when isPortFree
		// has been called
		var portFreeWG sync.WaitGroup
		portFreeWG.Add(1)
		t.Override(&isPortFree, func(string, int) bool {
			portFreeWG.Done()
			return false
		})

		// Create a wait group that will only be
		// fulfilled when the forward function returns
		var forwardFunctionWG sync.WaitGroup
		forwardFunctionWG.Add(1)
		t.Override(&deferFunc, func() {
			forwardFunctionWG.Done()
		})

		var buf bytes.Buffer
		k := KubectlForwarder{
			out: &buf,
		}
		pfe := newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 8080, false)

		k.Forward(context.Background(), pfe)

		// wait for isPortFree to be called
		portFreeWG.Wait()

		// then, end port forwarding and wait for the forward function to return.
		pfe.terminationLock.Lock()
		pfe.terminated = true
		pfe.terminationLock.Unlock()
		forwardFunctionWG.Wait()

		// read output to make sure logs are expected
		t.CheckContains("port 8080 is taken", buf.String())
	})
}

func TestTerminate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	pfe := newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 8080, false)
	pfe.cancel = cancel

	k := &KubectlForwarder{}
	k.Terminate(pfe)
	if pfe.terminated != true {
		t.Fatalf("expected pfe.terminated to be true after termination")
	}
	if ctx.Err() != context.Canceled {
		t.Fatalf("expected cancel to be called")
	}
}

func TestMonitorErrorLogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip flaky test until it's fixed")
	}
	tests := []struct {
		description string
		input       string
		cmdRunning  bool
	}{
		{
			description: "no error logs appear",
			input:       "some random logs",
			cmdRunning:  true,
		},
		{
			description: "match on 'error forwarding port'",
			input:       "error forwarding port 8080",
		},
		{
			description: "match on 'unable to forward'",
			input:       "unable to forward 8080",
		},
		{
			description: "match on 'error upgrading connection'",
			input:       "error upgrading connection 8080",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&waitErrorLogs, 10*time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			cmdStr := "sleep"
			if runtime.GOOS == "windows" {
				cmdStr = "timeout"
			}
			cmd := kubectl.CommandContext(ctx, cmdStr, "5")
			if err := cmd.Start(); err != nil {
				t.Fatalf("error starting command: %v", err)
			}

			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				logs := strings.NewReader(test.input)

				k := KubectlForwarder{}
				k.monitorErrorLogs(ctx, logs, cmd, &portForwardEntry{})

				wg.Done()
			}()

			wg.Wait()

			// make sure the command is running or killed based on what's expected
			if test.cmdRunning {
				assertCmdIsRunning(t, cmd)
				cmd.Terminate()
			} else {
				assertCmdWasKilled(t, cmd)
			}
		})
	}
}

func assertCmdIsRunning(t *testutil.T, cmd *kubectl.Cmd) {
	if cmd.ProcessState != nil {
		t.Fatal("cmd was killed but expected to continue running")
	}
}

func assertCmdWasKilled(t *testutil.T, cmd *kubectl.Cmd) {
	if err := cmd.Wait(); err == nil {
		t.Fatal("cmd was not killed but expected to be killed")
	}
}

func TestAddressArg(t *testing.T) {
	ctx := context.Background()
	pfe := newPortForwardEntry(0, latest.PortForwardResource{Address: "0.0.0.0"}, "", "", "", "", 8080, false)
	cli := kubectl.NewFromRunContext(&runcontext.RunContext{})
	cmd := portForwardCommand(ctx, cli, pfe, nil)
	assertCmdContainsArgs(t, cmd, true, "--address", "0.0.0.0")
}

func TestNoAddressArg(t *testing.T) {
	ctx := context.Background()
	pfe := newPortForwardEntry(0, latest.PortForwardResource{}, "", "", "", "", 8080, false)
	cli := kubectl.NewFromRunContext(&runcontext.RunContext{})
	cmd := portForwardCommand(ctx, cli, pfe, nil)
	assertCmdContainsArgs(t, cmd, false, "--address")
}

func TestDefaultAddressArg(t *testing.T) {
	ctx := context.Background()
	pfe := newPortForwardEntry(0, latest.PortForwardResource{Address: "127.0.0.1"}, "", "", "", "", 8080, false)
	cli := kubectl.NewFromRunContext(&runcontext.RunContext{})
	cmd := portForwardCommand(ctx, cli, pfe, nil)
	assertCmdContainsArgs(t, cmd, false, "--address")
}

func assertCmdContainsArgs(t *testing.T, cmd *kubectl.Cmd, expected bool, args ...string) {
	if len(args) == 0 {
		return
	}
	contains := false
	cmdArgs := cmd.Args
	var start int
	var cmdArg string
	for start, cmdArg = range cmdArgs {
		if cmdArg == args[0] {
			contains = true
			break
		}
	}
	if !contains {
		if expected {
			t.Fatalf("cmd expected to contain args %v but args are %v", args, cmdArgs)
		}
		return
	}
	for i, arg := range args[1:] {
		if arg != cmdArgs[start+i+1] {
			contains = false
			break
		}
	}
	if contains != expected {
		if expected {
			t.Fatalf("cmd expected to contain args %v but args are %v", args, cmdArgs)
		} else {
			t.Fatalf("cmd expected not to contain args %v but args are %v", args, cmdArgs)
		}
	}
}
