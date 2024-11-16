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

package docker

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

// This may be overridden by KIND_EXPERIMENTAL_DOCKER_NETWORK env,
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

// ensureNetwork checks if docker network by name exists, if not it creates it
func ensureNetwork(name string) error {
	// check if network exists already and remove any duplicate networks
	exists, err := removeDuplicateNetworks(name)
	if err != nil {
		return err
	}

	// network already exists, we're good
	// TODO: the network might already exist and not have ipv6 ... :|
	// discussion: https://github.com/kubernetes-sigs/kind/pull/1508#discussion_r414594198
	if exists {
		return nil
	}

	// Generate unique subnet per network based on the name
	// obtained from the ULA fc00::/8 range
	// Use the MTU configured for the docker default network
	// Make N attempts with "probing" in case we happen to collide
	subnet := generateULASubnetFromName(name, 0)
	mtu := getDefaultNetworkMTU()
	err = createNetworkNoDuplicates(name, subnet, mtu)
	if err == nil {
		// Success!
		return nil
	}

	// On the first try check if ipv6 fails entirely on this machine
	// https://github.com/kubernetes-sigs/kind/issues/1544
	// Otherwise if it's not a pool overlap error, fail
	// If it is, make more attempts below
	if isIPv6UnavailableError(err) {
		// only one attempt, IPAM is automatic in ipv4 only
		return createNetworkNoDuplicates(name, "", mtu)
	}
	if isPoolOverlapError(err) {
		// pool overlap suggests perhaps another process created the network
		// check if network exists already and remove any duplicate networks
		exists, err := checkIfNetworkExists(name)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		// otherwise we'll start trying with different subnets
	} else {
		// unknown error ...
		return err
	}

	// keep trying for ipv6 subnets
	const maxAttempts = 5
	for attempt := int32(1); attempt < maxAttempts; attempt++ {
		subnet := generateULASubnetFromName(name, attempt)
		err = createNetworkNoDuplicates(name, subnet, mtu)
		if err == nil {
			// success!
			return nil
		}
		if isPoolOverlapError(err) {
			// pool overlap suggests perhaps another process created the network
			// check if network exists already and remove any duplicate networks
			exists, err := checkIfNetworkExists(name)
			if err != nil {
				return err
			}
			if exists {
				return nil
			}
			// otherwise we'll try again
			continue
		}
		// unknown error ...
		return err
	}
	return errors.New("exhausted attempts trying to find a non-overlapping subnet")
}

func createNetworkNoDuplicates(name, ipv6Subnet string, mtu int) error {
	if err := createNetwork(name, ipv6Subnet, mtu); err != nil && !isNetworkAlreadyExistsError(err) {
		return err
	}
	_, err := removeDuplicateNetworks(name)
	return err
}

func removeDuplicateNetworks(name string) (bool, error) {
	networks, err := sortedNetworksWithName(name)
	if err != nil {
		return false, err
	}
	if len(networks) > 1 {
		if err := deleteNetworks(networks[1:]...); err != nil && !isOnlyErrorNoSuchNetwork(err) {
			return false, err
		}
	}
	return len(networks) > 0, nil
}

func createNetwork(name, ipv6Subnet string, mtu int) error {
	args := []string{"network", "create", "-d=bridge",
		"-o", "com.docker.network.bridge.enable_ip_masquerade=true",
	}
	if mtu > 0 {
		args = append(args, "-o", fmt.Sprintf("com.docker.network.driver.mtu=%d", mtu))
	}
	if ipv6Subnet != "" {
		args = append(args, "--ipv6", "--subnet", ipv6Subnet)
	}
	args = append(args, name)
	return exec.Command("docker", args...).Run()
}

// getDefaultNetworkMTU obtains the MTU from the docker default network
func getDefaultNetworkMTU() int {
	cmd := exec.Command("docker", "network", "inspect", "bridge",
		"-f", `{{ index .Options "com.docker.network.driver.mtu" }}`)
	lines, err := exec.OutputLines(cmd)
	if err != nil || len(lines) != 1 {
		return 0
	}
	mtu, err := strconv.Atoi(lines[0])
	if err != nil {
		return 0
	}
	return mtu
}

func sortedNetworksWithName(name string) ([]string, error) {
	// query which networks exist with the name
	ids, err := networksWithName(name)
	if err != nil {
		return nil, err
	}
	// we can skip sorting if there are less than 2
	if len(ids) < 2 {
		return ids, nil
	}
	// inspect them to get more detail for sorting
	networks, err := inspectNetworks(ids)
	if err != nil {
		return nil, err
	}
	// deterministically sort networks
	// NOTE: THIS PART IS IMPORTANT!
	sortNetworkInspectEntries(networks)
	// return network IDs
	sortedIDs := make([]string, 0, len(networks))
	for i := range networks {
		sortedIDs = append(sortedIDs, networks[i].ID)
	}
	return sortedIDs, nil
}

