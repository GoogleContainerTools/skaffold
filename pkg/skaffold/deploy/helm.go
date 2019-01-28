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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type HelmDeployer struct {
	*latest.HelmDeploy

	kubeContext string
	namespace   string
	defaultRepo string
}

// NewHelmDeployer returns a new HelmDeployer for a DeployConfig filled
// with the needed configuration for `helm`
func NewHelmDeployer(cfg *latest.HelmDeploy, kubeContext string, namespace string, defaultRepo string) *HelmDeployer {
	return &HelmDeployer{
		HelmDeploy:  cfg,
		kubeContext: kubeContext,
		namespace:   namespace,
		defaultRepo: defaultRepo,
	}
}

func (h *HelmDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "helm",
	}
}

func (h *HelmDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact, labellers []Labeller) error {
	var dRes []Artifact

	labels := merge(labellers...)

	for _, r := range h.Releases {
		results, err := h.deployRelease(ctx, out, r, builds)
		if err != nil {
			releaseName, _ := evaluateReleaseName(r.Name)
			return errors.Wrapf(err, "deploying %s", releaseName)
		}

		dRes = append(dRes, results...)
	}

	labelDeployResults(labels, dRes)
	return nil
}

func (h *HelmDeployer) Dependencies() ([]string, error) {
	var deps []string
	for _, release := range h.Releases {
		deps = append(deps, release.ValuesFiles...)
		chartDepsDir := filepath.Join(release.ChartPath, "charts")
		err := filepath.Walk(release.ChartPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "failure accessing path '%s'", path)
			}
			if !info.IsDir() && !strings.HasPrefix(path, chartDepsDir) {
				deps = append(deps, path)
			}
			return nil
		})
		if err != nil {
			return deps, errors.Wrap(err, "issue walking releases")
		}
	}
	sort.Strings(deps)
	return deps, nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (h *HelmDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	for _, r := range h.Releases {
		if err := h.deleteRelease(ctx, out, r); err != nil {
			releaseName, _ := evaluateReleaseName(r.Name)
			return errors.Wrapf(err, "deploying %s", releaseName)
		}
	}
	return nil
}

func (h *HelmDeployer) helm(ctx context.Context, out io.Writer, arg ...string) error {
	args := append([]string{"--kube-context", h.kubeContext}, arg...)

	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(cmd)
}

