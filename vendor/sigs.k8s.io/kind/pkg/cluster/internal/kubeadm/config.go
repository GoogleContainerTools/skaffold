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

package kubeadm

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/google/safetext/yamltemplate"

	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/version"
)

// ConfigData is supplied to the kubeadm config template, with values populated
// by the cluster package
type ConfigData struct {
	ClusterName       string
	KubernetesVersion string
	// The ControlPlaneEndpoint, that is the address of the external loadbalancer
	// if defined or the bootstrap node
	ControlPlaneEndpoint string
	// The Local API Server port
	APIBindPort int
	// The API server external listen IP (which we will port forward)
	APIServerAddress string

	// this should really be used for the --provider-id flag
	// ideally cluster config should not depend on the node backend otherwise ...
	NodeProvider string

	// ControlPlane flag specifies the node belongs to the control plane
	ControlPlane bool
	// The IP address or comma separated list IP addresses of of the node
	NodeAddress string
	// The name for the node (not the address)
	NodeName string

	// The Token for TLS bootstrap
	Token string

	// KubeProxyMode defines the kube-proxy mode between iptables, ipvs or nftables
	KubeProxyMode string
	// The subnet used for pods
	PodSubnet string
	// The subnet used for services
	ServiceSubnet string

	// Kubernetes FeatureGates
	FeatureGates map[string]bool

	// Kubernetes API Server RuntimeConfig
	RuntimeConfig map[string]string

	// IPFamily of the cluster, it can be IPv4, IPv6 or DualStack
	IPFamily config.ClusterIPFamily

	// Labels are the labels, in the format "key1=val1,key2=val2", with which the respective node will be labeled
	NodeLabels string

	// RootlessProvider is true if kind is running with rootless mode
	RootlessProvider bool

	// DerivedConfigData contains fields computed from the other fields for use
	// in the config templates and should only be populated by calling Derive()
	DerivedConfigData
}

// DerivedConfigData fields are automatically derived by
// ConfigData.Derive if they are not specified / zero valued
type DerivedConfigData struct {
	// AdvertiseAddress is the first address in NodeAddress
	AdvertiseAddress string
	// DockerStableTag is automatically derived from KubernetesVersion
	DockerStableTag string
	// SortedFeatureGates allows us to iterate FeatureGates deterministically
	SortedFeatureGates []FeatureGate
	// FeatureGatesString is of the form `Foo=true,Baz=false`
	FeatureGatesString string
	// RuntimeConfigString is of the form `Foo=true,Baz=false`
	RuntimeConfigString string
	// KubeadmFeatureGates contains Kubeadm only feature gates
	KubeadmFeatureGates map[string]bool
	// IPv4 values take precedence over IPv6 by default, if true set IPv6 default values
	IPv6 bool
	// kubelet cgroup driver, based on kubernetes version
	CgroupDriver string
	// JoinSkipPhases are the skipPhases values for the JoinConfiguration.
	JoinSkipPhases []string
	// InitSkipPhases are the skipPhases values for the InitConfiguration.
	InitSkipPhases []string
}

type FeatureGate struct {
	Name  string
	Value bool
}

// Derive automatically derives DockerStableTag if not specified
func (c *ConfigData) Derive() {
	// default cgroup driver
	// TODO: refactor and move all deriving logic to this method
	c.CgroupDriver = "systemd"

	// get the first address to use it as the API advertised address
	c.AdvertiseAddress = strings.Split(c.NodeAddress, ",")[0]

	if c.DockerStableTag == "" {
		c.DockerStableTag = strings.Replace(c.KubernetesVersion, "+", "_", -1)
	}

	// get the IP addresses family for defaulting components
	c.IPv6 = c.IPFamily == config.IPv6Family

	// get sorted list of FeatureGate keys
	featureGateKeys := make([]string, 0, len(c.FeatureGates))
	for k := range c.FeatureGates {
		featureGateKeys = append(featureGateKeys, k)
	}
	sort.Strings(featureGateKeys)

	// create a sorted key=value,... string of FeatureGates
	c.SortedFeatureGates = make([]FeatureGate, 0, len(c.FeatureGates))
	featureGates := make([]string, 0, len(c.FeatureGates))
	for _, k := range featureGateKeys {
		v := c.FeatureGates[k]
		featureGates = append(featureGates, fmt.Sprintf("%s=%t", k, v))
		c.SortedFeatureGates = append(c.SortedFeatureGates, FeatureGate{
			Name:  k,
			Value: v,
		})
	}
	c.FeatureGatesString = strings.Join(featureGates, ",")

	// create a sorted key=value,... string of RuntimeConfig
	// first get sorted list of FeatureGate keys
	runtimeConfigKeys := make([]string, 0, len(c.RuntimeConfig))
	for k := range c.RuntimeConfig {
		runtimeConfigKeys = append(runtimeConfigKeys, k)
	}
	sort.Strings(runtimeConfigKeys)
	// stringify
	var runtimeConfig []string
	for _, k := range runtimeConfigKeys {
		v := c.RuntimeConfig[k]
		// TODO: do we need to quote / escape these in the future?
		// Currently runtime config is in practice booleans, no special characters
		runtimeConfig = append(runtimeConfig, fmt.Sprintf("%s=%s", k, v))
	}
	c.RuntimeConfigString = strings.Join(runtimeConfig, ",")

	// Skip preflight to avoid pulling images.
	// Kind pre-pulls images and preflight may conflict with that.
	// requires kubeadm 1.22+
	c.JoinSkipPhases = []string{"preflight"}
	c.InitSkipPhases = []string{"preflight"}
	if c.KubeProxyMode == string(config.NoneProxyMode) {
		c.InitSkipPhases = append(c.InitSkipPhases, "addon/kube-proxy")
	}
}

