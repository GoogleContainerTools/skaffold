/*
Copyright 2019 The Skaffold Authors

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

package runcontext

import (
	"fmt"
	"os"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	runnerutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type RunContext struct {
	Opts config.SkaffoldOptions
	Cfg  latest.Pipeline

	KubeContext        string
	WorkingDir         string
	Namespaces         []string
	InsecureRegistries map[string]bool
}

func GetRunContext(opts config.SkaffoldOptions, cfg latest.Pipeline) (*RunContext, error) {
	kubeConfig, err := kubectx.CurrentConfig()
	if err != nil {
		return nil, fmt.Errorf("getting current cluster context: %w", err)
	}
	kubeContext := kubeConfig.CurrentContext
	logrus.Infof("Using kubectl context: %s", kubeContext)

	// TODO(dgageot): this should be the folder containing skaffold.yaml. Should also be moved elsewhere.
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("finding current directory: %w", err)
	}

	namespaces, err := runnerutil.GetAllPodNamespaces(opts.Namespace, cfg)
	if err != nil {
		return nil, fmt.Errorf("getting namespace list: %w", err)
	}

	// combine all provided lists of insecure registries into a map
	cfgRegistries, err := config.GetInsecureRegistries(opts.GlobalConfig)
	if err != nil {
		logrus.Warnf("error retrieving insecure registries from global config: push/pull issues may exist...")
	}
	regList := append(opts.InsecureRegistries, cfg.Build.InsecureRegistries...)
	regList = append(regList, cfgRegistries...)
	insecureRegistries := make(map[string]bool, len(regList))
	for _, r := range regList {
		insecureRegistries[r] = true
	}

	return &RunContext{
		Opts:               opts,
		Cfg:                cfg,
		WorkingDir:         cwd,
		KubeContext:        kubeContext,
		Namespaces:         namespaces,
		InsecureRegistries: insecureRegistries,
	}, nil
}

func (r *RunContext) UpdateNamespaces(ns []string) {
	if len(ns) == 0 {
		return
	}

	nsMap := map[string]bool{}
	for _, ns := range append(ns, r.Namespaces...) {
		nsMap[ns] = true
	}

	// Update RunContext Namespace
	updated := make([]string, 0, len(nsMap))
	for k := range nsMap {
		updated = append(updated, k)
	}
	sort.Strings(updated)
	r.Namespaces = updated
}
