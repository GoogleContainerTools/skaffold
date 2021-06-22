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
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
)

type Provider interface {
	GetKubernetesMonitor() Monitor
	GetNoopMonitor() Monitor
}

type fullProvider struct {
	kubernetesMonitor Monitor
}

var (
	provider *fullProvider
	once     sync.Once
)

func NewMonitorProvider(config status.Config, l *label.DefaultLabeller) Provider {
	once.Do(func() {
		var c Monitor
		enabled, _ := config.StatusCheck()
		if enabled != nil && !*enabled { // assume enabled if value unspecified
			c = &NoopMonitor{}
		} else {
			c = status.NewStatusMonitor(config, l)
		}
		provider = &fullProvider{
			kubernetesMonitor: c,
		}
	})
	return provider
}

func (p *fullProvider) GetKubernetesMonitor() Monitor {
	return p.kubernetesMonitor
}

func (p *fullProvider) GetNoopMonitor() Monitor {
	return &NoopMonitor{}
}

// NoopProvider is used in tests
type NoopProvider struct{}

func (p *NoopProvider) GetKubernetesMonitor() Monitor {
	return &NoopMonitor{}
}

func (p *NoopProvider) GetNoopMonitor() Monitor {
	return &NoopMonitor{}
}
