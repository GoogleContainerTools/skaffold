/*
Copyright 2020 The Kubernetes Authors.

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

package podman

import (
	"crypto/sha1"
	"encoding/binary"
	"net"
	"regexp"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// This may be overridden by KIND_EXPERIMENTAL_PODMAN_NETWORK env,
// experimentally...
//
// By default currently picking a single network is equivalent to the previous
// behavior *except* that we moved from the default bridge to a user defined
// network because the default bridge is actually special versus any other
// docker network and lacks the embedded DNS
//
// For now this also makes it easier for apps to join the same network, and
// leaves users with complex networking desires to create and manage their own
// networks.
const fixedNetworkName = "kind"

// ensureNetwork creates a new network
// podman only creates IPv6 networks for versions >= 2.2.0
func ensureNetwork(name string) error {
	// network already exists
	if checkIfNetworkExists(name) {
		return nil
	}

	// generate unique subnet per network based on the name
	// obtained from the ULA fc00::/8 range
	// Make N attempts with "probing" in case we happen to collide
	subnet := generateULASubnetFromName(name, 0)
	err := createNetwork(name, subnet)
	if err == nil {
		// Success!
		return nil
	}

	if isUnknownIPv6FlagError(err) ||
		isIPv6DisabledError(err) {
		return createNetwork(name, "")
	}

	// Only continue if the error is because of the subnet range
	// is already allocated
	if !isPoolOverlapError(err) {
		return err
	}

	// keep trying for ipv6 subnets
	const maxAttempts = 5
	for attempt := int32(1); attempt < maxAttempts; attempt++ {
		subnet := generateULASubnetFromName(name, attempt)
		err = createNetwork(name, subnet)
		if err == nil {
			// success!
			return nil
		} else if !isPoolOverlapError(err) {
			// unknown error ...
			return err
		}
	}
	return errors.New("exhausted attempts trying to find a non-overlapping subnet")

}

func createNetwork(name, ipv6Subnet string) error {
	if ipv6Subnet == "" {
		return exec.Command("podman", "network", "create", "-d=bridge", name).Run()
	}
	return exec.Command("podman", "network", "create", "-d=bridge",
		"--ipv6", "--subnet", ipv6Subnet, name).Run()
}

func checkIfNetworkExists(name string) bool {
	_, err := exec.Output(exec.Command(
		"podman", "network", "inspect",
		regexp.QuoteMeta(name),
	))
	return err == nil
}

func isUnknownIPv6FlagError(err error) bool {
	rerr := exec.RunErrorForError(err)
	return rerr != nil &&
		strings.Contains(string(rerr.Output), "unknown flag: --ipv6")
}

func isIPv6DisabledError(err error) bool {
	rerr := exec.RunErrorForError(err)
	return rerr != nil &&
		strings.Contains(string(rerr.Output), "is ipv6 enabled in the kernel")
}

func isPoolOverlapError(err error) bool {
	rerr := exec.RunErrorForError(err)
	if rerr == nil {
		return false
	}
	output := string(rerr.Output)
	return strings.Contains(output, "is already used on the host or by another config") ||
		strings.Contains(output, "is being used by a network interface") ||
		strings.Contains(output, "is already being used by a cni configuration")
}

// generateULASubnetFromName generate an IPv6 subnet based on the
// name and Nth probing attempt
func generateULASubnetFromName(name string, attempt int32) string {
	ip := make([]byte, 16)
	ip[0] = 0xfc
	ip[1] = 0x00
	h := sha1.New()
	_, _ = h.Write([]byte(name))
	_ = binary.Write(h, binary.LittleEndian, attempt)
	bs := h.Sum(nil)
	for i := 2; i < 8; i++ {
		ip[i] = bs[i]
	}
	subnet := &net.IPNet{
		IP:   net.IP(ip),
		Mask: net.CIDRMask(64, 128),
	}
	return subnet.String()
}
