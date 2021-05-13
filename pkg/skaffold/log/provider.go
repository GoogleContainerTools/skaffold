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
	Do() Logger
}

type FullProvider struct {
	tail bool

	kubernetesLogger *logger.LogAggregator
	noopLogger       *NoopLogger
}

var provider *FullProvider

func NewLogProvider(cli *kubectl.CLI, podSelector kubernetes.PodSelector, config logger.Config) Provider {
	once := sync.Once{}
	once.Do(func() {
		kLog := logger.NewLogAggregator(cli, podSelector, config)
		provider = &FullProvider{
			tail:             config.Tail(),
			kubernetesLogger: kLog,
			noopLogger:       &NoopLogger{},
		}
	})
	return provider
}

func (p *FullProvider) Do() Logger {
	if !p.tail {
		return p.noopLogger
	}
	return p.kubernetesLogger
}

// NoopProvider is used in tests
type NoopProvider struct{}

func (p *NoopProvider) Do() Logger {
	return &NoopLogger{}
}