// See docs for these APIs at:
// https://godoc.org/k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm#pkg-subdirectories
// EG:
// https://godoc.org/k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1

// ConfigTemplateBetaV2 is the kubeadm config template for API version v1beta2
const ConfigTemplateBetaV2 = `# config generated by kind
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
kubernetesVersion: {{.KubernetesVersion}}
clusterName: "{{.ClusterName}}"
{{ if .KubeadmFeatureGates}}featureGates:
{{ range $key, $value := .KubeadmFeatureGates }}
  "{{ (StructuralData $key) }}": {{ $value }}
{{end}}{{end}}
controlPlaneEndpoint: "{{ .ControlPlaneEndpoint }}"
# on docker for mac we have to expose the api server via port forward,
# so we need to ensure the cert is valid for localhost so we can talk
# to the cluster after rewriting the kubeconfig to point to localhost
apiServer:
  certSANs: [localhost, "{{.APIServerAddress}}"]
  extraArgs:
    "runtime-config": "{{ .RuntimeConfigString }}"
{{ if .FeatureGates }}
    "feature-gates": "{{ .FeatureGatesString }}"
{{ end}}
controllerManager:
  extraArgs:
{{ if .FeatureGates }}
    "feature-gates": "{{ .FeatureGatesString }}"
{{ end }}
    enable-hostpath-provisioner: "true"
    # configure ipv6 default addresses for IPv6 clusters
    {{ if .IPv6 -}}
    bind-address: "::"
    {{- end }}
scheduler:
  extraArgs:
{{ if .FeatureGates }}
    "feature-gates": "{{ .FeatureGatesString }}"
{{ end }}
    # configure ipv6 default addresses for IPv6 clusters
    {{ if .IPv6 -}}
    bind-address: "::1"
    {{- end }}
networking:
  podSubnet: "{{ .PodSubnet }}"
  serviceSubnet: "{{ .ServiceSubnet }}"
---
apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
metadata:
  name: config
# we use a well know token for TLS bootstrap
bootstrapTokens:
- token: "{{ .Token }}"
# we use a well know port for making the API server discoverable inside docker network. 
# from the host machine such port will be accessible via a random local port instead.
localAPIEndpoint:
  advertiseAddress: "{{ .AdvertiseAddress }}"
  bindPort: {{.APIBindPort}}
nodeRegistration:
  criSocket: "unix:///run/containerd/containerd.sock"
  kubeletExtraArgs:
    node-ip: "{{ .NodeAddress }}"
    provider-id: "kind://{{.NodeProvider}}/{{.ClusterName}}/{{.NodeName}}"
    node-labels: "{{ .NodeLabels }}"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
metadata:
  name: config
{{ if .ControlPlane -}}
controlPlane:
  localAPIEndpoint:
    advertiseAddress: "{{ .AdvertiseAddress }}"
    bindPort: {{.APIBindPort}}
{{- end }}
nodeRegistration:
  criSocket: "unix:///run/containerd/containerd.sock"
  kubeletExtraArgs:
    node-ip: "{{ .NodeAddress }}"
    provider-id: "kind://{{.NodeProvider}}/{{.ClusterName}}/{{.NodeName}}"
    node-labels: "{{ .NodeLabels }}"
discovery:
  bootstrapToken:
    apiServerEndpoint: "{{ .ControlPlaneEndpoint }}"
    token: "{{ .Token }}"
    unsafeSkipCAVerification: true
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
metadata:
  name: config
cgroupDriver: {{ .CgroupDriver }}
cgroupRoot: /kubelet
failSwapOn: false
# configure ipv6 addresses in IPv6 mode
{{ if .IPv6 -}}
address: "::"
healthzBindAddress: "::"
{{- end }}
# disable disk resource management by default
# kubelet will see the host disk that the inner container runtime
# is ultimately backed by and attempt to recover disk space. we don't want that.
imageGCHighThresholdPercent: 100
evictionHard:
  nodefs.available: "0%"
  nodefs.inodesFree: "0%"
  imagefs.available: "0%"
{{if .FeatureGates}}featureGates:
{{ range $index, $gate := .SortedFeatureGates }}
  "{{ (StructuralData $gate.Name) }}": {{ $gate.Value }}
{{end}}{{end}}
{{if ne .KubeProxyMode "none"}}
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
mode: "{{ .KubeProxyMode }}"
{{if .FeatureGates}}featureGates:
{{ range $index, $gate := .SortedFeatureGates }}
  "{{ (StructuralData $gate.Name) }}": {{ $gate.Value }}
{{end}}{{end}}
iptables:
  minSyncPeriod: 1s
conntrack:
# Skip setting sysctl value "net.netfilter.nf_conntrack_max"
# It is a global variable that affects other namespaces
  maxPerCore: 0
# Set sysctl value "net.netfilter.nf_conntrack_tcp_be_liberal"
# for nftables proxy (theoretically for kernels older than 6.1)
# xref: https://github.com/kubernetes/kubernetes/issues/117924
{{if and (eq .KubeProxyMode "nftables") (not .RootlessProvider)}}
  tcpBeLiberal: true
{{end}}
{{if .RootlessProvider}}
# Skip setting "net.netfilter.nf_conntrack_tcp_timeout_established"
  tcpEstablishedTimeout: 0s
# Skip setting "net.netfilter.nf_conntrack_tcp_timeout_close"
  tcpCloseWaitTimeout: 0s
{{end}}{{end}}
`

