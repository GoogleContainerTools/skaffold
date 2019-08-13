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
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetAvailablePort(t *testing.T) {
	AssertCompetingProcessesCanSucceed(&sync.Map{}, t)
}

//TODO this is copy pasted to portforward.forwardedPorts testing as well - it should go away when we introduce port brokering
// https://github.com/GoogleContainerTools/skaffold/issues/2503
func AssertCompetingProcessesCanSucceed(ports ForwardedPorts, t *testing.T) {
	t.Helper()
	N := 100
	var (
		errors int32
		wg     sync.WaitGroup
	)
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {

			port := GetAvailablePort(4503, ports)

			l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", Loopback, port))
			if err != nil {
				atomic.AddInt32(&errors, 1)
			} else {
				l.Close()
			}
			time.Sleep(2 * time.Second)
			wg.Done()
		}()
	}
	wg.Wait()
	if atomic.LoadInt32(&errors) > 0 {
		t.Fatalf("A port that was available couldn't be used %d times", errors)
	}
}
