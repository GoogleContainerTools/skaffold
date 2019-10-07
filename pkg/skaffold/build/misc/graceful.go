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
	"time"

	"github.com/sirupsen/logrus"
)

func HandleGracefulTermination(ctx context.Context, cmd *exec.Cmd) error {
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			// On windows we can't send specific signals to processes, so we kill the process immediately
			if runtime.GOOS == "windows" {
				cmd.Process.Kill()
				return
			}

			logrus.Debugf("Sending SIGINT to process %v\n", cmd.Process.Pid)
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				// kill process on error
				cmd.Process.Kill()
			}

			// wait 2 seconds or wait for the process to complete
			select {
			case <-time.After(2 * time.Second):
				logrus.Debugf("Killing process %v\n", cmd.Process.Pid)
				// forcefully kill process after 2 seconds grace period
				cmd.Process.Kill()
			case <-done:
				return
			}
		case <-done:
			return
		}
	}()

	return cmd.Wait()
}
