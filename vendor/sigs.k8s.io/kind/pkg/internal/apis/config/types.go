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

/*
NOTE: unlike the public types these should not have serialization tags and
should stay 100% internal. These are used to pass around the processed public
config for internal usage.
*/

// Cluster contains kind cluster configuration
type Cluster struct {
	// The cluster name.
	// Optional, this will be overridden by --name / KIND_CLUSTER_NAME
	Name string

	// Nodes contains the list of nodes defined in the `kind` Cluster
	// If unset this will default to a single control-plane node
	// Note that if more than one control plane is specified, an external
	// control plane load balancer will be provisioned implicitly
	Nodes []Node

	/* Advanced fields */

	// Networking contains cluster wide network settings
	Networking Networking

	// FeatureGates contains a map of Kubernetes feature gates to whether they
	// are enabled. The feature gates specified here are passed to all Kubernetes components as flags or in config.
	//
	// https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
	FeatureGates map[string]bool

	// RuntimeConfig Keys and values are translated into --runtime-config values for kube-apiserver, separated by commas.
	//
	// Use this to enable alpha APIs.
	RuntimeConfig map[string]string

	// KubeadmConfigPatches are applied to the generated kubeadm config as
	// strategic merge patches to `kustomize build` internally
	// https://github.com/kubernetes/community/blob/a9cf5c8f3380bb52ebe57b1e2dbdec136d8dd484/contributors/devel/sig-api-machinery/strategic-merge-patch.md
	// This should be an inline yaml blob-string
	KubeadmConfigPatches []string

	// KubeadmConfigPatchesJSON6902 are applied to the generated kubeadm config
	// as patchesJson6902 to `kustomize build`
	KubeadmConfigPatchesJSON6902 []PatchJSON6902

	// ContainerdConfigPatches are applied to every node's containerd config
	// in the order listed.
	// These should be toml stringsto be applied as merge patches
	ContainerdConfigPatches []string

	// ContainerdConfigPatchesJSON6902 are applied to every node's containerd config
	// in the order listed.
	// These should be YAML or JSON formatting RFC 6902 JSON patches
	ContainerdConfigPatchesJSON6902 []string
}

// Node contains settings for a node in the `kind` Cluster.
// A node in kind config represent a container that will be provisioned with all the components
// required for the assigned role in the Kubernetes cluster
type Node struct {
	// Role defines the role of the node in the in the Kubernetes cluster
	// created by kind
	//
	// Defaults to "control-plane"
	Role NodeRole

	// Image is the node image to use when creating this node
	// If unset a default image will be used, see defaults.Image
	Image string

	// Labels are the labels with which the respective node will be labeled
	Labels map[string]string

	/* Advanced fields */

	// ExtraMounts describes additional mount points for the node container
	// These may be used to bind a hostPath
	ExtraMounts []Mount

	// ExtraPortMappings describes additional port mappings for the node container
	// binded to a host Port
	ExtraPortMappings []PortMapping

	// KubeadmConfigPatches are applied to the generated kubeadm config as
	// strategic merge patches to `kustomize build` internally
	// https://github.com/kubernetes/community/blob/a9cf5c8f3380bb52ebe57b1e2dbdec136d8dd484/contributors/devel/sig-api-machinery/strategic-merge-patch.md
	// This should be an inline yaml blob-string
	KubeadmConfigPatches []string

	// KubeadmConfigPatchesJSON6902 are applied to the generated kubeadm config
	// as patchesJson6902 to `kustomize build`
	KubeadmConfigPatchesJSON6902 []PatchJSON6902
}

// NodeRole defines possible role for nodes in a Kubernetes cluster managed by `kind`
type NodeRole string

const (
	// ControlPlaneRole identifies a node that hosts a Kubernetes control-plane.
	// NOTE: in single node clusters, control-plane nodes act also as a worker
	// nodes, in which case the taint will be removed. see:
	// https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#control-plane-node-isolation
	ControlPlaneRole NodeRole = "control-plane"
	// WorkerRole identifies a node that hosts a Kubernetes worker
	WorkerRole NodeRole = "worker"
)

// Networking contains cluster wide network settings
type Networking struct {
	// IPFamily is the network cluster model, currently it can be ipv4 or ipv6
	IPFamily ClusterIPFamily
	// APIServerPort is the listen port on the host for the Kubernetes API Server
	// Defaults to a random port on the host obtained by kind
	//
	// NOTE: if you set the special value of `-1` then the node backend
	// (docker, podman...) will be left to pick the port instead.
	// This is potentially useful for remote hosts, BUT it means when the container
	// is restarted it will be randomized. Leave this unset to allow kind to pick it.
	APIServerPort int32
	// APIServerAddress is the listen address on the host for the Kubernetes
	// API Server. This should be an IP address.
	//
	// Defaults to 127.0.0.1
	APIServerAddress string
	// PodSubnet is the CIDR used for pod IPs
	// kind will select a default if unspecified
	PodSubnet string
	// ServiceSubnet is the CIDR used for services VIPs
	// kind will select a default if unspecified
	ServiceSubnet string
	// If DisableDefaultCNI is true, kind will not install the default CNI setup.
	// Instead the user should install their own CNI after creating the cluster.
	DisableDefaultCNI bool
	// KubeProxyMode defines if kube-proxy should operate in iptables or ipvs mode
	KubeProxyMode ProxyMode
	// DNSSearch defines the DNS search domain to use for nodes. If not set, this will be inherited from the host.
	DNSSearch *[]string
}