func sortNetworkInspectEntries(networks []networkInspectEntry) {
	sort.Slice(networks, func(i, j int) bool {
		// we want networks with active containers first
		if len(networks[i].Containers) > len(networks[j].Containers) {
			return true
		}
		return networks[i].ID < networks[j].ID
	})
}

func inspectNetworks(networkIDs []string) ([]networkInspectEntry, error) {
	inspectOut, err := exec.Output(exec.Command("docker", append([]string{"network", "inspect"}, networkIDs...)...))
	// NOTE: the caller can detect if the network isn't present in the output anyhow
	// we don't want to fail on this here.
	if err != nil && !isOnlyErrorNoSuchNetwork(err) {
		return nil, err
	}
	// parse
	networks := []networkInspectEntry{}
	if err := json.Unmarshal(inspectOut, &networks); err != nil {
		return nil, errors.Wrap(err, "failed to decode networks list")
	}
	return networks, nil
}

type networkInspectEntry struct {
	ID string `json:"Id"`
	// NOTE: we don't care about the contents here but we need to parse
	// how many entries exist in the containers map
	Containers map[string]map[string]string `json:"Containers"`
}

// networksWithName returns a list of network IDs for networks with this name
func networksWithName(name string) ([]string, error) {
	lsOut, err := exec.Output(exec.Command(
		"docker", "network", "ls",
		"--filter=name=^"+regexp.QuoteMeta(name)+"$",
		"--format={{.ID}}", // output as unambiguous IDs
	))
	if err != nil {
		return nil, err
	}
	cleaned := strings.TrimSuffix(string(lsOut), "\n")
	if cleaned == "" { // avoid returning []string{""}
		return nil, nil
	}
	return strings.Split(cleaned, "\n"), nil
}

func checkIfNetworkExists(name string) (bool, error) {
	out, err := exec.Output(exec.Command(
		"docker", "network", "ls",
		"--filter=name=^"+regexp.QuoteMeta(name)+"$",
		"--format={{.Name}}",
	))
	return strings.HasPrefix(string(out), name), err
}

func isIPv6UnavailableError(err error) bool {
	rerr := exec.RunErrorForError(err)
	if rerr == nil {
		return false
	}
	errorMessage := string(rerr.Output)
	// we get this error when ipv6 was disabled in docker
	const dockerIPV6DisabledError = "Error response from daemon: Cannot read IPv6 setup for bridge"
	// TODO: this is fragile, and only necessary due to docker enabling ipv6 by default
	// even on hosts that lack ip6tables setup.
	// Preferably users would either have ip6tables setup properly or else disable ipv6 in docker
	const dockerIPV6TablesError = "Error response from daemon: Failed to Setup IP tables: Unable to enable NAT rule:  (iptables failed: ip6tables"
	return strings.HasPrefix(errorMessage, dockerIPV6DisabledError) || strings.HasPrefix(errorMessage, dockerIPV6TablesError)
}

func isPoolOverlapError(err error) bool {
	rerr := exec.RunErrorForError(err)
	return rerr != nil && strings.HasPrefix(string(rerr.Output), "Error response from daemon: Pool overlaps with other one on this address space") || strings.Contains(string(rerr.Output), "networks have overlapping")
}

func isNetworkAlreadyExistsError(err error) bool {
	rerr := exec.RunErrorForError(err)
	return rerr != nil && strings.HasPrefix(string(rerr.Output), "Error response from daemon: network with name") && strings.Contains(string(rerr.Output), "already exists")
}

// returns true if:
// - err only contains no such network errors
func isOnlyErrorNoSuchNetwork(err error) bool {
	rerr := exec.RunErrorForError(err)
	if rerr == nil {
		return false
	}
	// check all lines of output from errored command
	b := bytes.NewBuffer(rerr.Output)
	for {
		l, err := b.ReadBytes('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return false
		}
		// if the line begins with Error: No such network: it's fine
		s := string(l)
		if strings.HasPrefix(s, "Error: No such network:") {
			continue
		}
		// other errors are not fine
		if strings.HasPrefix(s, "Error: ") {
			return false
		}
		// other line contents should just be network references
	}
	return true
}

func deleteNetworks(networks ...string) error {
	return exec.Command("docker", append([]string{"network", "rm"}, networks...)...).Run()
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