func (h *HelmDeployer) deployRelease(ctx context.Context, out io.Writer, r latest.HelmRelease, builds []build.Artifact) ([]Artifact, error) {
	isInstalled := true

	releaseName, err := evaluateReleaseName(r.Name)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse the release name template")
	}
	if err := h.helm(ctx, out, "get", releaseName); err != nil {
		color.Red.Fprintf(out, "Helm release %s not installed. Installing...\n", releaseName)
		isInstalled = false
	}
	params, err := h.joinTagsToBuildResult(builds, r.Values)
	if err != nil {
		return nil, errors.Wrap(err, "matching build results to chart values")
	}

	var setOpts []string
	for k, v := range params {
		setOpts = append(setOpts, "--set")
		if r.ImageStrategy.HelmImageConfig.HelmConventionConfig != nil {
			dockerRef, err := docker.ParseReference(v.Tag)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot parse the docker image reference %s", v.Tag)
			}
			imageRepositoryTag := fmt.Sprintf("%s.repository=%s,%s.tag=%s", k, dockerRef.BaseName, k, dockerRef.Tag)
			setOpts = append(setOpts, imageRepositoryTag)
		} else {
			setOpts = append(setOpts, fmt.Sprintf("%s=%s", k, v.Tag))
		}
	}

	if !r.SkipBuildDependencies {
		// First build dependencies.
		logrus.Infof("Building helm dependencies...")
		if err := h.helm(ctx, out, "dep", "build", r.ChartPath); err != nil {
			return nil, errors.Wrap(err, "building helm dependencies")
		}
	}

	var args []string
	if !isInstalled {
		args = append(args, "install", "--name", releaseName)
	} else {
		args = append(args, "upgrade", releaseName)
		if r.RecreatePods {
			args = append(args, "--recreate-pods")
		}
	}

	// There are 2 strategies:
	// 1) Deploy chart directly from filesystem path or from repository
	//    (like stable/kubernetes-dashboard). Version only applies to a
	//    chart from repository.
	// 2) Package chart into a .tgz archive with specific version and then deploy
	//    that packaged chart. This way user can apply any version and appVersion
	//    for the chart.
	if r.Packaged == nil {
		if r.Version != "" {
			args = append(args, "--version", r.Version)
		}
		args = append(args, r.ChartPath)
	} else {
		chartPath, err := h.packageChart(ctx, r)
		if err != nil {
			return nil, errors.WithMessage(err, "cannot package chart")
		}
		args = append(args, chartPath)
	}

	var ns string
	if h.namespace != "" {
		ns = h.namespace
	} else if r.Namespace != "" {
		ns = r.Namespace
	}
	if ns != "" {
		args = append(args, "--namespace", ns)
	}
	if len(r.Overrides) != 0 {
		overrides, err := yaml.Marshal(r.Overrides)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal overrides to create overrides values.yaml")
		}
		overridesFile, err := os.Create(constants.HelmOverridesFilename)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create file %s", constants.HelmOverridesFilename)
		}
		defer func() {
			overridesFile.Close()
			os.Remove(constants.HelmOverridesFilename)
		}()
		if _, err := overridesFile.WriteString(string(overrides)); err != nil {
			return nil, errors.Wrapf(err, "failed to write file %s", constants.HelmOverridesFilename)
		}
		args = append(args, "-f", constants.HelmOverridesFilename)
	}
	for _, valuesFile := range r.ValuesFiles {
		args = append(args, "-f", valuesFile)
	}

	setValues := r.SetValues
	if setValues == nil {
		setValues = map[string]string{}
	}
	if len(r.SetValueTemplates) != 0 {
		envMap := map[string]string{}
		for idx, b := range builds {
			suffix := ""
			if idx > 0 {
				suffix = strconv.Itoa(idx + 1)
			}
			m := tag.CreateEnvVarMap(b.ImageName, extractTag(b.Tag))
			for k, v := range m {
				envMap[k+suffix] = v
			}
			color.Default.Fprintf(out, "EnvVarMap: %#v\n", envMap)
		}
		for k, v := range r.SetValueTemplates {
			t, err := util.ParseEnvTemplate(v)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse setValueTemplates")
			}
			result, err := util.ExecuteEnvTemplate(t, envMap)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate setValueTemplates")
			}
			setValues[k] = result
		}
	}
	for k, v := range setValues {
		setOpts = append(setOpts, "--set")
		setOpts = append(setOpts, fmt.Sprintf("%s=%s", k, v))
	}
	if r.Wait {
		args = append(args, "--wait")
	}
	args = append(args, setOpts...)

	helmErr := h.helm(ctx, out, args...)
	return h.getDeployResults(ctx, ns, releaseName), helmErr
}

// imageName if the given string includes a fully qualified docker image name then lets trim just the tag part out
func extractTag(imageName string) string {
	idx := strings.LastIndex(imageName, "/")
	if idx < 0 {
		return imageName
	}
	tag := imageName[idx+1:]
	idx = strings.Index(tag, ":")
	if idx > 0 {
		return tag[idx+1:]
	}
	return tag
}

