/*
Copyright 2021 The Kubernetes Authors.

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

// ClusterHasIPv6 returns true if the cluster should have IPv6 enabled due to either
// being IPv6 cluster family or Dual Stack
func ClusterHasIPv6(c *Cluster) bool {
	return c.Networking.IPFamily == IPv6Family || c.Networking.IPFamily == DualStackFamily
}

// ClusterHasImplicitLoadBalancer returns true if this cluster has an implicit api-server LoadBalancer
func ClusterHasImplicitLoadBalancer(c *Cluster) bool {
	controlPlanes := 0
	for _, node := range c.Nodes {
		if node.Role == ControlPlaneRole {
			controlPlanes++
		}
	}
	return controlPlanes > 1
}
