/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliep.
See the License for the specific language governing permissions and
limitations under the License.
*/

package podman

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	internallogs "sigs.k8s.io/kind/pkg/cluster/internal/logs"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/cli"
	"sigs.k8s.io/kind/pkg/internal/sets"
	"sigs.k8s.io/kind/pkg/internal/version"
)

// NewProvider returns a new provider based on executing `podman ...`
func NewProvider(logger log.Logger) providers.Provider {
	logger.Warn("enabling experimental podman provider")
	return &provider{
		logger: logger,
	}
}

// Provider implements provider.Provider
// see NewProvider
type provider struct {
	logger log.Logger
	info   *providers.ProviderInfo
}

// String implements fmt.Stringer
// NOTE: the value of this should not currently be relied upon for anything!
// This is only used for setting the Node's providerID
func (p *provider) String() string {
	return "podman"
}

// Provision is part of the providers.Provider interface
func (p *provider) Provision(status *cli.Status, cfg *config.Cluster) (err error) {
	if err := ensureMinVersion(); err != nil {
		return err
	}

	// TODO: validate cfg
	// ensure node images are pulled before actually provisioning
	if err := ensureNodeImages(p.logger, status, cfg); err != nil {
		return err
	}

	// ensure the pre-requisite network exists
	networkName := fixedNetworkName
	if n := os.Getenv("KIND_EXPERIMENTAL_PODMAN_NETWORK"); n != "" {
		p.logger.Warn("WARNING: Overriding podman network due to KIND_EXPERIMENTAL_PODMAN_NETWORK")
		p.logger.Warn("WARNING: Here be dragons! This is not supported currently.")
		networkName = n
	}
	if err := ensureNetwork(networkName); err != nil {
		return errors.Wrap(err, "failed to ensure podman network")
	}

	// actually provision the cluster
	icons := strings.Repeat("📦 ", len(cfg.Nodes))
	status.Start(fmt.Sprintf("Preparing nodes %s", icons))
	defer func() { status.End(err == nil) }()

	// plan creating the containers
	createContainerFuncs, err := planCreation(cfg, networkName)
	if err != nil {
		return err
	}

	// actually create nodes
	return errors.UntilErrorConcurrent(createContainerFuncs)
}

// ListClusters is part of the providers.Provider interface
func (p *provider) ListClusters() ([]string, error) {
	cmd := exec.Command("podman",
		"ps",
		"-a", // show stopped nodes
		// filter for nodes with the cluster label
		"--filter", "label="+clusterLabelKey,
		// format to include the cluster name
		"--format", fmt.Sprintf(`{{index .Labels "%s"}}`, clusterLabelKey),
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list clusters")
	}
	return sets.NewString(lines...).List(), nil
}

// ListNodes is part of the providers.Provider interface
func (p *provider) ListNodes(cluster string) ([]nodes.Node, error) {
	cmd := exec.Command("podman",
		"ps",
		"-a", // show stopped nodes
		// filter for nodes with the cluster label
		"--filter", fmt.Sprintf("label=%s=%s", clusterLabelKey, cluster),
		// format to include the cluster name
		"--format", `{{.Names}}`,
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list nodes")
	}
	// convert names to node handles
	ret := make([]nodes.Node, 0, len(lines))
	for _, name := range lines {
		ret = append(ret, p.node(name))
	}
	return ret, nil
}

// DeleteNodes is part of the providers.Provider interface
func (p *provider) DeleteNodes(n []nodes.Node) error {
	if len(n) == 0 {
		return nil
	}
	const command = "podman"
	args := make([]string, 0, len(n)+3) // allocate once
	args = append(args,
		"rm",
		"-f", // force the container to be delete now
		"-v", // delete volumes
	)
	for _, node := range n {
		args = append(args, node.String())
	}
	if err := exec.Command(command, args...).Run(); err != nil {
		return errors.Wrap(err, "failed to delete nodes")
	}
	var nodeVolumes []string
	for _, node := range n {
		volumes, err := getVolumes(node.String())
		if err != nil {
			return err
		}
		nodeVolumes = append(nodeVolumes, volumes...)
	}
	if len(nodeVolumes) == 0 {
		return nil
	}
	return deleteVolumes(nodeVolumes)
}

// getHostIPOrDefault defaults HostIP to localhost if is not set
// xref: https://github.com/kubernetes-sigs/kind/issues/3777
func getHostIPOrDefault(hostIP string) string {
	if hostIP == "" {
		return "127.0.0.1"
	}
	return hostIP
}

