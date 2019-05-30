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

package context

import (
	"os"

	configutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	runnerutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	watchutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type RunContext struct {
	Opts *config.SkaffoldOptions
	Cfg  *latest.Pipeline

	// TODO(nkubala): these channels can be sent to by the client at anytime,
	// meaning if a trigger is sent by the user it will "stick" in the channel
	// and cannot be cancelled by the user. we should only open these channels
	// for writing when skaffold is explicitly waiting for a user signal.
	BuildTrigger  chan bool
	DeployTrigger chan bool

	DefaultRepo        string
	KubeContext        string
	WorkingDir         string
	Namespaces         []string
	InsecureRegistries map[string]bool
}

func GetRunContext(opts *config.SkaffoldOptions, cfg *latest.Pipeline) (*RunContext, error) {
	kubeContext, err := kubectx.CurrentContext()
	if err != nil {
		return nil, errors.Wrap(err, "getting current cluster context")
	}
	logrus.Infof("Using kubectl context: %s", kubeContext)

	// TODO(dgageot): this should be the folder containing skaffold.yaml. Should also be moved elsewhere.
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "finding current directory")
	}

	namespaces, err := runnerutil.GetAllPodNamespaces(opts.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "getting namespace list")
	}

	defaultRepo, err := configutil.GetDefaultRepo(opts.DefaultRepo)
	if err != nil {
		return nil, errors.Wrap(err, "getting default repo")
	}

	// combine all provided lists of insecure registries into a map
	cfgRegistries, err := configutil.GetInsecureRegistries()
	if err != nil {
		logrus.Warnf("error retrieving insecure registries from global config: push/pull issues may exist...")
	}
	regList := append(opts.InsecureRegistries, cfg.Build.InsecureRegistries...)
	regList = append(regList, cfgRegistries...)
	insecureRegistries := make(map[string]bool, len(regList))
	for _, r := range regList {
		insecureRegistries[r] = true
	}

	var buildTrigger, deployTrigger chan bool

	if opts.ManualDeploy {
		deployTrigger = make(chan bool, 1)
	}
	if watchutil.IsApiTrigger(opts.BuildTrigger) {
		buildTrigger = make(chan bool, 1)
	}

	return &RunContext{
		Opts:               opts,
		Cfg:                cfg,
		WorkingDir:         cwd,
		DefaultRepo:        defaultRepo,
		KubeContext:        kubeContext,
		Namespaces:         namespaces,
		InsecureRegistries: insecureRegistries,
		BuildTrigger:       buildTrigger,
		DeployTrigger:      deployTrigger,
	}, nil
}
