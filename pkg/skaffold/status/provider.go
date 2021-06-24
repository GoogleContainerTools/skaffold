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

package status

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
)

type Provider interface {
	GetKubernetesMonitor(config status.Config) Monitor
	GetNoopMonitor() Monitor
}

type fullProvider struct {
	k8sMonitor map[string]Monitor // keyed on KubeContext. TODO: make KubeContext a struct type.
	labeller   *label.DefaultLabeller
}

func NewMonitorProvider(l *label.DefaultLabeller) Provider {
	return &fullProvider{k8sMonitor: make(map[string]Monitor), labeller: l}
}

func (p *fullProvider) GetKubernetesMonitor(config status.Config) Monitor {
	enabled := config.StatusCheck()
	if enabled != nil && !*enabled { // assume disabled only if explicitly set to false
		return &NoopMonitor{}
	}
	context := config.GetKubeContext()
	if p.k8sMonitor[context] == nil {
		p.k8sMonitor[context] = status.NewStatusMonitor(config, p.labeller)
	}
	return p.k8sMonitor[context]
}

func (p *fullProvider) GetNoopMonitor() Monitor {
	return &NoopMonitor{}
}

// NoopProvider is used in tests
type NoopProvider struct{}

func (p *NoopProvider) GetKubernetesMonitor(config status.Config) Monitor {
	return &NoopMonitor{}
}

func (p *NoopProvider) GetNoopMonitor() Monitor {
	return &NoopMonitor{}
}
