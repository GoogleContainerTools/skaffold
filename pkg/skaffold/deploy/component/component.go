/*
Copyright 2021 The Skaffold Authors

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

package component

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/debugging"
	k8sloader "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	k8sstatus "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"

	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

// ComponentProvider distributes various sub-components to a Deployer
type ComponentProvider struct {
	config      Config
	cli         *kubectl.CLI
	k8sAccessor map[string]access.Accessor
	k8sMonitor  map[string]status.Monitor // keyed on KubeContext. TODO: make KubeContext a struct type.
	labeller    *label.DefaultLabeller
}

type Config interface {
	Tail() bool
	PipelineForImage(imageName string) (latestV1.Pipeline, bool)
	DefaultPipeline() latestV1.Pipeline
	Mode() config.RunMode
	GetNamespaces() []string
}

func NewComponentProvider(config Config, cli *kubectl.CLI, labeller *label.DefaultLabeller) ComponentProvider {
	return ComponentProvider{
		config:      config,
		cli:         cli,
		labeller:    labeller,
		k8sMonitor:  make(map[string]status.Monitor),
		k8sAccessor: make(map[string]access.Accessor),
	}
}

func (c ComponentProvider) GetKubernetesAccessor(config portforward.Config, podSelector *kubernetes.ImageList) access.Accessor {
	if !config.PortForwardOptions().Enabled() {
		return &access.NoopAccessor{}
	}
	context := config.GetKubeContext()

	if c.k8sAccessor[context] == nil {
		c.k8sAccessor[context] = portforward.NewForwarderManager(kubectl.NewCLI(config, ""),
			podSelector,
			c.labeller.RunIDSelector(),
			config.Mode(),
			config.PortForwardOptions(),
			config.PortForwardResources())
	}
	return c.k8sAccessor[context]
}

func (c ComponentProvider) GetKubernetesDebugger(podSelector *kubernetes.ImageList) debug.Debugger {
	if c.config.Mode() != config.RunModes.Debug {
		return &debug.NoopDebugger{}
	}

	return debugging.NewContainerManager(podSelector)
}

func (c ComponentProvider) GetKubernetesLogger(podSelector *kubernetes.ImageList) log.Logger {
	return logger.NewLogAggregator(c.cli, podSelector, c.config)
}

func (c ComponentProvider) GetKubernetesImageLoader(config k8sloader.Config) loader.ImageLoader {
	if config.LoadImages() {
		return k8sloader.NewImageLoader(config.GetKubeContext(), kubectl.NewCLI(config, ""))
	}
	return &loader.NoopImageLoader{}
}

func (c ComponentProvider) GetKubernetesMonitor(config k8sstatus.Config) status.Monitor {
	enabled := config.StatusCheck()
	if enabled != nil && !*enabled { // assume disabled only if explicitly set to false
		return &status.NoopMonitor{}
	}
	context := config.GetKubeContext()
	if c.k8sMonitor[context] == nil {
		c.k8sMonitor[context] = k8sstatus.NewStatusMonitor(config, c.labeller)
	}
	return c.k8sMonitor[context]
}

func (c ComponentProvider) GetKubernetesSyncer() sync.Syncer {
	return sync.NewPodSyncer(c.cli, c.config)
}
