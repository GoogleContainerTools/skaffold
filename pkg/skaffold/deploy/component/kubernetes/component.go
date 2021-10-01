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

package kubernetes

import (
	gosync "sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/debugging"
	k8sloader "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/loader"
	k8slogger "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	k8sstatus "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
)

// For testing
var (
	NewAccessor    = newAccessor
	NewDebugger    = newDebugger
	NewImageLoader = newImageLoader
	NewLogger      = newLogger
	NewMonitor     = newMonitor
	NewSyncer      = newSyncer

	accessLock  gosync.Mutex
	k8sAccessor map[string]access.Accessor

	monitorLock gosync.Mutex
	k8sMonitor  map[string]status.Monitor
)

func newAccessor(cfg portforward.Config, kubeContext string, cli *kubectl.CLI, podSelector kubernetes.PodSelector, labeller label.Config, namespaces *[]string) access.Accessor {
	accessLock.Lock()
	defer accessLock.Unlock()
	if k8sAccessor == nil {
		k8sAccessor = make(map[string]access.Accessor)
	}
	if k8sAccessor[kubeContext] == nil {
		if !cfg.PortForwardOptions().Enabled() {
			k8sAccessor[kubeContext] = &access.NoopAccessor{}
		}
		m := portforward.NewForwarderManager(cli, podSelector, labeller.RunIDSelector(), cfg.Mode(), namespaces, cfg.PortForwardOptions(), cfg.PortForwardResources())
		if m == nil {
			k8sAccessor[kubeContext] = &access.NoopAccessor{}
		} else {
			k8sAccessor[kubeContext] = m
		}
	} else if accessor, ok := k8sAccessor[kubeContext].(*portforward.ForwarderManager); ok {
		accessor.AddPodForwarder(cli, podSelector, cfg.Mode(), cfg.PortForwardOptions())
	}

	return k8sAccessor[kubeContext]
}

func newDebugger(mode config.RunMode, podSelector kubernetes.PodSelector, namespaces *[]string, kubeContext string) debug.Debugger {
	if mode != config.RunModes.Debug {
		return &debug.NoopDebugger{}
	}

	return debugging.NewContainerManager(podSelector, namespaces, kubeContext)
}

func newImageLoader(cfg k8sloader.Config, cli *kubectl.CLI) loader.ImageLoader {
	if cfg.LoadImages() {
		return k8sloader.NewImageLoader(cfg.GetKubeContext(), cli)
	}
	return &loader.NoopImageLoader{}
}

func newLogger(config k8slogger.Config, cli *kubectl.CLI, podSelector kubernetes.PodSelector, namespaces *[]string) k8slogger.Logger {
	return k8slogger.NewLogAggregator(cli, podSelector, namespaces, config)
}

func newMonitor(cfg k8sstatus.Config, kubeContext string, labeller *label.DefaultLabeller, namespaces *[]string) status.Monitor {
	monitorLock.Lock()
	defer monitorLock.Unlock()
	if k8sMonitor == nil {
		k8sMonitor = make(map[string]status.Monitor)
	}
	if k8sMonitor[kubeContext] == nil {
		enabled := cfg.StatusCheck()
		if enabled != nil && !*enabled { // assume disabled only if explicitly set to false
			k8sMonitor[kubeContext] = &status.NoopMonitor{}
		} else {
			k8sMonitor[kubeContext] = k8sstatus.NewStatusMonitor(cfg, labeller, namespaces)
		}
	}
	return k8sMonitor[kubeContext]
}

func newSyncer(cli *kubectl.CLI, namespaces *[]string, formatter k8slogger.Formatter) sync.Syncer {
	return sync.NewPodSyncer(cli, namespaces, formatter)
}
