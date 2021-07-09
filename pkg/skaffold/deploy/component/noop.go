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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	k8sloader "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	k8sstatus "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
)

// NoopComponentProvider is for tests
type NoopComponentProvider struct{}

func (n NoopComponentProvider) GetKubernetesAccessor(portforward.Config, *kubernetes.ImageList) access.Accessor {
	return &access.NoopAccessor{}
}

func (n NoopComponentProvider) GetKubernetesDebugger(*kubernetes.ImageList) debug.Debugger {
	return &debug.NoopDebugger{}
}

func (n NoopComponentProvider) GetKubernetesLogger(*kubernetes.ImageList) log.Logger {
	return &log.NoopLogger{}
}

func (n NoopComponentProvider) GetKubernetesImageLoader(k8sloader.Config) loader.ImageLoader {
	return &loader.NoopImageLoader{}
}

func (n NoopComponentProvider) GetKubernetesMonitor(k8sstatus.Config) status.Monitor {
	return &status.NoopMonitor{}
}

func (n NoopComponentProvider) GetKubernetesSyncer() sync.Syncer {
	return &sync.NoopSyncer{}
}
