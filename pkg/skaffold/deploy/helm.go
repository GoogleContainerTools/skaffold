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

func (h *HelmDeployer) Deploy(ctx context.Context, out io.Writer, b *build.BuildResult) error {
	for _, r := range h.HelmDeploy.Releases {
		if err := h.deployRelease(out, r, b); err != nil {
			return errors.Wrapf(err, "deploying %s", r.Name)
		}
	}
	return nil
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

func (h *HelmDeployer) helm(out io.Writer, arg ...string) error {
	args := append([]string{"--kube-context", h.kubeContext}, arg...)

	cmd := exec.Command("helm", args...)
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(cmd)
}

func (h *HelmDeployer) deployRelease(out io.Writer, r v1alpha2.HelmRelease, b *build.BuildResult) error {
	isInstalled := true
	if err := h.helm(out, "get", r.Name); err != nil {
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
	if err := h.helm(out, "dep", "build", r.ChartPath); err != nil {
		return errors.Wrap(err, "building helm dependencies")
	}

	var args []string
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

	return h.helm(out, args...)
}

func (h *HelmDeployer) deleteRelease(out io.Writer, r v1alpha2.HelmRelease) error {
	if err := h.helm(out, "delete", r.Name, "--purge"); err != nil {
		logrus.Debugf("deleting release %s: %v\n", r.Name, err)
	}

	return nil
}
