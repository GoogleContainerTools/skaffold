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

package v1alpha4

import (
	"sigs.k8s.io/kind/pkg/apis/config/defaults"
)

// SetDefaultsCluster sets uninitialized fields to their default value.
func SetDefaultsCluster(obj *Cluster) {
	// default to a one node cluster
	if len(obj.Nodes) == 0 {
		obj.Nodes = []Node{
			{
				Image: defaults.Image,
				Role:  ControlPlaneRole,
			},
		}
	}
	// default the nodes
	for i := range obj.Nodes {
		a := &obj.Nodes[i]
		SetDefaultsNode(a)
	}
	if obj.Networking.IPFamily == "" {
		obj.Networking.IPFamily = IPv4Family
	}
	// default to listening on 127.0.0.1:randomPort on ipv4
	// and [::1]:randomPort on ipv6
	if obj.Networking.APIServerAddress == "" {
		obj.Networking.APIServerAddress = "127.0.0.1"
		if obj.Networking.IPFamily == IPv6Family {
			obj.Networking.APIServerAddress = "::1"
		}
	}
	// default the pod CIDR
	if obj.Networking.PodSubnet == "" {
		obj.Networking.PodSubnet = "10.244.0.0/16"
		if obj.Networking.IPFamily == IPv6Family {
			// node-mask cidr default is /64 so we need a larger subnet, we use /56 following best practices
			// xref: https://www.ripe.net/publications/docs/ripe-690#4--size-of-end-user-prefix-assignment---48---56-or-something-else-
			obj.Networking.PodSubnet = "fd00:10:244::/56"
		}
		if obj.Networking.IPFamily == DualStackFamily {
			obj.Networking.PodSubnet = "10.244.0.0/16,fd00:10:244::/56"
		}
	}
	// default the service CIDR using a different subnet than kubeadm default
	// https://github.com/kubernetes/kubernetes/blob/746404f82a28e55e0b76ffa7e40306fb88eb3317/cmd/kubeadm/app/apis/kubeadm/v1beta2/defaults.go#L32
	// Note: kubeadm is using a /12 subnet, that may allocate a 2^20 bitmap in etcd
	// we allocate a /16 subnet that allows 65535 services (current Kubernetes tested limit is O(10k) services)
	if obj.Networking.ServiceSubnet == "" {
		obj.Networking.ServiceSubnet = "10.96.0.0/16"
		if obj.Networking.IPFamily == IPv6Family {
			obj.Networking.ServiceSubnet = "fd00:10:96::/112"
		}
		if obj.Networking.IPFamily == DualStackFamily {
			obj.Networking.ServiceSubnet = "10.96.0.0/16,fd00:10:96::/112"
		}
	}
	// default the KubeProxyMode using iptables as it's already the default
	if obj.Networking.KubeProxyMode == "" {
		obj.Networking.KubeProxyMode = IPTablesProxyMode
	}
}

// SetDefaultsNode sets uninitialized fields to their default value.
func SetDefaultsNode(obj *Node) {
	if obj.Image == "" {
		obj.Image = defaults.Image
	}

	if obj.Role == "" {
		obj.Role = ControlPlaneRole
	}
}