// GetAPIServerEndpoint is part of the providers.Provider interface
func (p *provider) GetAPIServerEndpoint(cluster string) (string, error) {
	// locate the node that hosts this
	allNodes, err := p.ListNodes(cluster)
	if err != nil {
		return "", errors.Wrap(err, "failed to list nodes")
	}
	n, err := nodeutils.APIServerEndpointNode(allNodes)
	if err != nil {
		return "", errors.Wrap(err, "failed to get api server endpoint")
	}

	// TODO: get rid of this once podman settles on how to get the port mapping using podman inspect
	// This is only used to get the Kubeconfig server field
	v, err := getPodmanVersion()
	if err != nil {
		return "", errors.Wrap(err, "failed to check podman version")
	}
	// podman inspect was broken between 2.2.0 and 3.0.0
	// https://github.com/containers/podman/issues/8444
	if v.AtLeast(version.MustParseSemantic("2.2.0")) &&
		v.LessThan(version.MustParseSemantic("3.0.0")) {
		p.logger.Warnf("WARNING: podman version %s not fully supported, please use versions 3.0.0+")

		cmd := exec.Command(
			"podman", "inspect",
			"--format",
			"{{range .NetworkSettings.Ports }}{{range .}}{{.HostIP}}/{{.HostPort}}{{end}}{{end}}",
			n.String(),
		)

		lines, err := exec.OutputLines(cmd)
		if err != nil {
			return "", errors.Wrap(err, "failed to get api server port")
		}
		if len(lines) != 1 {
			return "", errors.Errorf("network details should only be one line, got %d lines", len(lines))
		}
		// output is in the format IP/Port
		parts := strings.Split(strings.TrimSpace(lines[0]), "/")
		if len(parts) != 2 {
			return "", errors.Errorf("network details should be in the format IP/Port, received: %s", parts)
		}
		host := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", errors.Errorf("network port not an integer: %v", err)
		}

		return net.JoinHostPort(host, strconv.Itoa(port)), nil
	}

	cmd := exec.Command(
		"podman", "inspect",
		"--format",
		"{{ json .NetworkSettings.Ports }}",
		n.String(),
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed to get api server port")
	}
	if len(lines) != 1 {
		return "", errors.Errorf("network details should only be one line, got %d lines", len(lines))
	}

	// portMapping19 maps to the standard CNI portmapping capability used in podman 1.9
	// see: https://github.com/containernetworking/cni/blob/spec-v0.4.0/CONVENTIONS.md
	type portMapping19 struct {
		HostPort      int32  `json:"hostPort"`
		ContainerPort int32  `json:"containerPort"`
		Protocol      string `json:"protocol"`
		HostIP        string `json:"hostIP"`
	}
	// portMapping20 maps to the podman 2.0 portmap type
	// see: https://github.com/containers/podman/blob/05988fc74fc25f2ad2256d6e011dfb7ad0b9a4eb/libpod/define/container_inspect.go#L134-L143
	type portMapping20 struct {
		HostPort string `json:"HostPort"`
		HostIP   string `json:"HostIp"`
	}

	portMappings20 := make(map[string][]portMapping20)
	if err := json.Unmarshal([]byte(lines[0]), &portMappings20); err == nil {
		for k, v := range portMappings20 {
			protocol := "tcp"
			parts := strings.Split(k, "/")
			if len(parts) == 2 {
				protocol = strings.ToLower(parts[1])
			}
			containerPort, err := strconv.Atoi(parts[0])
			if err != nil {
				return "", err
			}
			for _, pm := range v {
				if containerPort == common.APIServerInternalPort && protocol == "tcp" {
					return net.JoinHostPort(getHostIPOrDefault(pm.HostIP), pm.HostPort), nil
				}
			}
		}
	}

	var portMappings19 []portMapping19
	if err := json.Unmarshal([]byte(lines[0]), &portMappings19); err != nil {
		return "", errors.Errorf("invalid network details: %v", err)
	}
	for _, pm := range portMappings19 {
		if pm.ContainerPort == common.APIServerInternalPort && pm.Protocol == "tcp" {
			return net.JoinHostPort(getHostIPOrDefault(pm.HostIP), strconv.Itoa(int(pm.HostPort))), nil
		}
	}

	return "", errors.Errorf("failed to get api server port")
}

// GetAPIServerInternalEndpoint is part of the providers.Provider interface
func (p *provider) GetAPIServerInternalEndpoint(cluster string) (string, error) {
	// locate the node that hosts this
	allNodes, err := p.ListNodes(cluster)
	if err != nil {
		return "", errors.Wrap(err, "failed to list nodes")
	}
	n, err := nodeutils.APIServerEndpointNode(allNodes)
	if err != nil {
		return "", errors.Wrap(err, "failed to get apiserver endpoint")
	}
	// NOTE: we're using the nodes's hostnames which are their names
	return net.JoinHostPort(n.String(), fmt.Sprintf("%d", common.APIServerInternalPort)), nil
}

