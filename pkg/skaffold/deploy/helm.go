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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type HelmDeployer struct {
	*latest.HelmDeploy

	kubeContext string
	namespace   string
	defaultRepo string
	forceDeploy bool
}

// NewHelmDeployer returns a new HelmDeployer for a DeployConfig filled
// with the needed configuration for `helm`
func NewHelmDeployer(runCtx *runcontext.RunContext) *HelmDeployer {
	return &HelmDeployer{
		HelmDeploy:  runCtx.Cfg.Deploy.HelmDeploy,
		kubeContext: runCtx.KubeContext,
		namespace:   runCtx.Opts.Namespace,
		defaultRepo: runCtx.DefaultRepo,
		forceDeploy: runCtx.Opts.ForceDeploy(),
	}
}

func (h *HelmDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "helm",
	}
}

func (h *HelmDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact, labellers []Labeller) error {
	var dRes []Artifact

	event.DeployInProgress()

	for _, r := range h.Releases {
		results, err := h.deployRelease(ctx, out, r, builds)
		if err != nil {
			releaseName, _ := evaluateReleaseName(r.Name)

			event.DeployFailed(err)
			return errors.Wrapf(err, "deploying %s", releaseName)
		}

		dRes = append(dRes, results...)
	}

	event.DeployComplete()

	labels := merge(labellers...)
	labelDeployResults(labels, dRes)

	return nil
}

func (h *HelmDeployer) Dependencies() ([]string, error) {
	var deps []string
	for _, release := range h.Releases {
		deps = append(deps, release.ValuesFiles...)

		if release.Remote {
			// chart path is only a dependency if it exists on the local filesystem
			continue
		}

		chartDepsDir := filepath.Join(release.ChartPath, "charts")
		err := filepath.Walk(release.ChartPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "failure accessing path '%s'", path)
			}

			if !info.IsDir() {
				if !strings.HasPrefix(path, chartDepsDir) || release.SkipBuildDependencies {
					// We can always add a dependency if it is not contained in our chartDepsDir.
					// However, if the file is in  our chartDepsDir, we can only include the file
					// if we are not running the helm dep build phase, as that modifies files inside
					// the chartDepsDir and results in an infinite build loop.
					deps = append(deps, path)
				}
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

func (h *HelmDeployer) helm(ctx context.Context, out io.Writer, useSecrets bool, arg ...string) error {
	args := append([]string{"--kube-context", h.kubeContext}, arg...)
	args = append(args, h.Flags.Global...)

	if useSecrets {
		args = append([]string{"secrets"}, args...)
	}

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
	if err := h.helm(ctx, out, false, "get", releaseName); err != nil {
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

	// Dependency builds should be skipped when trying to install a chart
	// with local dependencies in the chart folder, e.g. the istio helm chart.
	// This decision is left to the user.
	// Dep builds should also be skipped whenever a remote chart path is specified.
	if !r.SkipBuildDependencies && !r.Remote {
		// First build dependencies.
		logrus.Infof("Building helm dependencies...")
		if err := h.helm(ctx, out, false, "dep", "build", r.ChartPath); err != nil {
			return nil, errors.Wrap(err, "building helm dependencies")
		}
	}

	var args []string
	if !isInstalled {
		args = append(args, "install", "--name", releaseName)
		args = append(args, h.Flags.Install...)
	} else {
		args = append(args, "upgrade", releaseName)
		args = append(args, h.Flags.Upgrade...)
		if h.forceDeploy {
			args = append(args, "--force")
		}
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
	if len(r.Overrides.Values) != 0 {
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
	for _, valuesFile := range expandPaths(r.ValuesFiles) {
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
			m := createEnvVarMap(b.ImageName, extractTag(b.Tag))
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

	helmErr := h.helm(ctx, out, r.UseHelmSecrets, args...)
	return h.getDeployResults(ctx, ns, releaseName), helmErr
}

func createEnvVarMap(imageName string, digest string) map[string]string {
	customMap := map[string]string{}
	customMap["IMAGE_NAME"] = imageName
	customMap["DIGEST"] = digest
	if digest != "" {
		names := strings.SplitN(digest, ":", 2)
		if len(names) >= 2 {
			customMap["DIGEST_ALGO"] = names[0]
			customMap["DIGEST_HEX"] = names[1]
		} else {
			customMap["DIGEST_HEX"] = digest
		}
	}
	return customMap
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
	err := h.helm(ctx, buf, false, packageArgs...)
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
	if err := h.helm(ctx, &releaseInfo, false, "get", release); err != nil {
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

	if err := h.helm(ctx, out, false, "delete", releaseName, "--purge"); err != nil {
		logrus.Debugf("deleting release %s: %v\n", releaseName, err)
	}

	return nil
}

func (h *HelmDeployer) joinTagsToBuildResult(builds []build.Artifact, params map[string]string) (map[string]build.Artifact, error) {
	imageToBuildResult := map[string]build.Artifact{}
	for _, b := range builds {
		imageToBuildResult[b.ImageName] = b
	}

	paramToBuildResult := map[string]build.Artifact{}
	for param, imageName := range params {
		newImageName := util.SubstituteDefaultRepoIntoImage(h.defaultRepo, imageName)
		b, ok := imageToBuildResult[newImageName]
		if !ok {
			if len(builds) == 0 {
				logrus.Debugf("no build artifacts present. Assuming skaffold deploy. Continuing with %s", imageName)
				b = build.Artifact{ImageName: imageName, Tag: imageName}
			} else {
				return nil, fmt.Errorf("no build present for %s", imageName)
			}
		}
		paramToBuildResult[param] = b
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

func expandPaths(paths []string) []string {
	for i, path := range paths {
		expanded, err := homedir.Expand(path)
		if err == nil {
			paths[i] = expanded
		}
	}

	return paths
}
