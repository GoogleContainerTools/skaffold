/*
Copyright 2018 The Kubernetes Authors.

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

package config

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/internal/sets"
)

// similar to valid docker container names, but since we will prefix
// and suffix this name, we can relax it a little
// see NewContext() for usage
// https://godoc.org/github.com/docker/docker/daemon/names#pkg-constants
var validNameRE = regexp.MustCompile(`^[a-z0-9.-]+$`)

// Validate returns a ConfigErrors with an entry for each problem
// with the config, or nil if there are none
func (c *Cluster) Validate() error {
	errs := []error{}

	// validate the name
	if !validNameRE.MatchString(c.Name) {
		errs = append(errs, errors.Errorf("'%s' is not a valid cluster name, cluster names must match `%s`",
			c.Name, validNameRE.String()))
	}

	// the api server port only needs checking if we aren't picking a random one
	// at runtime
	if c.Networking.APIServerPort != 0 {
		// validate api server listen port
		if err := validatePort(c.Networking.APIServerPort); err != nil {
			errs = append(errs, errors.Wrapf(err, "invalid apiServerPort"))
		}
	}

	// podSubnet should be a valid CIDR
	if err := validateSubnets(c.Networking.PodSubnet, c.Networking.IPFamily); err != nil {
		errs = append(errs, errors.Errorf("invalid pod subnet %v", err))
	}

	// serviceSubnet should be a valid CIDR
	if err := validateSubnets(c.Networking.ServiceSubnet, c.Networking.IPFamily); err != nil {
		errs = append(errs, errors.Errorf("invalid service subnet %v", err))
	}

	// KubeProxyMode should be iptables or ipvs
	if c.Networking.KubeProxyMode != IPTablesProxyMode && c.Networking.KubeProxyMode != IPVSProxyMode &&
		c.Networking.KubeProxyMode != NoneProxyMode {
		errs = append(errs, errors.Errorf("invalid kubeProxyMode: %s", c.Networking.KubeProxyMode))
	}

	// validate nodes
	numByRole := make(map[NodeRole]int32)
	// All nodes in the config should be valid
	for i, n := range c.Nodes {
		// validate the node
		if err := n.Validate(); err != nil {
			errs = append(errs, errors.Errorf("invalid configuration for node %d: %v", i, err))
		}
		// update role count
		if num, ok := numByRole[n.Role]; ok {
			numByRole[n.Role] = 1 + num
		} else {
			numByRole[n.Role] = 1
		}
	}

	// there must be at least one control plane node
	numControlPlane, anyControlPlane := numByRole[ControlPlaneRole]
	if !anyControlPlane || numControlPlane < 1 {
		errs = append(errs, errors.Errorf("must have at least one %s node", string(ControlPlaneRole)))
	}

	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}

// Validate returns a ConfigErrors with an entry for each problem
// with the Node, or nil if there are none
func (n *Node) Validate() error {
	errs := []error{}

	// validate node role should be one of the expected values
	switch n.Role {
	case ControlPlaneRole,
		WorkerRole:
	default:
		errs = append(errs, errors.Errorf("%q is not a valid node role", n.Role))
	}

	// image should be defined
	if n.Image == "" {
		errs = append(errs, errors.New("image is a required field"))
	}

	// validate extra port forwards
	for _, mapping := range n.ExtraPortMappings {
		if err := validatePort(mapping.HostPort); err != nil {
			errs = append(errs, errors.Wrapf(err, "invalid hostPort"))
		}

		if err := validatePort(mapping.ContainerPort); err != nil {
			errs = append(errs, errors.Wrapf(err, "invalid containerPort"))
		}
	}

	if err := validatePortMappings(n.ExtraPortMappings); err != nil {
		errs = append(errs, errors.Wrapf(err, "invalid portMapping"))
	}

	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}

	return nil
}

func validatePortMappings(portMappings []PortMapping) error {
	errMsg := "port mapping with same listen address, port and protocol already configured"

	wildcardAddrIPv4 := net.ParseIP("0.0.0.0")
	wildcardAddrIPv6 := net.ParseIP("::")

	// bindMap has the following key-value structure
	// PORT/PROTOCOL: [ IP ]
	// { 80/TCP: [ 127.0.0.1, 192.168.2.3 ], 80/UDP: [ 0.0.0.0 ] }
	bindMap := make(map[string]sets.String)

	formatPortProtocol := func(port int32, protocol PortMappingProtocol) string {
		return fmt.Sprintf("%d/%s", port, protocol)
	}

	for _, portMapping := range portMappings {
		addr := net.ParseIP(portMapping.ListenAddress)
		addrString := addr.String()

		portProtocol := formatPortProtocol(portMapping.HostPort, portMapping.Protocol)
		possibleErr := fmt.Errorf("%s: %s:%s", errMsg, addrString, portProtocol)

		// in golang 0.0.0.0 and [::] are equivalent, convert [::] -> 0.0.0.0
		// https://github.com/golang/go/issues/48723
		if addr.Equal(wildcardAddrIPv6) {
			addr = wildcardAddrIPv4
			addrString = addr.String()
		}

		if _, ok := bindMap[portProtocol]; ok {

			// wildcard address case:
			// return error if there already exists any listen address for same port and protocol
			if addr.Equal(wildcardAddrIPv4) {
				if bindMap[portProtocol].Len() > 0 {
					return possibleErr
				}
			}

			// direct duplicate & wild card present check:
			// return error if same combination of ip, port and protocol already exists in bindMap.
			// return error if wildcard address is already present for same port & protocol
			if bindMap[portProtocol].Has(addrString) || bindMap[portProtocol].Has(wildcardAddrIPv4.String()) {
				return possibleErr
			}
		} else {
			// initialize the set
			bindMap[portProtocol] = sets.NewString()
		}

		// add the entry to bindMap
		bindMap[portProtocol].Insert(addrString)
	}
	return nil
}

func validatePort(port int32) error {
	// NOTE: -1 is a special value for auto-selecting the port in the container
	// backend where possible as opposed to in kind itself.
	if port < -1 || port > 65535 {
		return errors.Errorf("invalid port number: %d", port)
	}
	return nil
}

func validateSubnets(subnetStr string, ipFamily ClusterIPFamily) error {
	allErrs := []error{}

	cidrsString := strings.Split(subnetStr, ",")
	subnets := make([]*net.IPNet, 0, len(cidrsString))
	for _, cidrString := range cidrsString {
		_, cidr, err := net.ParseCIDR(cidrString)
		if err != nil {
			return fmt.Errorf("failed to parse cidr value:%q with error: %v", cidrString, err)
		}
		subnets = append(subnets, cidr)
	}

	dualstack := ipFamily == DualStackFamily
	switch {
	// if no subnets are defined
	case len(subnets) == 0:
		allErrs = append(allErrs, errors.New("no subnets defined"))
	// if DualStack only 2 CIDRs allowed
	case dualstack && len(subnets) > 2:
		allErrs = append(allErrs, errors.New("expected one (IPv4 or IPv6) CIDR or two CIDRs from each family for dual-stack networking"))
	// if DualStack and there are 2 CIDRs validate if there is at least one of each IP family
	case dualstack && len(subnets) == 2:
		areDualStackCIDRs, err := isDualStackCIDRs(subnets)
		if err != nil {
			allErrs = append(allErrs, err)
		} else if !areDualStackCIDRs {
			allErrs = append(allErrs, errors.New("expected one (IPv4 or IPv6) CIDR or two CIDRs from each family for dual-stack networking"))
		}
	// if not DualStack only one CIDR allowed
	case !dualstack && len(subnets) > 1:
		allErrs = append(allErrs, errors.New("only one CIDR allowed for single-stack networking"))
	case ipFamily == IPv4Family && subnets[0].IP.To4() == nil:
		allErrs = append(allErrs, errors.New("expected IPv4 CIDR for IPv4 family"))
	case ipFamily == IPv6Family && subnets[0].IP.To4() != nil:
		allErrs = append(allErrs, errors.New("expected IPv6 CIDR for IPv6 family"))
	}

	if len(allErrs) > 0 {
		return errors.NewAggregate(allErrs)
	}
	return nil
}

// isDualStackCIDRs returns if
// - all are valid cidrs
// - at least one cidr from each family (v4 or v6)
func isDualStackCIDRs(cidrs []*net.IPNet) (bool, error) {
	v4Found := false
	v6Found := false
	for _, cidr := range cidrs {
		if cidr == nil {
			return false, fmt.Errorf("cidr %v is invalid", cidr)
		}

		if v4Found && v6Found {
			continue
		}

		if cidr.IP != nil && cidr.IP.To4() == nil {
			v6Found = true
			continue
		}
		v4Found = true
	}

	return v4Found && v6Found, nil
}
