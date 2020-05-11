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

package misc

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// For testing
var (
	gracePeriod = 2 * time.Second
)

func HandleGracefulTermination(ctx context.Context, cmd *exec.Cmd) error {
	done := make(chan bool, 1) // Non blocking
	defer close(done)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			// On windows we can't send specific signals to processes, so we kill the process immediately
			if runtime.GOOS == "windows" {
				cmd.Process.Kill()
				return
			}

			logrus.Debugln("Sending SIGINT to process", cmd.Process.Pid)
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				// kill process on error
				cmd.Process.Kill()
				return
			}

			// wait 2 seconds or wait for the process to complete
			select {
			case <-time.After(gracePeriod):
				logrus.Debugln("Killing process", cmd.Process.Pid)
				// forcefully kill process after grace period
				cmd.Process.Kill()
			case <-done:
				return
			}
		case <-done:
			return
		}
	}()

	err := cmd.Wait()
	done <- true
	wg.Wait()
	return err
}
