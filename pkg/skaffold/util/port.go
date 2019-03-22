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

	"github.com/sirupsen/logrus"
)

// Loopback network address. Skaffold should not bind to 0.0.0.0
// unless we really want to expose something to the network.
const Loopback = "127.0.0.1"

// First, check if the provided port is available. If so, use it.
// If not, check if any of the next 10 subsequent ports are available.
// If not, check if any of ports 4503-4533 are available.
// If not, return a random port, which hopefully won't collide with any future containers

// See https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt,
func GetAvailablePort(port int, forwardedPorts *sync.Map) int {
	if getPortIfAvailable(port, forwardedPorts) {
		return port
	}

	// try the next 10 ports after the provided one
	for i := 0; i < 10; i++ {
		port++
		if getPortIfAvailable(port, forwardedPorts) {
			logrus.Debugf("found open port: %d", port)
			return port
		}
	}

	for port = 4503; port <= 4533; port++ {
		if getPortIfAvailable(port, forwardedPorts) {
			return port
		}
	}

	l, err := net.Listen("tcp", fmt.Sprintf("%s:0", Loopback))
	if err != nil {
		return -1
	}

	p := l.Addr().(*net.TCPAddr).Port

	forwardedPorts.Store(p, true)
	l.Close()
	return p
}

func getPortIfAvailable(p int, forwardedPorts *sync.Map) bool {
	alreadyUsed, loaded := forwardedPorts.LoadOrStore(p, true)
	if loaded && alreadyUsed.(bool) {
		return false
	}

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", Loopback, p))
	if err != nil {
		return false
	}

	l.Close()
	return true
}