// ConfigTemplateBetaV3 is the kubeadm config template for API version v1beta3
const ConfigTemplateBetaV3 = `# config generated by kind
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
metadata:
  name: config
kubernetesVersion: {{.KubernetesVersion}}
clusterName: "{{.ClusterName}}"
{{ if .KubeadmFeatureGates}}featureGates:
{{ range $key, $value := .KubeadmFeatureGates }}
  "{{ (StructuralData $key) }}": {{ $value }}
{{end}}{{end}}
controlPlaneEndpoint: "{{ .ControlPlaneEndpoint }}"
# on docker for mac we have to expose the api server via port forward,
# so we need to ensure the cert is valid for localhost so we can talk
# to the cluster after rewriting the kubeconfig to point to localhost
apiServer:
  certSANs: [localhost, "{{.APIServerAddress}}"]
  extraArgs:
    "runtime-config": "{{ .RuntimeConfigString }}"
{{ if .FeatureGates }}
    "feature-gates": "{{ .FeatureGatesString }}"
{{ end}}
controllerManager:
  extraArgs:
{{ if .FeatureGates }}
    "feature-gates": "{{ .FeatureGatesString }}"
{{ end }}
    enable-hostpath-provisioner: "true"
    # configure ipv6 default addresses for IPv6 clusters
    {{ if .IPv6 -}}
    bind-address: "::"
    {{- end }}
scheduler:
  extraArgs:
{{ if .FeatureGates }}
    "feature-gates": "{{ .FeatureGatesString }}"
{{ end }}
    # configure ipv6 default addresses for IPv6 clusters
    {{ if .IPv6 -}}
    bind-address: "::1"
    {{- end }}
networking:
  podSubnet: "{{ .PodSubnet }}"
  serviceSubnet: "{{ .ServiceSubnet }}"
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
metadata:
  name: config
# we use a well know token for TLS bootstrap
bootstrapTokens:
- token: "{{ .Token }}"
# we use a well know port for making the API server discoverable inside docker network. 
# from the host machine such port will be accessible via a random local port instead.
localAPIEndpoint:
  advertiseAddress: "{{ .AdvertiseAddress }}"
  bindPort: {{.APIBindPort}}
nodeRegistration:
  criSocket: "unix:///run/containerd/containerd.sock"
  kubeletExtraArgs:
    node-ip: "{{ .NodeAddress }}"
    provider-id: "kind://{{.NodeProvider}}/{{.ClusterName}}/{{.NodeName}}"
    node-labels: "{{ .NodeLabels }}"
{{ if .InitSkipPhases -}}
skipPhases:
  {{- range $phase := .InitSkipPhases }}
  - "{{ $phase }}"
  {{- end }}
{{- end }}
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
metadata:
  name: config
{{ if .ControlPlane -}}
controlPlane:
  localAPIEndpoint:
    advertiseAddress: "{{ .AdvertiseAddress }}"
    bindPort: {{.APIBindPort}}
{{- end }}
nodeRegistration:
  criSocket: "unix:///run/containerd/containerd.sock"
  kubeletExtraArgs:
    node-ip: "{{ .NodeAddress }}"
    provider-id: "kind://{{.NodeProvider}}/{{.ClusterName}}/{{.NodeName}}"
    node-labels: "{{ .NodeLabels }}"
discovery:
  bootstrapToken:
    apiServerEndpoint: "{{ .ControlPlaneEndpoint }}"
    token: "{{ .Token }}"
    unsafeSkipCAVerification: true
{{ if .JoinSkipPhases -}}
skipPhases:
  {{ range $phase := .JoinSkipPhases -}}
  - "{{ $phase }}"
  {{- end }}
{{- end }}
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
metadata:
  name: config
cgroupDriver: {{ .CgroupDriver }}
cgroupRoot: /kubelet
failSwapOn: false
# configure ipv6 addresses in IPv6 mode
{{ if .IPv6 -}}
address: "::"
healthzBindAddress: "::"
{{- end }}
# disable disk resource management by default
# kubelet will see the host disk that the inner container runtime
# is ultimately backed by and attempt to recover disk space. we don't want that.
imageGCHighThresholdPercent: 100
evictionHard:
  nodefs.available: "0%"
  nodefs.inodesFree: "0%"
  imagefs.available: "0%"
{{if .FeatureGates}}featureGates:
{{ range $index, $gate := .SortedFeatureGates }}
  "{{ (StructuralData $gate.Name) }}": {{ $gate.Value }}
{{end}}{{end}}
{{if ne .KubeProxyMode "none"}}
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
mode: "{{ .KubeProxyMode }}"
{{if .FeatureGates}}featureGates:
{{ range $index, $gate := .SortedFeatureGates }}
  "{{ (StructuralData $gate.Name) }}": {{ $gate.Value }}
{{end}}{{end}}
iptables:
  minSyncPeriod: 1s
conntrack:
# Skip setting sysctl value "net.netfilter.nf_conntrack_max"
# It is a global variable that affects other namespaces
  maxPerCore: 0
# Set sysctl value "net.netfilter.nf_conntrack_tcp_be_liberal"
# for nftables proxy (theoretically for kernels older than 6.1)
# xref: https://github.com/kubernetes/kubernetes/issues/117924
{{if and (eq .KubeProxyMode "nftables") (not .RootlessProvider)}}
  tcpBeLiberal: true
{{end}}
{{if .RootlessProvider}}
# Skip setting "net.netfilter.nf_conntrack_tcp_timeout_established"
  tcpEstablishedTimeout: 0s
# Skip setting "net.netfilter.nf_conntrack_tcp_timeout_close"
  tcpCloseWaitTimeout: 0s
{{end}}{{end}}
`

