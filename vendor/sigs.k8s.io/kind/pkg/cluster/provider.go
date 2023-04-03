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

package cluster

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"sigs.k8s.io/kind/pkg/cmd/kind/version"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"

	internalcreate "sigs.k8s.io/kind/pkg/cluster/internal/create"
	internaldelete "sigs.k8s.io/kind/pkg/cluster/internal/delete"
	"sigs.k8s.io/kind/pkg/cluster/internal/kubeconfig"
	internalproviders "sigs.k8s.io/kind/pkg/cluster/internal/providers"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/docker"
	"sigs.k8s.io/kind/pkg/cluster/internal/providers/podman"
)

// DefaultName is the default cluster name
const DefaultName = constants.DefaultClusterName

// defaultName is a helper that given a name defaults it if unset
func defaultName(name string) string {
	if name == "" {
		name = DefaultName
	}
	return name
}

// Provider is used to perform cluster operations
type Provider struct {
	provider internalproviders.Provider
	logger   log.Logger
}

// NewProvider returns a new provider based on the supplied options
func NewProvider(options ...ProviderOption) *Provider {
	p := &Provider{
		logger: log.NoopLogger{},
	}
	// Ensure we apply the logger options first, while maintaining the order
	// otherwise. This way we can trivially init the internal provider with
	// the logger.
	sort.SliceStable(options, func(i, j int) bool {
		_, iIsLogger := options[i].(providerLoggerOption)
		_, jIsLogger := options[j].(providerLoggerOption)
		return iIsLogger && !jIsLogger
	})
	for _, o := range options {
		if o != nil {
			o.apply(p)
		}
	}

	// ensure a provider if none was set
	// NOTE: depends on logger being set (see sorting above)
	if p.provider == nil {
		// DetectNodeProvider does not fallback to allow callers to determine
		// this behavior
		// However for compatibility if the caller of NewProvider supplied no
		// option and we autodetect internally, we default to the docker provider
		// for fallback, to avoid a breaking change for now.
		// This may change in the future.
		// TODO: consider breaking this API for earlier errors.
		providerOpt, _ := DetectNodeProvider()
		if providerOpt == nil {
			providerOpt = ProviderWithDocker()
		}
		providerOpt.apply(p)
	}
	return p
}

// NoNodeProviderDetectedError indicates that we could not autolocate an available
// NodeProvider backend on the host
var NoNodeProviderDetectedError = errors.NewWithoutStack("failed to detect any supported node provider")

// DetectNodeProvider allows callers to autodetect the node provider
// *without* fallback to the default.
//
// Pass the returned ProviderOption to NewProvider to pass the auto-detect Docker
// or Podman option explicitly (in the future there will be more options)
//
// NOTE: The kind *cli* also checks `KIND_EXPERIMENTAL_PROVIDER` for "podman" or
// "docker" currently and does not auto-detect / respects this if set.
//
// This will be replaced with some other mechanism in the future (likely when
// podman support is GA), in the meantime though your tool may wish to match this.
//
// In the future when this is not considered experimental,
// that logic will be in a public API as well.
func DetectNodeProvider() (ProviderOption, error) {
	// auto-detect based on each node provider's IsAvailable() function
	if docker.IsAvailable() {
		return ProviderWithDocker(), nil
	}
	if podman.IsAvailable() {
		return ProviderWithPodman(), nil
	}
	return nil, errors.WithStack(NoNodeProviderDetectedError)
}

// ProviderOption is an option for configuring a provider
type ProviderOption interface {
	apply(p *Provider)
}

// providerLoggerOption is a trivial ProviderOption adapter
// we use a type specific to logging options so we can handle them first
type providerLoggerOption func(p *Provider)

func (a providerLoggerOption) apply(p *Provider) {
	a(p)
}

var _ ProviderOption = providerLoggerOption(nil)

// ProviderWithLogger configures the provider to use Logger logger
func ProviderWithLogger(logger log.Logger) ProviderOption {
	return providerLoggerOption(func(p *Provider) {
		p.logger = logger
	})
}

// providerLoggerOption is a trivial ProviderOption adapter
// we use a type specific to logging options so we can handle them first
type providerRuntimeOption func(p *Provider)

func (a providerRuntimeOption) apply(p *Provider) {
	a(p)
}

var _ ProviderOption = providerRuntimeOption(nil)

// ProviderWithDocker configures the provider to use docker runtime
func ProviderWithDocker() ProviderOption {
	return providerRuntimeOption(func(p *Provider) {
		p.provider = docker.NewProvider(p.logger)
	})
}

// ProviderWithPodman configures the provider to use podman runtime
func ProviderWithPodman() ProviderOption {
	return providerRuntimeOption(func(p *Provider) {
		p.provider = podman.NewProvider(p.logger)
	})
}

// Create provisions and starts a kubernetes-in-docker cluster
func (p *Provider) Create(name string, options ...CreateOption) error {
	// apply options
	opts := &internalcreate.ClusterOptions{
		NameOverride: name,
	}
	for _, o := range options {
		if err := o.apply(opts); err != nil {
			return err
		}
	}
	return internalcreate.Cluster(p.logger, p.provider, opts)
}

// Delete tears down a kubernetes-in-docker cluster
func (p *Provider) Delete(name, explicitKubeconfigPath string) error {
	return internaldelete.Cluster(p.logger, p.provider, defaultName(name), explicitKubeconfigPath)
}

// List returns a list of clusters for which nodes exist
func (p *Provider) List() ([]string, error) {
	return p.provider.ListClusters()
}

// KubeConfig returns the KUBECONFIG for the cluster
// If internal is true, this will contain the internal IP etc.
// If internal is false, this will contain the host IP etc.
func (p *Provider) KubeConfig(name string, internal bool) (string, error) {
	return kubeconfig.Get(p.provider, defaultName(name), !internal)
}

// ExportKubeConfig exports the KUBECONFIG for the cluster, merging
// it into the selected file, following the rules from
// https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#config
// where explicitPath is the --kubeconfig value.
func (p *Provider) ExportKubeConfig(name string, explicitPath string, internal bool) error {
	return kubeconfig.Export(p.provider, defaultName(name), explicitPath, !internal)
}

// ListNodes returns the list of container IDs for the "nodes" in the cluster
func (p *Provider) ListNodes(name string) ([]nodes.Node, error) {
	return p.provider.ListNodes(defaultName(name))
}

// ListInternalNodes returns the list of container IDs for the "nodes" in the cluster
// that are not external
func (p *Provider) ListInternalNodes(name string) ([]nodes.Node, error) {
	n, err := p.provider.ListNodes(name)
	if err != nil {
		return nil, err
	}
	return nodeutils.InternalNodes(n)
}

// CollectLogs will populate dir with cluster logs and other debug files
func (p *Provider) CollectLogs(name, dir string) error {
	// TODO: should use ListNodes and Collect should handle nodes differently
	// based on role ...
	n, err := p.ListInternalNodes(name)
	if err != nil {
		return err
	}
	// ensure directory
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to create logs directory")
	}
	// write kind version
	if err := ioutil.WriteFile(
		filepath.Join(dir, "kind-version.txt"),
		[]byte(version.DisplayVersion()),
		0666, // match os.Create
	); err != nil {
		return errors.Wrap(err, "failed to write kind-version.txt")
	}
	// collect and write cluster logs
	return p.provider.CollectLogs(dir, n)
}
