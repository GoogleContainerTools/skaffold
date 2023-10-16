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

package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"

	"sigs.k8s.io/kind/pkg/cluster/internal/loadbalancer"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// planCreation creates a slice of funcs that will create the containers
func planCreation(cfg *config.Cluster, networkName string) (createContainerFuncs []func() error, err error) {
	// these apply to all container creation
	nodeNamer := common.MakeNodeNamer(cfg.Name)
	names := make([]string, len(cfg.Nodes))
	for i, node := range cfg.Nodes {
		name := nodeNamer(string(node.Role)) // name the node
		names[i] = name
	}
	haveLoadbalancer := config.ClusterHasImplicitLoadBalancer(cfg)
	if haveLoadbalancer {
		names = append(names, nodeNamer(constants.ExternalLoadBalancerNodeRoleValue))
	}
	genericArgs, err := commonArgs(cfg, networkName, names)
	if err != nil {
		return nil, err
	}

	// only the external LB should reflect the port if we have multiple control planes
	apiServerPort := cfg.Networking.APIServerPort
	apiServerAddress := cfg.Networking.APIServerAddress
	if config.ClusterHasImplicitLoadBalancer(cfg) {
		// TODO: picking ports locally is less than ideal with a remote runtime
		// (does podman have this?)
		// but this is supposed to be an implementation detail and NOT picking
		// them breaks host reboot ...
		// For now remote podman + multi control plane is not supported
		apiServerPort = 0              // replaced with random ports
		apiServerAddress = "127.0.0.1" // only the LB needs to be non-local
		// only for IPv6 only clusters
		if cfg.Networking.IPFamily == config.IPv6Family {
			apiServerAddress = "::1" // only the LB needs to be non-local
		}
		// plan loadbalancer node
		name := names[len(names)-1]
		createContainerFuncs = append(createContainerFuncs, func() error {
			args, err := runArgsForLoadBalancer(cfg, name, genericArgs)
			if err != nil {
				return err
			}
			return createContainer(name, args)
		})
	}

	// plan normal nodes
	for i, node := range cfg.Nodes {
		node := node.DeepCopy() // copy so we can modify
		name := names[i]

		// fixup relative paths, podman can only handle absolute paths
		for i := range node.ExtraMounts {
			hostPath := node.ExtraMounts[i].HostPath
			absHostPath, err := filepath.Abs(hostPath)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to resolve absolute path for hostPath: %q", hostPath)
			}
			node.ExtraMounts[i].HostPath = absHostPath
		}

		// plan actual creation based on role
		switch node.Role {
		case config.ControlPlaneRole:
			createContainerFuncs = append(createContainerFuncs, func() error {
				node.ExtraPortMappings = append(node.ExtraPortMappings,
					config.PortMapping{
						ListenAddress: apiServerAddress,
						HostPort:      apiServerPort,
						ContainerPort: common.APIServerInternalPort,
					},
				)
				args, err := runArgsForNode(node, cfg.Networking.IPFamily, name, genericArgs)
				if err != nil {
					return err
				}
				return createContainerWithWaitUntilSystemdReachesMultiUserSystem(name, args)
			})
		case config.WorkerRole:
			createContainerFuncs = append(createContainerFuncs, func() error {
				args, err := runArgsForNode(node, cfg.Networking.IPFamily, name, genericArgs)
				if err != nil {
					return err
				}
				return createContainerWithWaitUntilSystemdReachesMultiUserSystem(name, args)
			})
		default:
			return nil, errors.Errorf("unknown node role: %q", node.Role)
		}
	}
	return createContainerFuncs, nil
}

