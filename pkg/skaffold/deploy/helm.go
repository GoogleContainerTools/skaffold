/*
Copyright 2018 Google LLC

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

package deploy

import (
	"fmt"
	"os/exec"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type HelmDeployer struct {
	*config.DeployConfig
}

func NewHelmDeployer(cfg *config.DeployConfig) (*HelmDeployer, error) {
	return &HelmDeployer{cfg}, nil
}

func (h *HelmDeployer) Run(b *build.BuildResult) (*Result, error) {
	for _, r := range h.HelmDeploy.Releases {
		if err := deployRelease(r, b); err != nil {
			return nil, errors.Wrapf(err, "deploying %s", r.Name)
		}
	}
	return nil, nil
}

func deployRelease(r config.HelmRelease, b *build.BuildResult) error {
	isInstalled := true
	getCmd := exec.Command("helm", "get", r.Name)
	if out, stderr, err := util.RunCommand(getCmd, nil); err != nil {
		logrus.Debugf("Error getting release %s: %s stdout: %s stderr: %s", r.Name, err, string(out), string(stderr))
		logrus.Infof("Helm release %s not installed. Installing...", r.Name)
		isInstalled = false
	}
	params, err := JoinTagsToBuildResult(b.Builds, r.Values)
	if err != nil {
		return errors.Wrap(err, "matching build results to chart values")
	}

	var setOpts []string
	for k, v := range params {
		setOpts = append(setOpts, "--set")
		setOpts = append(setOpts, fmt.Sprintf("%s=%s", k, v.Tag))
	}

	logrus.Infof("Building helm dependencies...")
	// First build dependencies.
	depCmd := exec.Command("helm", "dep", "build", r.ChartPath)
	out, stderr, err := util.RunCommand(depCmd, nil)
	if err != nil {
		return errors.Wrapf(err, "helm dep build stdout: %s, stderr: %s", string(out), string(stderr))
	}
	logrus.Infof("Helm: %s", string(out))

	var args []string
	if !isInstalled {
		args = []string{"install", "--name", r.Name, r.ChartPath}
	} else {
		args = []string{"upgrade", r.Name, r.ChartPath}
	}

	args = append(args, setOpts...)
	out, stderr, err = util.RunCommand(exec.Command("helm", args...), nil)
	if err != nil {
		return errors.Wrapf(err, "helm updater stdout: %s, stderr: %s", string(out), string(stderr))
	}
	logrus.Infof("Helm: %s", string(out))
	return nil
}
