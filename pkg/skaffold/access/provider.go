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

package access

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
)

type Provider interface {
	GetKubernetesAccessor(portforward.Config, *kubernetes.ImageList) Accessor
	GetNoopAccessor() Accessor
}

type fullProvider struct {
	label       label.Config
	k8sAccessor map[string]Accessor
}

func NewAccessorProvider(labelConfig label.Config) Provider {
	return &fullProvider{label: labelConfig, k8sAccessor: make(map[string]Accessor)}
}

func (p *fullProvider) GetKubernetesAccessor(config portforward.Config, podSelector *kubernetes.ImageList) Accessor {
	if !config.PortForwardOptions().Enabled() {
		return &NoopAccessor{}
	}
	context := config.GetKubeContext()

	if p.k8sAccessor[context] == nil {
		p.k8sAccessor[context] = portforward.NewForwarderManager(kubectl.NewCLI(config, ""),
			podSelector,
			p.label.RunIDSelector(),
			config.Mode(),
			config.PortForwardOptions(),
			config.PortForwardResources())
	}
	return p.k8sAccessor[context]
}

func (p *fullProvider) GetNoopAccessor() Accessor {
	return &NoopAccessor{}
}

// NoopProvider is used in tests
type NoopProvider struct{}

func (p *NoopProvider) GetKubernetesAccessor(_ portforward.Config, _ *kubernetes.ImageList) Accessor {
	return &NoopAccessor{}
}

func (p *NoopProvider) GetNoopAccessor() Accessor {
	return &NoopAccessor{}
}
