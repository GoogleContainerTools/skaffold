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

// Package config implements the kubeadm config action
package config

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeadm"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/patch"
)

// Action implements action for creating the node config files
type Action struct{}

// NewAction returns a new action for creating the config files
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Writing configuration 📜")
	defer ctx.Status.End(false)

	providerInfo, err := ctx.Provider.Info()
	if err != nil {
		return err
	}

	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	controlPlaneEndpoint, err := ctx.Provider.GetAPIServerInternalEndpoint(ctx.Config.Name)
	if err != nil {
		return err
	}

	// create kubeadm init config
	fns := []func() error{}

	provider := fmt.Sprintf("%s", ctx.Provider)
	configData := kubeadm.ConfigData{
		NodeProvider:         provider,
		ClusterName:          ctx.Config.Name,
		ControlPlaneEndpoint: controlPlaneEndpoint,
		APIBindPort:          common.APIServerInternalPort,
		APIServerAddress:     ctx.Config.Networking.APIServerAddress,
		Token:                kubeadm.Token,
		PodSubnet:            ctx.Config.Networking.PodSubnet,
		KubeProxyMode:        string(ctx.Config.Networking.KubeProxyMode),
		ServiceSubnet:        ctx.Config.Networking.ServiceSubnet,
		ControlPlane:         true,
		IPFamily:             ctx.Config.Networking.IPFamily,
		FeatureGates:         ctx.Config.FeatureGates,
		RuntimeConfig:        ctx.Config.RuntimeConfig,
		RootlessProvider:     providerInfo.Rootless,
	}

	kubeadmConfigPlusPatches := func(node nodes.Node, data kubeadm.ConfigData) func() error {
		return func() error {
			data.NodeName = node.String()
			kubeadmConfig, err := getKubeadmConfig(ctx.Config, data, node, provider)
			if err != nil {
				// TODO(bentheelder): logging here
				return errors.Wrap(err, "failed to generate kubeadm config content")
			}

			ctx.Logger.V(2).Infof("Using the following kubeadm config for node %s:\n%s", node.String(), kubeadmConfig)
			return writeKubeadmConfig(kubeadmConfig, node)
		}
	}

	// create the kubeadm join configuration for the kubernetes cluster nodes only
	kubeNodes, err := nodeutils.InternalNodes(allNodes)
	if err != nil {
		return err
	}

	for _, node := range kubeNodes {
		node := node             // capture loop variable
		configData := configData // copy config data
		fns = append(fns, kubeadmConfigPlusPatches(node, configData))
	}

	// Create the kubeadm config in all nodes concurrently
	if err := errors.UntilErrorConcurrent(fns); err != nil {
		return err
	}

	// if we have containerd config, patch all the nodes concurrently
	if len(ctx.Config.ContainerdConfigPatches) > 0 || len(ctx.Config.ContainerdConfigPatchesJSON6902) > 0 {
		fns := make([]func() error, len(kubeNodes))
		for i, node := range kubeNodes {
			node := node // capture loop variable
			fns[i] = func() error {
				// read and patch the config
				const containerdConfigPath = "/etc/containerd/config.toml"
				var buff bytes.Buffer
				if err := node.Command("cat", containerdConfigPath).SetStdout(&buff).Run(); err != nil {
					return errors.Wrap(err, "failed to read containerd config from node")
				}
				patched, err := patch.TOML(buff.String(), ctx.Config.ContainerdConfigPatches, ctx.Config.ContainerdConfigPatchesJSON6902)
				if err != nil {
					return errors.Wrap(err, "failed to patch containerd config")
				}
				if err := nodeutils.WriteFile(node, containerdConfigPath, patched); err != nil {
					return errors.Wrap(err, "failed to write patched containerd config")
				}
				// restart containerd now that we've re-configured it
				// skip if containerd is not running
				if err := node.Command("bash", "-c", `! pgrep --exact containerd || systemctl restart containerd`).Run(); err != nil {
					return errors.Wrap(err, "failed to restart containerd after patching config")
				}
				return nil
			}
		}
		if err := errors.UntilErrorConcurrent(fns); err != nil {
			return err
		}
	}

	// mark success
	ctx.Status.End(true)
	return nil
}

