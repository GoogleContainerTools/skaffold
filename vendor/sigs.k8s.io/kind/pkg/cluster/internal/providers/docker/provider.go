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

package docker

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	internallogs "sigs.k8s.io/kind/pkg/cluster/internal/logs"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/cli"
	"sigs.k8s.io/kind/pkg/internal/sets"
)

// NewProvider returns a new provider based on executing `docker ...`
func NewProvider(logger log.Logger) providers.Provider {
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
	return "docker"
}

// Provision is part of the providers.Provider interface
func (p *provider) Provision(status *cli.Status, cfg *config.Cluster) (err error) {
	// TODO: validate cfg
	// ensure node images are pulled before actually provisioning
	if err := ensureNodeImages(p.logger, status, cfg); err != nil {
		return err
	}

	// ensure the pre-requisite network exists
	networkName := fixedNetworkName
	if n := os.Getenv("KIND_EXPERIMENTAL_DOCKER_NETWORK"); n != "" {
		p.logger.Warn("WARNING: Overriding docker network due to KIND_EXPERIMENTAL_DOCKER_NETWORK")
		p.logger.Warn("WARNING: Here be dragons! This is not supported currently.")
		networkName = n
	}
	if err := ensureNetwork(networkName); err != nil {
		return errors.Wrap(err, "failed to ensure docker network")
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
	cmd := exec.Command("docker",
		"ps",
		"-a", // show stopped nodes
		// filter for nodes with the cluster label
		"--filter", "label="+clusterLabelKey,
		// format to include the cluster name
		"--format", fmt.Sprintf(`{{.Label "%s"}}`, clusterLabelKey),
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list clusters")
	}
	return sets.NewString(lines...).List(), nil
}

// ListNodes is part of the providers.Provider interface
func (p *provider) ListNodes(cluster string) ([]nodes.Node, error) {
	cmd := exec.Command("docker",
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
	const command = "docker"
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
	return nil
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

	// if the 'desktop.docker.io/ports/<PORT>/tcp' label is present,
	// defer to its value for the api server endpoint
	//
	// For example:
	// "Labels": {
	// 	"desktop.docker.io/ports/6443/tcp": "10.0.1.7:6443",
	// }
	cmd := exec.Command(
		"docker", "inspect",
		"--format", fmt.Sprintf(
			"{{ index .Config.Labels \"desktop.docker.io/ports/%d/tcp\" }}", common.APIServerInternalPort,
		),
		n.String(),
	)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed to get api server port")
	}
	if len(lines) == 1 && lines[0] != "" {
		return lines[0], nil
	}

	// else, retrieve the specific port mapping via NetworkSettings.Ports
	cmd = exec.Command(
		"docker", "inspect",
		"--format", fmt.Sprintf(
			"{{ with (index (index .NetworkSettings.Ports \"%d/tcp\") 0) }}{{ printf \"%%s\t%%s\" .HostIp .HostPort }}{{ end }}", common.APIServerInternalPort,
		),
		n.String(),
	)
	lines, err = exec.OutputLines(cmd)
	if err != nil {
		return "", errors.Wrap(err, "failed to get api server port")
	}
	if len(lines) != 1 {
		return "", errors.Errorf("network details should only be one line, got %d lines", len(lines))
	}
	parts := strings.Split(lines[0], "\t")
	if len(parts) != 2 {
		return "", errors.Errorf("network details should only be two parts, got %d", len(parts))
	}

	// join host and port
	return net.JoinHostPort(parts[0], parts[1]), nil
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
		return "", errors.Wrap(err, "failed to get api server endpoint")
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
		// record info about the host docker
		execToPathFn(
			exec.Command("docker", "info"),
			filepath.Join(dir, "docker-info.txt"),
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
			execToPathFn(exec.Command("docker", "inspect", name), filepath.Join(path, "inspect.json")),
			func() error {
				f, err := common.FileOnHost(filepath.Join(path, "serial.log"))
				if err != nil {
					return err
				}
				defer f.Close()
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
	var err error
	if p.info == nil {
		p.info, err = info()
	}
	return p.info, err
}

// dockerInfo corresponds to `docker info --format '{{json .}}'`
type dockerInfo struct {
	CgroupDriver    string   `json:"CgroupDriver"`  // "systemd", "cgroupfs", "none"
	CgroupVersion   string   `json:"CgroupVersion"` // e.g. "2"
	MemoryLimit     bool     `json:"MemoryLimit"`
	PidsLimit       bool     `json:"PidsLimit"`
	CPUShares       bool     `json:"CPUShares"`
	SecurityOptions []string `json:"SecurityOptions"`
}

func info() (*providers.ProviderInfo, error) {
	cmd := exec.Command("docker", "info", "--format", "{{json .}}")
	out, err := exec.Output(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get docker info")
	}
	var dInfo dockerInfo
	if err := json.Unmarshal(out, &dInfo); err != nil {
		return nil, err
	}
	info := providers.ProviderInfo{
		Cgroup2: dInfo.CgroupVersion == "2",
	}
	// When CgroupDriver == "none", the MemoryLimit/PidsLimit/CPUShares
	// values are meaningless and need to be considered false.
	// https://github.com/moby/moby/issues/42151
	if dInfo.CgroupDriver != "none" {
		info.SupportsMemoryLimit = dInfo.MemoryLimit
		info.SupportsPidsLimit = dInfo.PidsLimit
		info.SupportsCPUShares = dInfo.CPUShares
	}
	for _, o := range dInfo.SecurityOptions {
		// o is like "name=seccomp,profile=default", or "name=rootless",
		csvReader := csv.NewReader(strings.NewReader(o))
		sliceSlice, err := csvReader.ReadAll()
		if err != nil {
			return nil, err
		}
		for _, f := range sliceSlice {
			for _, ff := range f {
				if ff == "name=rootless" {
					info.Rootless = true
				}
			}
		}
	}
	return &info, nil
}
