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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type HelmDeployer struct {
	*v1alpha2.DeployConfig
	kubeContext string
}

// NewHelmDeployer returns a new HelmDeployer for a DeployConfig filled
// with the needed configuration for `helm`
func NewHelmDeployer(cfg *v1alpha2.DeployConfig, kubeContext string) *HelmDeployer {
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

// Not implemented
func (k *HelmDeployer) Dependencies() ([]string, error) {
	return nil, nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (h *HelmDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	for _, r := range h.HelmDeploy.Releases {
		if err := h.deleteRelease(out, r); err != nil {
			return errors.Wrapf(err, "deploying %s", r.Name)
		}
	}
	return nil
}

func (h *HelmDeployer) args(moreArgs ...string) []string {
	return append([]string{"--kube-context", h.kubeContext}, moreArgs...)
}

func (h *HelmDeployer) deployRelease(out io.Writer, r v1alpha2.HelmRelease, b *build.BuildResult) error {
	isInstalled := true
	getCmd := exec.Command("helm", h.args("get", r.Name)...)
	if stdout, stderr, err := util.RunCommand(getCmd); err != nil {
		logrus.Debugf("Error getting release %s: %s stdout: %s stderr: %s", r.Name, err, string(stdout), string(stderr))
		fmt.Fprintf(out, "Helm release %s not installed. Installing...\n", r.Name)
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

	// First build dependencies.
	logrus.Infof("Building helm dependencies...")
	depCmd := exec.Command("helm", h.args("dep", "build", r.ChartPath)...)
	stdout, stderr, err := util.RunCommand(depCmd)
	if err != nil {
		return errors.Wrapf(err, "helm dep build stdout: %s, stderr: %s", string(stdout), string(stderr))
	}
	out.Write(stdout)

	args := h.args()
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

	execCmd := exec.Command("helm", args...)
	stdout, stderr, err = util.RunCommand(execCmd)
	if err != nil {
		return errors.Wrapf(err, "helm updater stdout: %s, stderr: %s", string(stdout), string(stderr))
	}

	out.Write(stdout)
	return nil
}

func (h *HelmDeployer) deleteRelease(out io.Writer, r v1alpha2.HelmRelease) error {
	getCmd := exec.Command("helm", h.args("delete", r.Name, "--purge")...)
	stdout, stderr, err := util.RunCommand(getCmd)
	if err != nil {
		logrus.Debugf("running helm delete %s: %v stdout: %s stderr: %s", r.Name, err, string(stdout), string(stderr))
	}

	out.Write(stdout)
	return nil
}