// ClusterIPFamily defines cluster network IP family
type ClusterIPFamily string

const (
	// IPv4Family sets ClusterIPFamily to ipv4
	IPv4Family ClusterIPFamily = "ipv4"
	// IPv6Family sets ClusterIPFamily to ipv6
	IPv6Family ClusterIPFamily = "ipv6"
	// DualStackFamily sets ClusterIPFamily to dual
	DualStackFamily ClusterIPFamily = "dual"
)

// ProxyMode defines a proxy mode for kube-proxy
type ProxyMode string

const (
	// IPTablesProxyMode sets ProxyMode to iptables
	IPTablesProxyMode ProxyMode = "iptables"
	// IPVSProxyMode sets ProxyMode to ipvs
	IPVSProxyMode ProxyMode = "ipvs"
	// NoneProxyMode disables kube-proxy
	NoneProxyMode ProxyMode = "none"
)

// PatchJSON6902 represents an inline kustomize json 6902 patch
// https://tools.ietf.org/html/rfc6902
type PatchJSON6902 struct {
	// these fields specify the patch target resource
	Group   string
	Version string
	Kind    string
	// Patch should contain the contents of the json patch as a string
	Patch string
}

// Mount specifies a host volume to mount into a container.
// This is a close copy of the upstream cri Mount type
// see: k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2
// It additionally serializes the "propagation" field with the string enum
// names on disk as opposed to the int32 values, and the serialized field names
// have been made closer to core/v1 VolumeMount field names
// In yaml this looks like:
//
//	containerPath: /foo
//	hostPath: /bar
//	readOnly: true
//	selinuxRelabel: false
//	propagation: None
//
// Propagation may be one of: None, HostToContainer, Bidirectional
type Mount struct {
	// Path of the mount within the container.
	ContainerPath string
	// Path of the mount on the host. If the hostPath doesn't exist, then runtimes
	// should report error. If the hostpath is a symbolic link, runtimes should
	// follow the symlink and mount the real destination to container.
	HostPath string
	// If set, the mount is read-only.
	Readonly bool
	// If set, the mount needs SELinux relabeling.
	SelinuxRelabel bool
	// Requested propagation mode.
	Propagation MountPropagation
}

// PortMapping specifies a host port mapped into a container port.
// In yaml this looks like:
//
//	containerPort: 80
//	hostPort: 8000
//	listenAddress: 127.0.0.1
//	protocol: TCP
type PortMapping struct {
	// Port within the container.
	ContainerPort int32
	// Port on the host.
	//
	// If unset, a random port will be selected.
	//
	// NOTE: if you set the special value of `-1` then the node backend
	// (docker, podman...) will be left to pick the port instead.
	// This is potentially useful for remote hosts, BUT it means when the container
	// is restarted it will be randomized. Leave this unset to allow kind to pick it.
	HostPort int32
	// TODO: add protocol (tcp/udp) and port-ranges
	ListenAddress string
	// Protocol (TCP/UDP/SCTP)
	Protocol PortMappingProtocol
}

// MountPropagation represents an "enum" for mount propagation options,
// see also Mount.
type MountPropagation string

const (
	// MountPropagationNone specifies that no mount propagation
	// ("private" in Linux terminology).
	MountPropagationNone MountPropagation = "None"
	// MountPropagationHostToContainer specifies that mounts get propagated
	// from the host to the container ("rslave" in Linux).
	MountPropagationHostToContainer MountPropagation = "HostToContainer"
	// MountPropagationBidirectional specifies that mounts get propagated from
	// the host to the container and from the container to the host
	// ("rshared" in Linux).
	MountPropagationBidirectional MountPropagation = "Bidirectional"
)

// PortMappingProtocol represents an "enum" for port mapping protocol options,
// see also PortMapping.
type PortMappingProtocol string

const (
	// PortMappingProtocolTCP specifies TCP protocol
	PortMappingProtocolTCP PortMappingProtocol = "TCP"
	// PortMappingProtocolUDP specifies UDP protocol
	PortMappingProtocolUDP PortMappingProtocol = "UDP"
	// PortMappingProtocolSCTP specifies SCTP protocol
	PortMappingProtocolSCTP PortMappingProtocol = "SCTP"
)