// commonArgs computes static arguments that apply to all containers
func commonArgs(cfg *config.Cluster, networkName string, nodeNames []string) ([]string, error) {
	// standard arguments all nodes containers need, computed once
	args := []string{
		"--detach",           // run the container detached
		"--tty",              // allocate a tty for entrypoint logs
		"--net", networkName, // attach to its own network
		// label the node with the cluster ID
		"--label", fmt.Sprintf("%s=%s", clusterLabelKey, cfg.Name),
		// specify container implementation to systemd
		"-e", "container=podman",
		// this is the default in cgroupsv2 but not in v1
		"--cgroupns=private",
	}

	// enable IPv6 if necessary
	if config.ClusterHasIPv6(cfg) {
		args = append(args, "--sysctl=net.ipv6.conf.all.disable_ipv6=0", "--sysctl=net.ipv6.conf.all.forwarding=1")
	}

	// pass proxy environment variables
	proxyEnv, err := getProxyEnv(cfg, networkName, nodeNames)
	if err != nil {
		return nil, errors.Wrap(err, "proxy setup error")
	}
	for key, val := range proxyEnv {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	// handle Podman on Btrfs or ZFS same as we do with Docker
	// https://github.com/kubernetes-sigs/kind/issues/1416#issuecomment-606514724
	if mountDevMapper() {
		args = append(args, "--volume", "/dev/mapper:/dev/mapper")
	}

	// rootless: use fuse-overlayfs by default
	// https://github.com/kubernetes-sigs/kind/issues/2275
	if mountFuse() {
		args = append(args, "--device", "/dev/fuse")
	}

	if cfg.Networking.DNSSearch != nil {
		args = append(args, "-e", "KIND_DNS_SEARCH="+strings.Join(*cfg.Networking.DNSSearch, " "))
	}

	return args, nil
}

func runArgsForNode(node *config.Node, clusterIPFamily config.ClusterIPFamily, name string, args []string) ([]string, error) {
	// Pre-create anonymous volumes to enable specifying mount options
	// during container run time
	varVolume, err := createAnonymousVolume(name)
	if err != nil {
		return nil, err
	}

	args = append([]string{
		"--hostname", name, // make hostname match container name
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", nodeRoleLabelKey, node.Role),
		// running containers in a container requires privileged
		// NOTE: we could try to replicate this with --cap-add, and use less
		// privileges, but this flag also changes some mounts that are necessary
		// including some ones podman would otherwise do by default.
		// for now this is what we want. in the future we may revisit this.
		"--privileged",
		// runtime temporary storage
		"--tmpfs", "/tmp", // various things depend on working /tmp
		"--tmpfs", "/run", // systemd wants a writable /run
		// runtime persistent storage
		// this ensures that E.G. pods, logs etc. are not on the container
		// filesystem, which is not only better for performance, but allows
		// running kind in kind for "party tricks"
		// (please don't depend on doing this though!)
		// also enable default docker volume options
		// suid: SUID applications on the volume will be able to change their privilege
		// exec: executables on the volume will be able to executed within the container
		// dev: devices on the volume will be able to be used by processes within the container
		"--volume", fmt.Sprintf("%s:/var:suid,exec,dev", varVolume),
		// some k8s things want to read /lib/modules
		"--volume", "/lib/modules:/lib/modules:ro",
		// propagate KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER to the entrypoint script
		"-e", "KIND_EXPERIMENTAL_CONTAINERD_SNAPSHOTTER",
	},
		args...,
	)

	// convert mounts and port mappings to container run args
	args = append(args, generateMountBindings(node.ExtraMounts...)...)
	mappingArgs, err := generatePortMappings(clusterIPFamily, node.ExtraPortMappings...)
	if err != nil {
		return nil, err
	}
	args = append(args, mappingArgs...)

	switch node.Role {
	case config.ControlPlaneRole:
		args = append(args, "-e", "KUBECONFIG=/etc/kubernetes/admin.conf")
	}

	// finally, specify the image to run
	_, image := sanitizeImage(node.Image)
	return append(args, image), nil
}

func runArgsForLoadBalancer(cfg *config.Cluster, name string, args []string) ([]string, error) {
	args = append([]string{
		"--hostname", name, // make hostname match container name
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", nodeRoleLabelKey, constants.ExternalLoadBalancerNodeRoleValue),
	},
		args...,
	)

	// load balancer port mapping
	mappingArgs, err := generatePortMappings(cfg.Networking.IPFamily,
		config.PortMapping{
			ListenAddress: cfg.Networking.APIServerAddress,
			HostPort:      cfg.Networking.APIServerPort,
			ContainerPort: common.APIServerInternalPort,
		},
	)
	if err != nil {
		return nil, err
	}
	args = append(args, mappingArgs...)

	// finally, specify the image to run
	_, image := sanitizeImage(loadbalancer.Image)
	return append(args, image), nil
}

func getProxyEnv(cfg *config.Cluster, networkName string, nodeNames []string) (map[string]string, error) {
	envs := common.GetProxyEnvs(cfg)
	// Specifically add the podman network subnets to NO_PROXY if we are using a proxy
	if len(envs) > 0 {
		// kind default bridge is "kind"
		subnets, err := getSubnets(networkName)
		if err != nil {
			return nil, err
		}
		noProxyList := append(subnets, envs[common.NOProxy])
		noProxyList = append(noProxyList, nodeNames...)
		// Add pod,service and all the cluster nodes' dns names to no_proxy to allow in cluster
		// Note: this is best effort based on the default CoreDNS spec
		// https://github.com/kubernetes/dns/blob/master/docs/specification.md
		// Any user created pod/service hostnames, namespaces, custom DNS services
		// are expected to be no-proxied by the user explicitly.

		noProxyList = append(noProxyList, ".svc", ".svc.cluster", ".svc.cluster.local")
		noProxyJoined := strings.Join(noProxyList, ",")
		envs[common.NOProxy] = noProxyJoined
		envs[strings.ToLower(common.NOProxy)] = noProxyJoined
	}
	return envs, nil
}

type podmanNetworks []struct {
	// v4+
	Subnets []struct {
		Subnet  string `json:"subnet"`
		Gateway string `json:"gateway"`
	} `json:"subnets"`
	// v3 and anything still using CNI/IPAM
	Plugins []struct {
		Ipam struct {
			Ranges [][]struct {
				Gateway string `json:"gateway"`
				Subnet  string `json:"subnet"`
			} `json:"ranges"`
		} `json:"ipam,omitempty"`
	} `json:"plugins"`
}

func getSubnets(networkName string) ([]string, error) {
	cmd := exec.Command("podman", "network", "inspect", networkName)
	out, err := exec.Output(cmd)

	if err != nil {
		return nil, errors.Wrap(err, "failed to get subnets")
	}

	networks := podmanNetworks{}
	jsonErr := json.Unmarshal([]byte(out), &networks)
	if jsonErr != nil {
		return nil, errors.Wrap(jsonErr, "failed to get subnets")
	}
	subnets := []string{}
	for _, network := range networks {
		if len(network.Subnets) > 0 {
			for _, subnet := range network.Subnets {
				subnets = append(subnets, subnet.Subnet)
			}
		}
		if len(network.Plugins) > 0 {
			for _, plugin := range network.Plugins {
				for _, r := range plugin.Ipam.Ranges {
					for _, rr := range r {
						subnets = append(subnets, rr.Subnet)
					}
				}
			}
		}
	}
	return subnets, nil
}

// generateMountBindings converts the mount list to a list of args for podman
// '<HostPath>:<ContainerPath>[:options]', where 'options'
// is a comma-separated list of the following strings:
// 'ro', if the path is read only
// 'Z', if the volume requires SELinux relabeling
func generateMountBindings(mounts ...config.Mount) []string {
	args := make([]string, 0, len(mounts))
	for _, m := range mounts {
		bind := fmt.Sprintf("%s:%s", m.HostPath, m.ContainerPath)
		var attrs []string
		if m.Readonly {
			attrs = append(attrs, "ro")
		}
		// Only request relabeling if the pod provides an SELinux context. If the pod
		// does not provide an SELinux context relabeling will label the volume with
		// the container's randomly allocated MCS label. This would restrict access
		// to the volume to the container which mounts it first.
		if m.SelinuxRelabel {
			attrs = append(attrs, "Z")
		}
		switch m.Propagation {
		case config.MountPropagationNone:
			// noop, private is default
		case config.MountPropagationBidirectional:
			attrs = append(attrs, "rshared")
		case config.MountPropagationHostToContainer:
			attrs = append(attrs, "rslave")
		default: // Falls back to "private"
		}
		if len(attrs) > 0 {
			bind = fmt.Sprintf("%s:%s", bind, strings.Join(attrs, ","))
		}
		args = append(args, fmt.Sprintf("--volume=%s", bind))
	}
	return args
}

// generatePortMappings converts the portMappings list to a list of args for podman
func generatePortMappings(clusterIPFamily config.ClusterIPFamily, portMappings ...config.PortMapping) ([]string, error) {
	args := make([]string, 0, len(portMappings))
	for _, pm := range portMappings {
		// do provider internal defaulting
		// in a future API revision we will handle this at the API level and remove this
		if pm.ListenAddress == "" {
			switch clusterIPFamily {
			case config.IPv4Family, config.DualStackFamily:
				pm.ListenAddress = "0.0.0.0"
			case config.IPv6Family:
				pm.ListenAddress = "::"
			default:
				return nil, errors.Errorf("unknown cluster IP family: %v", clusterIPFamily)
			}
		}
		if string(pm.Protocol) == "" {
			pm.Protocol = config.PortMappingProtocolTCP // TCP is the default
		}

		// validate that the provider can handle this binding
		switch pm.Protocol {
		case config.PortMappingProtocolTCP:
		case config.PortMappingProtocolUDP:
		case config.PortMappingProtocolSCTP:
		default:
			return nil, errors.Errorf("unknown port mapping protocol: %v", pm.Protocol)
		}

		// get a random port if necessary (port = 0)
		hostPort, err := common.PortOrGetFreePort(pm.HostPort, pm.ListenAddress)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get random host port for port mapping")
		}

		// generate the actual mapping arg
		protocol := string(pm.Protocol)
		hostPortBinding := net.JoinHostPort(pm.ListenAddress, fmt.Sprintf("%d", hostPort))
		// Podman expects empty string instead of 0 to assign a random port
		// https://github.com/containers/libpod/blob/master/pkg/spec/ports.go#L68-L69
		if strings.HasSuffix(hostPortBinding, ":0") {
			hostPortBinding = strings.TrimSuffix(hostPortBinding, "0")
		}
		args = append(args, fmt.Sprintf("--publish=%s:%d/%s", hostPortBinding, pm.ContainerPort, strings.ToLower(protocol)))
	}
	return args, nil
}

func createContainer(name string, args []string) error {
	if err := exec.Command("podman", append([]string{"run", "--name", name}, args...)...).Run(); err != nil {
		return err
	}
	return nil
}

func createContainerWithWaitUntilSystemdReachesMultiUserSystem(name string, args []string) error {
	if err := exec.Command("podman", append([]string{"run", "--name", name}, args...)...).Run(); err != nil {
		return err
	}

	logCtx, logCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer logCancel()
	logCmd := exec.CommandContext(logCtx, "podman", "logs", "-f", name)
	return common.WaitUntilLogRegexpMatches(logCtx, logCmd, common.NodeReachedCgroupsReadyRegexp())
}
