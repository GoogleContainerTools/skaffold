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
	GetKubernetesChecker() Checker
	GetNoopChecker() Checker
}

type fullProvider struct {
	kubernetesChecker Checker
}

var (
	provider *fullProvider
	once     sync.Once
)

func NewCheckerProvider(config status.Config, l *label.DefaultLabeller) Provider {
	once.Do(func() {
		var c Checker
		enabled, _ := config.StatusCheck()
		if enabled != nil && !*enabled { // assume enabled if value unspecified
			c = &NoopChecker{}
		} else {
			c = status.NewStatusChecker(config, l)
		}
		provider = &fullProvider{
			kubernetesChecker: c,
		}
	})
	return provider
}

func (p *fullProvider) GetKubernetesChecker() Checker {
	return p.kubernetesChecker
}

func (p *fullProvider) GetNoopChecker() Checker {
	return &NoopChecker{}
}

// NoopProvider is used in tests
type NoopProvider struct{}

func (p *NoopProvider) GetKubernetesChecker() Checker {
	return &NoopChecker{}
}

func (p *NoopProvider) GetNoopChecker() Checker {
	return &NoopChecker{}
}
