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

package preview

import (
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
)

type Provider interface {
	GetKubernetesPreviewer(*kubernetes.ImageList) ResourcePreviewer
	GetNoopPreviewer() ResourcePreviewer
}

type fullProvider struct {
	kubernetesPreviewer func(*kubernetes.ImageList) ResourcePreviewer
	noopPreviewer       func() ResourcePreviewer
}

var (
	provider *fullProvider
	once     sync.Once
)

func NewPreviewProvider(config portforward.Config, labelConfig label.Config, cli *kubectl.CLI) Provider {
	once.Do(func() {
		provider = &fullProvider{
			kubernetesPreviewer: func(podSelector *kubernetes.ImageList) ResourcePreviewer {
				if !config.PortForwardOptions().Enabled() {
					return &NoopPreviewer{}
				}

				return portforward.NewForwarderManager(cli,
					podSelector,
					labelConfig.RunIDSelector(),
					config.Mode(),
					config.PortForwardOptions(),
					config.PortForwardResources())
			},
			noopPreviewer: func() ResourcePreviewer {
				return &NoopPreviewer{}
			},
		}
	})
	return provider
}

func (p *fullProvider) GetKubernetesPreviewer(s *kubernetes.ImageList) ResourcePreviewer {
	return p.kubernetesPreviewer(s)
}

func (p *fullProvider) GetNoopPreviewer() ResourcePreviewer {
	return p.noopPreviewer()
}

// NoopProvider is used in tests
type NoopProvider struct{}

func (p *NoopProvider) GetKubernetesPreviewer(_ *kubernetes.ImageList) ResourcePreviewer {
	return &NoopPreviewer{}
}

func (p *NoopProvider) GetNoopPreviewer() ResourcePreviewer {
	return &NoopPreviewer{}
}
