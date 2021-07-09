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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
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
)

func newAccessor(cfg portforward.Config, cli *kubectl.CLI, podSelector kubernetes.PodSelector, labeller label.Config) access.Accessor {
	if !cfg.PortForwardOptions().Enabled() {
		return &access.NoopAccessor{}
	}
	return portforward.NewForwarderManager(cli, podSelector, labeller.RunIDSelector(), cfg.Mode(), cfg.PortForwardOptions(), cfg.PortForwardResources())
}

func newDebugger(mode config.RunMode, podSelector kubernetes.PodSelector) debug.Debugger {
	if mode != config.RunModes.Debug {
		return &debug.NoopDebugger{}
	}

	return debugging.NewContainerManager(podSelector)
}

func newImageLoader(cfg k8sloader.Config, cli *kubectl.CLI) loader.ImageLoader {
	if cfg.LoadImages() {
		return k8sloader.NewImageLoader(cfg.GetKubeContext(), cli)
	}
	return &loader.NoopImageLoader{}
}

func newLogger(config k8slogger.Config, cli *kubectl.CLI, podSelector kubernetes.PodSelector) log.Logger {
	return k8slogger.NewLogAggregator(cli, podSelector, config)
}

func newMonitor(cfg k8sstatus.Config, labeller *label.DefaultLabeller) status.Monitor {
	enabled := cfg.StatusCheck()
	if enabled != nil && !*enabled { // assume disabled only if explicitly set to false
		return &status.NoopMonitor{}
	}
	return k8sstatus.NewStatusMonitor(cfg, labeller)
}

func newSyncer(config sync.Config, cli *kubectl.CLI) sync.Syncer {
	return sync.NewPodSyncer(cli, config)
}
