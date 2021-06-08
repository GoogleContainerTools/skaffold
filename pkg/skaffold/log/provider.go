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

package log

import (
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/logger"
)

type Provider interface {
	GetKubernetesLogger(*kubernetes.ImageList) Logger
	GetNoopLogger() Logger
}

type fullProvider struct {
	tail bool

	kubernetesLogger func(*kubernetes.ImageList) Logger
	noopLogger       func() Logger
}

var (
	provider *fullProvider
	once     sync.Once
)

func NewLogProvider(config logger.Config, cli *kubectl.CLI) Provider {
	once.Do(func() {
		provider = &fullProvider{
			tail: config.Tail(),
			kubernetesLogger: func(podSelector *kubernetes.ImageList) Logger {
				return logger.NewLogAggregator(cli, podSelector, config)
			},
			noopLogger: func() Logger {
				return &NoopLogger{}
			},
		}
	})
	return provider
}

func (p *fullProvider) GetKubernetesLogger(s *kubernetes.ImageList) Logger {
	if !p.tail {
		return p.noopLogger()
	}
	return p.kubernetesLogger(s)
}

func (p *fullProvider) GetNoopLogger() Logger {
	return p.noopLogger()
}

// NoopProvider is used in tests
type NoopProvider struct{}

func (p *NoopProvider) GetKubernetesLogger(_ *kubernetes.ImageList) Logger {
	return &NoopLogger{}
}

func (p *NoopProvider) GetNoopLogger() Logger {
	return &NoopLogger{}
}