// getKubeadmConfig generates the kubeadm config contents for the cluster
// by running data through the template and applying patches as needed.
func getKubeadmConfig(cfg *config.Cluster, data kubeadm.ConfigData, node nodes.Node, provider string) (path string, err error) {
	kubeVersion, err := nodeutils.KubeVersion(node)
	if err != nil {
		// TODO(bentheelder): logging here
		return "", errors.Wrap(err, "failed to get kubernetes version from node")
	}
	data.KubernetesVersion = kubeVersion

	// TODO: gross hack!
	// identify node in config by matching name (since these are named in order)
	// we should really just streamline the bootstrap code and maintain
	// this mapping ... something for the next major refactor
	var configNode *config.Node
	namer := common.MakeNodeNamer("")
	for i := range cfg.Nodes {
		n := &cfg.Nodes[i]
		nodeSuffix := namer(string(n.Role))
		if strings.HasSuffix(node.String(), nodeSuffix) {
			configNode = n
		}
	}
	if configNode == nil {
		return "", errors.Errorf("failed to match node %q to config", node.String())
	}

	// get the node ip address
	nodeAddress, nodeAddressIPv6, err := node.IP()
	if err != nil {
		return "", errors.Wrap(err, "failed to get IP for node")
	}

	data.NodeAddress = nodeAddress
	// configure the right protocol addresses
	if cfg.Networking.IPFamily == config.IPv6Family || cfg.Networking.IPFamily == config.DualStackFamily {
		if ip := net.ParseIP(nodeAddressIPv6); ip.To16() == nil {
			return "", errors.Errorf("failed to get IPv6 address for node %s; is %s configured to use IPv6 correctly?", node.String(), provider)
		}
		data.NodeAddress = nodeAddressIPv6
		if cfg.Networking.IPFamily == config.DualStackFamily {
			// order matters since the nodeAddress will be used later to configure the apiserver advertise address
			// Ref: #2484
			primaryServiceSubnet := strings.Split(cfg.Networking.ServiceSubnet, ",")[0]
			ip, _, err := net.ParseCIDR(primaryServiceSubnet)
			if err != nil {
				return "", fmt.Errorf("failed to parse primary Service Subnet %s (%s): %w", primaryServiceSubnet, cfg.Networking.ServiceSubnet, err)
			}
			if ip.To4() != nil {
				data.NodeAddress = fmt.Sprintf("%s,%s", nodeAddress, nodeAddressIPv6)
			} else {
				data.NodeAddress = fmt.Sprintf("%s,%s", nodeAddressIPv6, nodeAddress)
			}
		}
	}

	// configure the node labels
	if len(configNode.Labels) > 0 {
		data.NodeLabels = hashMapLabelsToCommaSeparatedLabels(configNode.Labels)
	}

	// set the node role
	data.ControlPlane = string(configNode.Role) == constants.ControlPlaneNodeRoleValue

	// generate the config contents
	cf, err := kubeadm.Config(data)
	if err != nil {
		return "", err
	}

	clusterPatches, clusterJSONPatches := allPatchesFromConfig(cfg)
	// apply cluster-level patches first
	patchedConfig, err := patch.KubeYAML(cf, clusterPatches, clusterJSONPatches)
	if err != nil {
		return "", err
	}

	// if needed, apply current node's patches
	if len(configNode.KubeadmConfigPatches) > 0 || len(configNode.KubeadmConfigPatchesJSON6902) > 0 {
		patchedConfig, err = patch.KubeYAML(patchedConfig, configNode.KubeadmConfigPatches, configNode.KubeadmConfigPatchesJSON6902)
		if err != nil {
			return "", err
		}
	}

	// fix all the patches to have name metadata matching the generated config
	return removeMetadata(patchedConfig), nil
}

// trims out the metadata.name we put in the config for kustomize matching,
// kubeadm will complain about this otherwise
func removeMetadata(kustomized string) string {
	return strings.Replace(
		kustomized,
		`metadata:
  name: config
`,
		"",
		-1,
	)
}

func allPatchesFromConfig(cfg *config.Cluster) (patches []string, jsonPatches []config.PatchJSON6902) {
	return cfg.KubeadmConfigPatches, cfg.KubeadmConfigPatchesJSON6902
}

// writeKubeadmConfig writes the kubeadm configuration in the specified node
func writeKubeadmConfig(kubeadmConfig string, node nodes.Node) error {
	// copy the config to the node
	if err := nodeutils.WriteFile(node, "/kind/kubeadm.conf", kubeadmConfig); err != nil {
		// TODO(bentheelder): logging here
		return errors.Wrap(err, "failed to copy kubeadm config to node")
	}

	return nil
}

// hashMapLabelsToCommaSeparatedLabels converts labels in hashmap form to labels in a comma-separated string form like "key1=value1,key2=value2"
func hashMapLabelsToCommaSeparatedLabels(labels map[string]string) string {
	output := ""
	for key, value := range labels {
		output += fmt.Sprintf("%s=%s,", key, value)
	}
	return strings.TrimSuffix(output, ",") // remove the last character (comma) in the output string
}
