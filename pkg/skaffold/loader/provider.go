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

package loader

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/loader"
)

type Provider interface {
	GetKubernetesImageLoader() ImageLoader
	GetNoopImageLoader() ImageLoader
}

type fullProvider struct {
	loadImages  bool
	kubeContext string
	cli         *kubectl.CLI
}

func NewImageLoaderProvider(config loader.Config, cli *kubectl.CLI) Provider {
	return &fullProvider{
		loadImages:  config.LoadImages(),
		kubeContext: config.GetKubeContext(),
		cli:         cli,
	}
}

func (p *fullProvider) GetKubernetesImageLoader() ImageLoader {
	if p.loadImages {
		return loader.NewImageLoader(p.kubeContext, p.cli)
	}
	return &NoopImageLoader{}
}

func (p *fullProvider) GetNoopImageLoader() ImageLoader {
	return &NoopImageLoader{}
}

// NoopProvider is used in tests
type NoopProvider struct{}

func (p *NoopProvider) GetKubernetesImageLoader() ImageLoader {
	return &NoopImageLoader{}
}

func (p *NoopProvider) GetNoopImageLoader() ImageLoader {
	return &NoopImageLoader{}
}
