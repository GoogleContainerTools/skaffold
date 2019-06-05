// +build !windows

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

package util

import (
	"context"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestCatchCtrlC(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	CatchCtrlC(cancel)

	go func() {
		<-ctx.Done()
		wg.Done()
	}()

	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	wg.Wait()
}

func TestWaitForSignalOrCtrlC(t *testing.T) {
	tests := []struct {
		name          string
		killWithCtrlC bool
	}{
		{
			name:          "kill with signal",
			killWithCtrlC: false,
		},
		{
			name:          "kill with ctrl-c",
			killWithCtrlC: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			trigger := make(chan bool, 1)

			go func() {
				WaitForSignalOrCtrlC(context.Background(), trigger)
				wg.Done()
			}()

			// give goroutine time to start worker before sending SIGINT
			// otherwise, SIGINT sometimes gets sent before we can catch it
			time.Sleep(50 * time.Millisecond)

			if tt.killWithCtrlC {
				syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			} else {
				trigger <- true
			}

			wg.Wait()
		})
	}
}
