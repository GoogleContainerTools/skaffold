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
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type HelmDeployer struct {
	*config.DeployConfig
	kubeContext string
}

// NewHelmDeployer returns a new HelmDeployer for a DeployConfig filled
// with the needed configuration for `helm`
func NewHelmDeployer(cfg *config.DeployConfig, kubeContext string) *HelmDeployer {
	return &HelmDeployer{
		DeployConfig: cfg,
		kubeContext:  kubeContext,
	}
}

func (h *HelmDeployer) Deploy(ctx context.Context, out io.Writer, b *build.BuildResult) (*Result, error) {
	for _, r := range h.HelmDeploy.Releases {
		if err := h.deployRelease(out, r, b); err != nil {
			return nil, errors.Wrapf(err, "deploying %s", r.Name)
		}
	}
	return nil, nil
}

func (h *HelmDeployer) deployRelease(out io.Writer, r config.HelmRelease, b *build.BuildResult) error {
	isInstalled := true
	getCmd := exec.Command("helm", "--kube-context", h.kubeContext, "get", r.Name)
	if stdout, stderr, err := util.RunCommand(getCmd, nil); err != nil {
		logrus.Debugf("Error getting release %s: %s stdout: %s stderr: %s", r.Name, err, string(stdout), string(stderr))
		fmt.Fprintf(out, "Helm release %s not installed. Installing...", r.Name)
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
	depCmd := exec.Command("helm", "--kube-context", h.kubeContext, "dep", "build", r.ChartPath)
	stdout, stderr, err := util.RunCommand(depCmd, nil)
	if err != nil {
		return errors.Wrapf(err, "helm dep build stdout: %s, stderr: %s", string(stdout), string(stderr))
	}
	out.Write(stdout)

	args := []string{"--kube-context", h.kubeContext}
	if !isInstalled {
		args = append(args, "install", "--name", r.Name, r.ChartPath)
	} else {
		args = append(args, "upgrade", r.Name, r.ChartPath)
	}

	if r.Namespace != "" {
		args = append(args, "--namespace", r.Namespace)
	}

	if r.ValuesFilePath != "" {
		args = append(args, "-f", r.ValuesFilePath)
	}

	if r.Version != "" {
		args = append(args, "--version", r.Version)
	}

	if len(r.SetValues) != 0 {
		for k, v := range r.SetValues {
			setOpts = append(setOpts, "--set")
			setOpts = append(setOpts, fmt.Sprintf("%s=%s", k, v))
		}
	}

	args = append(args, setOpts...)
	stdout, stderr, err = util.RunCommand(exec.Command("helm", args...), nil)
	if err != nil {
		return errors.Wrapf(err, "helm updater stdout: %s, stderr: %s", string(stdout), string(stderr))
	}
	out.Write(stdout)
	return nil
}