// node returns a new node handle for this provider
func (p *provider) node(name string) nodes.Node {
	return &node{
		name: name,
	}
}

// CollectLogs will populate dir with cluster logs and other debug files
func (p *provider) CollectLogs(dir string, nodes []nodes.Node) error {
	execToPathFn := func(cmd exec.Cmd, path string) func() error {
		return func() error {
			f, err := common.FileOnHost(path)
			if err != nil {
				return err
			}
			defer f.Close()
			return cmd.SetStdout(f).SetStderr(f).Run()
		}
	}
	// construct a slice of methods to collect logs
	fns := []func() error{
		// record info about the host podman
		execToPathFn(
			exec.Command("podman", "info"),
			filepath.Join(dir, "podman-info.txt"),
		),
	}

	// collect /var/log for each node and plan collecting more logs
	var errs []error
	for _, n := range nodes {
		node := n // https://golang.org/doc/faq#closures_and_goroutines
		name := node.String()
		path := filepath.Join(dir, name)
		if err := internallogs.DumpDir(p.logger, node, "/var/log", path); err != nil {
			errs = append(errs, err)
		}

		fns = append(fns,
			func() error { return common.CollectLogs(node, path) },
			execToPathFn(exec.Command("podman", "inspect", name), filepath.Join(path, "inspect.json")),
			func() error {
				f, err := common.FileOnHost(filepath.Join(path, "serial.log"))
				if err != nil {
					return err
				}
				return node.SerialLogs(f)
			},
		)
	}

	// run and collect up all errors
	errs = append(errs, errors.AggregateConcurrent(fns))
	return errors.NewAggregate(errs)
}

// Info returns the provider info.
// The info is cached on the first time of the execution.
func (p *provider) Info() (*providers.ProviderInfo, error) {
	if p.info == nil {
		var err error
		p.info, err = info(p.logger)
		if err != nil {
			return p.info, err
		}
	}
	return p.info, nil
}

// podmanInfo corresponds to `podman info --format 'json`.
// The structure is different from `docker info --format '{{json .}}'`,
// and lacks information about the availability of the cgroup controllers.
type podmanInfo struct {
	Host struct {
		CgroupVersion     string   `json:"cgroupVersion,omitempty"` // "v2"
		CgroupControllers []string `json:"cgroupControllers,omitempty"`
		Security          struct {
			Rootless bool `json:"rootless,omitempty"`
		} `json:"security"`
	} `json:"host"`
}

// info detects ProviderInfo by executing `podman info --format json`.
func info(logger log.Logger) (*providers.ProviderInfo, error) {
	const podman = "podman"
	args := []string{"info", "--format", "json"}
	cmd := exec.Command(podman, args...)
	out, err := exec.Output(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get podman info (%s %s): %q",
			podman, strings.Join(args, " "), string(out))
	}
	var pInfo podmanInfo
	if err := json.Unmarshal(out, &pInfo); err != nil {
		return nil, err
	}
	stringSliceContains := func(s []string, str string) bool {
		for _, v := range s {
			if v == str {
				return true
			}
		}
		return false
	}

	// Since Podman version before v4.0.0 does not gives controller info.
	// We assume all the cgroup controllers to be available.
	// For rootless, this assumption is not always correct,
	// so we print the warning below.
	cgroupSupportsMemoryLimit := true
	cgroupSupportsPidsLimit := true
	cgroupSupportsCPUShares := true

	v, err := getPodmanVersion()
	if err != nil {
		return nil, errors.Wrap(err, "failed to check podman version")
	}
	// Info for controllers must be available after v4.0.0
	// via https://github.com/containers/podman/pull/10387
	if v.AtLeast(version.MustParseSemantic("4.0.0")) {
		cgroupSupportsMemoryLimit = stringSliceContains(pInfo.Host.CgroupControllers, "memory")
		cgroupSupportsPidsLimit = stringSliceContains(pInfo.Host.CgroupControllers, "pids")
		cgroupSupportsCPUShares = stringSliceContains(pInfo.Host.CgroupControllers, "cpu")
	}

	info := &providers.ProviderInfo{
		Rootless:            pInfo.Host.Security.Rootless,
		Cgroup2:             pInfo.Host.CgroupVersion == "v2",
		SupportsMemoryLimit: cgroupSupportsMemoryLimit,
		SupportsPidsLimit:   cgroupSupportsPidsLimit,
		SupportsCPUShares:   cgroupSupportsCPUShares,
	}
	if info.Rootless && !v.AtLeast(version.MustParseSemantic("4.0.0")) {
		if logger != nil {
			logger.Warn("Cgroup controller detection is not implemented for Podman. " +
				"If you see cgroup-related errors, you might need to set systemd property \"Delegate=yes\", see https://kind.sigs.k8s.io/docs/user/rootless/")
		}
	}
	return info, nil
}
