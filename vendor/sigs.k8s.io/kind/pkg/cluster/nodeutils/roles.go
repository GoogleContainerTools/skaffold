/*
Copyright 2019 The Kubernetes Authors.

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

package nodeutils

import (
	"sort"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
)

// SelectNodesByRole returns a list of nodes with the matching role
// TODO(bentheelder): remove this in favor of specific role select methods
// and avoid the unnecessary error handling
func SelectNodesByRole(allNodes []nodes.Node, role string) ([]nodes.Node, error) {
	out := []nodes.Node{}
	for _, node := range allNodes {
		nodeRole, err := node.Role()
		if err != nil {
			return nil, err
		}
		if nodeRole == role {
			out = append(out, node)
		}
	}
	return out, nil
}

// InternalNodes returns the list of container IDs for the "nodes" in the cluster
// that are ~Kubernetes nodes, as opposed to e.g. the external loadbalancer for HA
func InternalNodes(allNodes []nodes.Node) ([]nodes.Node, error) {
	selectedNodes := []nodes.Node{}
	for _, node := range allNodes {
		nodeRole, err := node.Role()
		if err != nil {
			return nil, err
		}
		if nodeRole == constants.WorkerNodeRoleValue || nodeRole == constants.ControlPlaneNodeRoleValue {
			selectedNodes = append(selectedNodes, node)
		}
	}
	return selectedNodes, nil
}

// ExternalLoadBalancerNode returns a node handle for the external control plane
// loadbalancer node or nil if there isn't one
func ExternalLoadBalancerNode(allNodes []nodes.Node) (nodes.Node, error) {
	// identify and validate external load balancer node
	loadBalancerNodes, err := SelectNodesByRole(
		allNodes,
		constants.ExternalLoadBalancerNodeRoleValue,
	)
	if err != nil {
		return nil, err
	}
	if len(loadBalancerNodes) < 1 {
		return nil, nil
	}
	if len(loadBalancerNodes) > 1 {
		return nil, errors.Errorf(
			"unexpected number of %s nodes %d",
			constants.ExternalLoadBalancerNodeRoleValue,
			len(loadBalancerNodes),
		)
	}
	return loadBalancerNodes[0], nil
}

// APIServerEndpointNode selects the node from allNodes which hosts the API Server endpoint
// This should be the control plane node if there is one control plane node, or a LoadBalancer otherwise.
// It returns an error if the node list is invalid (E.G. two control planes and no load balancer)
func APIServerEndpointNode(allNodes []nodes.Node) (nodes.Node, error) {
	if n, err := ExternalLoadBalancerNode(allNodes); err != nil {
		return nil, errors.Wrap(err, "failed to find api-server endpoint node")
	} else if n != nil {
		return n, nil
	}
	n, err := ControlPlaneNodes(allNodes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find api-server endpoint node")
	}
	if len(n) != 1 {
		return nil, errors.Errorf("expected one control plane node or a load balancer, not %d and none", len(n))
	}
	return n[0], nil
}

// ControlPlaneNodes returns all control plane nodes such that the first entry
// is the bootstrap control plane node
func ControlPlaneNodes(allNodes []nodes.Node) ([]nodes.Node, error) {
	controlPlaneNodes, err := SelectNodesByRole(
		allNodes,
		constants.ControlPlaneNodeRoleValue,
	)
	if err != nil {
		return nil, err
	}
	// pick the first by sorting
	// TODO(bentheelder): perhaps in the future we should mark this node
	// specially at container creation time
	sort.Slice(controlPlaneNodes, func(i, j int) bool {
		return strings.Compare(controlPlaneNodes[i].String(), controlPlaneNodes[j].String()) < 0
	})
	return controlPlaneNodes, nil
}

// BootstrapControlPlaneNode returns a handle to the bootstrap control plane node
// TODO(bentheelder): remove this. This node shouldn't be special (fix that first)
func BootstrapControlPlaneNode(allNodes []nodes.Node) (nodes.Node, error) {
	controlPlaneNodes, err := ControlPlaneNodes(allNodes)
	if err != nil {
		return nil, err
	}
	if len(controlPlaneNodes) < 1 {
		return nil, errors.Errorf(
			"expected at least one %s node",
			constants.ControlPlaneNodeRoleValue,
		)
	}
	return controlPlaneNodes[0], nil
}

// SecondaryControlPlaneNodes returns handles to the secondary
// control plane nodes and NOT the bootstrap control plane node
func SecondaryControlPlaneNodes(allNodes []nodes.Node) ([]nodes.Node, error) {
	controlPlaneNodes, err := ControlPlaneNodes(allNodes)
	if err != nil {
		return nil, err
	}
	if len(controlPlaneNodes) < 1 {
		return nil, errors.Errorf(
			"expected at least one %s node",
			constants.ControlPlaneNodeRoleValue,
		)
	}
	return controlPlaneNodes[1:], nil
}
