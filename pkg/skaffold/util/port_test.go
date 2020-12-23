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
	"testing"
)

func TestPortSet(t *testing.T) {
	pf := &PortSet{}

	// Try to store a port
	pf.Set(9000)

	// Try to load the port
	if alreadySet := pf.LoadOrSet(9000); !alreadySet {
		t.Fatal("didn't load port 9000 correctly")
	}

	if alreadySet := pf.LoadOrSet(4000); alreadySet {
		t.Fatal("didn't store port 4000 correctly")
	}

	if alreadySet := pf.LoadOrSet(4000); !alreadySet {
		t.Fatal("didn't load port 4000 correctly")
	}
}

func TestGetAvailablePort(t *testing.T) {
	N := 100

	var (
		ports  PortSet
		lock   sync.Mutex
		wg     sync.WaitGroup
		errors = map[int]error{}
	)

	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			port := GetAvailablePort("127.0.0.1", 4503, &ports)

			l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", Loopback, port))
			if err != nil {
				lock.Lock()
				errors[port] = err
				lock.Unlock()
			} else {
				l.Close()
			}
			wg.Done()
		}()
	}
	wg.Wait()

	for port, err := range errors {
		t.Errorf("available port (%d) couldn't be used: %w", port, err)
	}
}