// Config returns a kubeadm config generated from config data, in particular
// the kubernetes version
func Config(data ConfigData) (config string, err error) {
	ver, err := version.ParseGeneric(data.KubernetesVersion)
	if err != nil {
		return "", err
	}

	// ensure featureGates is non-nil, as we may add entries
	if data.FeatureGates == nil {
		data.FeatureGates = make(map[string]bool)
	}

	if data.RootlessProvider {
		if ver.LessThan(version.MustParseSemantic("v1.22.0")) {
			// rootless kind v0.12.x supports Kubernetes v1.22 with KubeletInUserNamespace gate.
			// rootless kind v0.11.x supports older Kubernetes with fake procfs.
			return "", errors.Errorf("version %q is not compatible with rootless provider (hint: kind v0.11.x may work with this version)", ver)
		}
		data.FeatureGates["KubeletInUserNamespace"] = true
	}

	// assume the latest API version, then fallback if the k8s version is too low
	templateSource := ConfigTemplateBetaV3
	if ver.LessThan(version.MustParseSemantic("v1.23.0")) {
		templateSource = ConfigTemplateBetaV2
	}

	t, err := yamltemplate.New("kubeadm-config").Parse(templateSource)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}

	// derive any automatic fields if not supplied
	data.Derive()

	// Kubeadm has its own feature-gate for dual stack
	// we need to enable it for Kubernetes version 1.20 only
	// dual-stack is only supported in 1.20+
	// TODO: remove this when 1.20 is EOL or we no longer support
	// dual-stack for 1.20 in KIND
	if ver.LessThan(version.MustParseSemantic("v1.21.0")) &&
		ver.AtLeast(version.MustParseSemantic("v1.20.0")) {
		data.KubeadmFeatureGates = make(map[string]bool)
		data.KubeadmFeatureGates["IPv6DualStack"] = true
	}

	// before 1.24 kind uses cgroupfs
	// after 1.24 kind uses systemd starting in kind v0.13.0
	// before kind v0.13.0 kubernetes 1.24 wasn't released yet
	if ver.LessThan(version.MustParseSemantic("v1.24.0")) {
		data.CgroupDriver = "cgroupfs"
	}

	// execute the template
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}
	return buff.String(), nil
}
