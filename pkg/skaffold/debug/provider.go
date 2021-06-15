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

package debug

import (
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/debugging"
)

// Provider is an object that distributes instances of all implementations of
// Debugger in the Skaffold codebase.
type Provider interface {
	// GetKubernetesDebugger returns a new instance of the Kubernetes Debugger implementation
	GetKubernetesDebugger(*kubernetes.ImageList) Debugger

	// GetNoopDebugger returns a new instance of a Debugger that does nothing.
	GetNoopDebugger() Debugger
}

type fullProvider struct {
	kubernetesDebugger func(*kubernetes.ImageList) Debugger
}

var (
	provider *fullProvider
	once     sync.Once
)

// NewDebugProvider instantiates the Provider object that is used to retrieve instances of Debuggers.
// This method is used by the Runner, which then passes it along to the Deployers it instantiates.
func NewDebugProvider(debugConfig Config) Provider {
	once.Do(func() {
		provider = &fullProvider{
			kubernetesDebugger: func(podSelector *kubernetes.ImageList) Debugger {
				if debugConfig.Mode() != config.RunModes.Debug {
					return &NoopDebugger{}
				}

				return debugging.NewContainerManager(podSelector)
			},
		}
	})
	return provider
}

func (p *fullProvider) GetKubernetesDebugger(podSelector *kubernetes.ImageList) Debugger {
	return p.kubernetesDebugger(podSelector)
}

func (p *fullProvider) GetNoopDebugger() Debugger {
	return &NoopDebugger{}
}

// NoopProvider is used in tests
type NoopProvider struct{}

func (p *NoopProvider) GetKubernetesDebugger(_ *kubernetes.ImageList) Debugger {
	return &NoopDebugger{}
}

func (p *NoopProvider) GetNoopDebugger() Debugger {
	return &NoopDebugger{}
}
