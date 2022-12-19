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
	"fmt"
	"net"
	"sort"
	"sync"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// Loopback network address. Skaffold should not bind to 0.0.0.0
// unless we really want to expose something to the network.
const Loopback = "127.0.0.1"

// Network address which represent any address. This is the default that
// we should use when checking if port is free.
const Any = ""

type PortSet struct {
	ports map[int]bool
	lock  sync.Mutex
}

func (f *PortSet) Set(port int) {
	f.lock.Lock()

	if f.ports == nil {
		f.ports = map[int]bool{}
	}
	f.ports[port] = true

	f.lock.Unlock()
}

func (f *PortSet) LoadOrSet(port int) bool {
	f.lock.Lock()

	exists := f.ports[port]
	if !exists {
		if f.ports == nil {
			f.ports = map[int]bool{}
		}
		f.ports[port] = true
	}

	f.lock.Unlock()

	return exists
}

func (f *PortSet) Delete(port int) {
	f.lock.Lock()
	delete(f.ports, port)
	f.lock.Unlock()
}

func (f *PortSet) Length() int {
	f.lock.Lock()
	length := len(f.ports)
	f.lock.Unlock()
	return length
}

func (f *PortSet) List() []int {
	var list []int

	f.lock.Lock()
	for k := range f.ports {
		list = append(list, k)
	}
	f.lock.Unlock()

	sort.Ints(list)
	return list
}

// GetAvailablePort returns an available port that is near the requested port when possible.
// First, check if the provided port is available on the specified address and INADDR_ANY. If so, use it.
// If not, check if any of the next 10 subsequent ports are available.
// If not, check if any of ports 4503-4533 are available.
// If not, return a random port, which hopefully won't collide with any future containers
//
// See https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt
func GetAvailablePort(address string, port int, usedPorts *PortSet) int {
	log.Entry(context.TODO()).Tracef("looking for port: %s:%d", address, port)
	if port > 0 {
		if getPortIfAvailable(address, port, usedPorts) {
			log.Entry(context.TODO()).Debugf("found open port: %d", port)
			return port
		}

		// try the next 10 ports after the provided one
		for i := 0; i < 10; i++ {
			port++
			if getPortIfAvailable(address, port, usedPorts) {
				log.Entry(context.TODO()).Debugf("found open port: %d", port)
				return port
			}
		}
	}

	for port = 4503; port <= 4533; port++ {
		if getPortIfAvailable(address, port, usedPorts) {
			log.Entry(context.TODO()).Debugf("found open port: %d", port)
			return port
		}
	}

	l, err := net.Listen("tcp", fmt.Sprintf("%s:0", address))
	if err != nil {
		return -1
	}

	p := l.Addr().(*net.TCPAddr).Port

	usedPorts.Set(p)
	l.Close()
	return p
}

func getPortIfAvailable(address string, p int, usedPorts *PortSet) bool {
	if alreadySet := usedPorts.LoadOrSet(p); alreadySet {
		log.Entry(context.TODO()).Tracef("port %d already allocated", p)
		return false
	}

	return IsPortFree(address, p)
}

func IsPortFree(address string, p int) bool {
	// Ensure the port is available across all interfaces
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
	if err != nil {
		log.Entry(context.TODO()).Tracef("port INADDR_ANY:%d already bound: %v", p, err)
		return false
	} else if l == nil {
		log.Entry(context.TODO()).Tracef("port INADDR_ANY:%d nil listener", p)
		return false
	}
	l.Close()
	log.Entry(context.TODO()).Tracef("was able to obtain INADDR_ANY:%d", p)

	if address != Any {
		// Ensure the port is available on the specific interface too
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, p))
		if err != nil {
			log.Entry(context.TODO()).Tracef("port %s:%d already bound: %v", address, p, err)
			return false
		} else if l == nil {
			log.Entry(context.TODO()).Tracef("port %s:%d nil listener", address, p)
			return false
		}
		l.Close()
		log.Entry(context.TODO()).Tracef("was able to obtain %s:%d", address, p)
	}
	return true
}

// AllocatePort looks for a port close to desiredPort, using the provided implementation of
// isPortAvailable to determine what ports can be used.
// We deal with wrapping and avoid allocating ports < 1024
// TODO(nkubala)[09/14/21]: plumb through context from callers
func AllocatePort(isPortAvailable func(int32) bool, desiredPort int32) int32 {
	var maxPort int32 = 65535 // ports are normally [1-65535]
	if desiredPort < 1024 || desiredPort > maxPort {
		log.Entry(context.TODO()).Debugf("skipping reserved port %d", desiredPort)
		desiredPort = 1024 // skip reserved ports
	}
	// We assume ports are rather sparsely allocated, so even if desiredPort
	// is allocated, desiredPort+1 or desiredPort+2 are likely to be free
	for port := desiredPort; port < maxPort; port++ {
		if isPortAvailable(port) {
			return port
		}
	}
	for port := desiredPort; port > 1024; port-- {
		if isPortAvailable(port) {
			return port
		}
	}
	panic("cannot find available port") // exceedingly unlikely
}
