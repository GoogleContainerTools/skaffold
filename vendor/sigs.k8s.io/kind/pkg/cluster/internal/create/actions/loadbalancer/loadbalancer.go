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

// Package loadbalancer implements the load balancer configuration action
package loadbalancer

import (
	"fmt"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/internal/apis/config"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/loadbalancer"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
)

// Action implements and action for configuring and starting the
// external load balancer in front of the control-plane nodes.
type Action struct{}

// NewAction returns a new Action for configuring the load balancer
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// identify external load balancer node
	loadBalancerNode, err := nodeutils.ExternalLoadBalancerNode(allNodes)
	if err != nil {
		return err
	}

	// if there's no loadbalancer we're done
	if loadBalancerNode == nil {
		return nil
	}

	// otherwise notify the user
	ctx.Status.Start("Configuring the external load balancer ⚖️")
	defer ctx.Status.End(false)

	// collect info about the existing controlplane nodes
	var backendServers = map[string]string{}
	controlPlaneNodes, err := nodeutils.SelectNodesByRole(
		allNodes,
		constants.ControlPlaneNodeRoleValue,
	)
	if err != nil {
		return err
	}
	for _, n := range controlPlaneNodes {
		backendServers[n.String()] = fmt.Sprintf("%s:%d", n.String(), common.APIServerInternalPort)
	}

	loadbalancerConfigdata := &loadbalancer.ConfigData{
		ControlPlanePort: common.APIServerInternalPort,
		BackendServers:   backendServers,
		IPv6:             ctx.Config.Networking.IPFamily == config.IPv6Family,
	}

	// Generate the Dynamic Config strings
	ldsConfig, err := loadbalancer.Config(loadbalancerConfigdata, loadbalancer.ProxyLDSConfigTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to generate loadbalancer config data")
	}
	cdsConfig, err := loadbalancer.Config(loadbalancerConfigdata, loadbalancer.ProxyCDSConfigTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to generate loadbalancer config data")
	}

	// Atomic Update inside the container
	tmpLDS := loadbalancer.ProxyConfigPathLDS + ".tmp"
	tmpCDS := loadbalancer.ProxyConfigPathCDS + ".tmp"

	if err := nodeutils.WriteFile(loadBalancerNode, tmpLDS, ldsConfig); err != nil {
		return errors.Wrap(err, "failed to copy loadbalancer config to node")
	}
	if err := nodeutils.WriteFile(loadBalancerNode, tmpCDS, cdsConfig); err != nil {
		return errors.Wrap(err, "failed to copy loadbalancer config to node")
	}
	cmd := fmt.Sprintf("chmod 666 %s %s && mv %s %s && mv %s %s", tmpCDS, tmpLDS, tmpCDS, loadbalancer.ProxyConfigPathCDS, tmpLDS, loadbalancer.ProxyConfigPathLDS)
	if err := loadBalancerNode.Command("sh", "-c", cmd).Run(); err != nil {
		return errors.Wrap(err, "failed to reload envoy config")
	}

	ctx.Status.End(true)
	return nil
}
