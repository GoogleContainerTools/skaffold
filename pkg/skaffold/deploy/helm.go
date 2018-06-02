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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

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

// For testing
var environ = os.Environ

// NewHelmDeployer returns a new HelmDeployer for a DeployConfig filled
// with the needed configuration for `helm`
func NewHelmDeployer(cfg *v1alpha2.DeployConfig, kubeContext string) *HelmDeployer {
	return &HelmDeployer{
		DeployConfig: cfg,
		kubeContext:  kubeContext,
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

	ns := r.Namespace
	if ns == "" {
		ns = os.Getenv("SKAFFOLD_DEPLOY_NAMESPACE")
	}
	if ns != "" {
		args = append(args, "--namespace", ns)
	}
	if r.ValuesFilePath != "" {
		args = append(args, "-f", r.ValuesFilePath)
	}
	if r.Version != "" {
		args = append(args, "--version", r.Version)
	}

	setValues := r.SetValues
	if setValues == nil {
		setValues = map[string]string{}
	}
	if len(r.SetValueTemplates) != 0 {
		m, err := h.evaluateTemplates(r.SetValueTemplates, b)
		if err != nil {
			return errors.Wrapf(err, "failed to generate setValueTemplates")
		}
		for k, v := range m {
			setValues[k] = v
		}
	}
	for k, v := range setValues {
		setOpts = append(setOpts, "--set")
		setOpts = append(setOpts, fmt.Sprintf("%s=%s", k, v))
	}
	args = append(args, setOpts...)

	return h.helm(out, args...)
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

func (h *HelmDeployer) evaluateTemplates(setValueTemplates map[string]string, r *build.BuildResult) (map[string]string, error) {
	results := map[string]string{}
	envMap := map[string]string{}
	for _, env := range environ() {
		kvp := strings.SplitN(env, "=", 2)
		if len(kvp) != 2 {
			return results, fmt.Errorf("error parsing environment variables, %s does not contain an =", kvp)
		}
		envMap[kvp[0]] = kvp[1]
	}

	for idx, b := range r.Builds {
		suffix := ""
		if idx > 0 {
			suffix = strconv.Itoa(idx + 1)
		}
		envMap["IMAGE_NAME"+suffix] = b.ImageName
		envMap["TAG"+suffix] = b.Tag
	}

	for k, v := range setValueTemplates {
		tmpl, err := template.New("envTemplate").Parse(v)
		if err != nil {
			return results, errors.Wrap(err, "parsing template")
		}

		logrus.Debugf("Executing template %v with environment %v", tmpl, envMap)
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, envMap); err != nil {
			return results, errors.Wrap(err, "executing template")
		}
		results[k] = buf.String()
	}
	return results, nil
}