// packageChart packages the chart and returns path to the chart archive file.
// If this function returns an error, it will always be wrapped.
func (h *HelmDeployer) packageChart(ctx context.Context, r latest.HelmRelease) (string, error) {
	tmp := os.TempDir()
	packageArgs := []string{"package", r.ChartPath, "--destination", tmp}
	if r.Packaged.Version != "" {
		v, err := concretize(r.Packaged.Version)
		if err != nil {
			return "", errors.Wrap(err, `concretize "packaged.version" template`)
		}
		packageArgs = append(packageArgs, "--version", v)
	}
	if r.Packaged.AppVersion != "" {
		av, err := concretize(r.Packaged.AppVersion)
		if err != nil {
			return "", errors.Wrap(err, `concretize "packaged.appVersion" template`)
		}
		packageArgs = append(packageArgs, "--app-version", av)
	}

	buf := &bytes.Buffer{}
	err := h.helm(ctx, buf, packageArgs...)
	output := strings.TrimSpace(buf.String())
	if err != nil {
		return "", errors.Wrapf(err, "package chart into a .tgz archive (%s)", output)
	}

	fpath, err := extractChartFilename(output, tmp)
	if err != nil {
		return "", err
	}

	return filepath.Join(tmp, fpath), nil
}

func (h *HelmDeployer) getReleaseInfo(ctx context.Context, release string) (*bufio.Reader, error) {
	var releaseInfo bytes.Buffer
	if err := h.helm(ctx, &releaseInfo, "get", release); err != nil {
		return nil, fmt.Errorf("error retrieving helm deployment info: %s", releaseInfo.String())
	}
	return bufio.NewReader(&releaseInfo), nil
}

// Retrieve info about all releases using helm get
// Skaffold labels will be applied to each deployed k8s object
// Since helm isn't always consistent with retrieving results, don't return errors here
func (h *HelmDeployer) getDeployResults(ctx context.Context, namespace string, release string) []Artifact {
	b, err := h.getReleaseInfo(ctx, release)
	if err != nil {
		logrus.Warnf(err.Error())
		return nil
	}
	return parseReleaseInfo(namespace, b)
}

func (h *HelmDeployer) deleteRelease(ctx context.Context, out io.Writer, r latest.HelmRelease) error {
	releaseName, err := evaluateReleaseName(r.Name)
	if err != nil {
		return errors.Wrap(err, "cannot parse the release name template")
	}

	if err := h.helm(ctx, out, "delete", releaseName, "--purge"); err != nil {
		logrus.Debugf("deleting release %s: %v\n", releaseName, err)
	}

	return nil
}

func (h *HelmDeployer) joinTagsToBuildResult(builds []build.Artifact, params map[string]string) (map[string]build.Artifact, error) {
	imageToBuildResult := map[string]build.Artifact{}
	for _, build := range builds {
		imageToBuildResult[build.ImageName] = build
	}

	paramToBuildResult := map[string]build.Artifact{}
	for param, imageName := range params {
		newImageName := util.SubstituteDefaultRepoIntoImage(h.defaultRepo, imageName)
		build, ok := imageToBuildResult[newImageName]
		if !ok {
			return nil, fmt.Errorf("no build present for %s", imageName)
		}
		paramToBuildResult[param] = build
	}
	return paramToBuildResult, nil
}

func evaluateReleaseName(nameTemplate string) (string, error) {
	tmpl, err := util.ParseEnvTemplate(nameTemplate)
	if err != nil {
		return "", errors.Wrap(err, "parsing template")
	}

	return util.ExecuteEnvTemplate(tmpl, nil)
}

// concretize parses and executes template s with OS environment variables.
// If s is not a template but a simple string, returns unchanged s.
func concretize(s string) (string, error) {
	tmpl, err := util.ParseEnvTemplate(s)
	if err != nil {
		return "", errors.Wrap(err, "parsing template")
	}

	tmpl.Option("missingkey=error")
	return util.ExecuteEnvTemplate(tmpl, nil)
}

func extractChartFilename(s, tmp string) (string, error) {
	s = strings.TrimSpace(s)
	idx := strings.Index(s, tmp)
	if idx == -1 {
		return "", errors.New("cannot locate packaged chart archive")
	}

	return s[idx+len(tmp):], nil
}
