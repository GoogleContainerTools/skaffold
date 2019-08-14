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
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestUnavailablePort(t *testing.T) {

	original := isPortFree
	defer func() { isPortFree = original }()

	portFreeWG := &sync.WaitGroup{}
	portFreeWG.Add(1)
	isPortFree = func(_ int) bool {
		defer portFreeWG.Done()
		return false
	}

	buf := bytes.NewBuffer([]byte{})
	wg := &sync.WaitGroup{}
	k := KubectlForwarder{
		out:     buf,
		testing: true,
		wg:      wg,
	}
	pfe := &portForwardEntry{localPort: 8080, terminationLock: &sync.Mutex{}}
	k.Forward(context.Background(), pfe)

	// wait for isPortFree to be called
	portFreeWG.Wait()

	// then, end port forwarding
	pfe.terminationLock.Lock()
	pfe.terminated = true
	pfe.terminationLock.Unlock()
	wg.Wait()

	// read output to make sure ports are expected
	output := buf.String()
	if !strings.Contains(output, "port 8080 is taken") {
		t.Fatalf("port wasn't available but didn't get warning, got: \n%s", output)
	}
}

func TestMonitorErrorLogs(t *testing.T) {
	tests := []struct {
		description string
		input       string
		cmdRunning  bool
	}{
		{
			description: "no error logs appear",
			input:       "some random logs",
			cmdRunning:  true,
		}, {
			description: "a match on 'error forwarding port'",
			input:       "error forwarding port 8080",
		}, {
			description: "match on 'unable to forward'",
			input:       "unable to forward 8080",
		}, {
			description: "match on 'error upgrading connection'",
			input:       "error upgrading connection 8080",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			cmd := exec.Command("sleep", "5")
			if err := cmd.Start(); err != nil {
				t.Fatal("error starting command")
			}

			wg := &sync.WaitGroup{}
			wg.Add(1)

			go func() {
				defer wg.Done()
				k := KubectlForwarder{}
				logs := bytes.NewBuffer([]byte(test.input))
				k.monitorErrorLogs(ctx, logs, cmd, &portForwardEntry{})
			}()

			// need to sleep for one second before cancelling the context
			// because there is a one second sleep in the switch statement
			// of monitorLogs
			time.Sleep(1 * time.Second)

			// cancel the context and then wait for monitorErrorLogs to return
			cancel()
			wg.Wait()

			// make sure the command is running or killed based on what's expected
			if test.cmdRunning {
				assertCmdIsRunning(t, cmd)
				cmd.Process.Kill()
			} else {
				assertCmdWasKilled(t, cmd)
			}
		})
	}
}

func assertCmdIsRunning(t *testing.T, cmd *exec.Cmd) {
	if cmd.ProcessState != nil {
		t.Fatal("cmd was killed but expected to continue running")
	}
}

func assertCmdWasKilled(t *testing.T, cmd *exec.Cmd) {
	if err := cmd.Wait(); err == nil {
		t.Fatal("cmd was not killed but expected to be killed")
	}
}
