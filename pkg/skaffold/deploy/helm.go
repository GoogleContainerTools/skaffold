/*
Copyright 2018 The Skaffold Authors

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
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type HelmDeployer struct {
	*v1alpha2.DeployConfig
	kubeContext string
	namespace   string
}

// NewHelmDeployer returns a new HelmDeployer for a DeployConfig filled
// with the needed configuration for `helm`
func NewHelmDeployer(cfg *v1alpha2.DeployConfig, kubeContext string, namespace string) *HelmDeployer {
	return &HelmDeployer{
		DeployConfig: cfg,
		kubeContext:  kubeContext,
		namespace:    namespace,
	}
}

func (h *HelmDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Build) error {
	for _, r := range h.HelmDeploy.Releases {
		if err := h.deployRelease(out, r, builds); err != nil {
			releaseName, _ := evaluateReleaseName(r.Name)
			return errors.Wrapf(err, "deploying %s", releaseName)
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
			releaseName, _ := evaluateReleaseName(r.Name)
			return errors.Wrapf(err, "deploying %s", releaseName)
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

func (h *HelmDeployer) deployRelease(out io.Writer, r v1alpha2.HelmRelease, builds []build.Build) error {
	isInstalled := true

	releaseName, err := evaluateReleaseName(r.Name)
	if err != nil {
		return errors.Wrap(err, "cannot parse the release name template")
	}
	if err := h.helm(out, "get", releaseName); err != nil {
		fmt.Fprintf(out, "Helm release %s not installed. Installing...\n", releaseName)
		isInstalled = false
	}
	params, err := JoinTagsToBuildResult(builds, r.Values)
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
		args = append(args, "install", "--name", releaseName, r.ChartPath)
	} else {
		args = append(args, "upgrade", releaseName, r.ChartPath)
	}

	var ns string
	if h.namespace != "" {
		ns = h.namespace
	} else if r.Namespace != "" {
		ns = r.Namespace
	} else {
		ns = os.Getenv("SKAFFOLD_DEPLOY_NAMESPACE")
	}
	if ns != "" {
		args = append(args, "--namespace", ns)
	}
	if len(r.Overrides) != 0 {
		overrides, err := yaml.Marshal(r.Overrides)
		if err != nil {
			return errors.Wrap(err, "cannot marshal overrides to create overrides values.yaml")
		}
		overridesFile, err := os.Create("skaffold-overrides.yaml")
		if err != nil {
			return errors.Wrap(err, "cannot create file skaffold-overrides.yaml")
		}
		if _, err := overridesFile.WriteString(string(overrides)); err != nil {
			return errors.Wrap(err, "failed to write file skaffold-overrides.yaml")
		}
		args = append(args, "-f", "skaffold-overrides.yaml")
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
	if r.Wait {
		args = append(args, "--wait")
	}
	args = append(args, setOpts...)

	helmErr := h.helm(out, args...)
	if len(r.Overrides) != 0 {
		os.Remove("skaffold-overrides.yaml")
	}
	return helmErr
}

func (h *HelmDeployer) deleteRelease(out io.Writer, r v1alpha2.HelmRelease) error {
	releaseName, err := evaluateReleaseName(r.Name)
	if err != nil {
		return errors.Wrap(err, "cannot parse the release name template")
	}

	if err := h.helm(out, "delete", releaseName, "--purge"); err != nil {
		logrus.Debugf("deleting release %s: %v\n", releaseName, err)
	}

	return nil
}

func evaluateReleaseName(nameTemplate string) (string, error) {

	tmpl, err := util.ParseEnvTemplate(nameTemplate)
	if err != nil {
		return "", errors.Wrap(err, "parsing template")
	}

	return util.ExecuteEnvTemplate(tmpl, nil)
}
