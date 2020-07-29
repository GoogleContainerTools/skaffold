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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/cenkalti/backoff/v4"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/walk"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
)

var (
	// versionRegex extracts version from "helm version --client", for instance: "2.14.0-rc.2"
	versionRegex = regexp.MustCompile(`v(\d[\w.\-]+)`)

	// helm3Version represents the version cut-off for helm3 behavior
	helm3Version = semver.MustParse("3.0.0-beta.0")

	// error to throw when helm version can't be determined
	versionErrorString = "failed to determine binary version: %w"
)

// HelmDeployer deploys workflows using the helm CLI
type HelmDeployer struct {
	*latest.HelmDeploy

	kubeContext string
	kubeConfig  string
	namespace   string

	// packaging temporary directory, used for predictable test output
	pkgTmpDir string

	labels map[string]string

	forceDeploy bool

	// bV is the helm binary version
	bV semver.Version
}

// NewHelmDeployer returns a configured HelmDeployer
func NewHelmDeployer(runCtx *runcontext.RunContext, labels map[string]string) *HelmDeployer {
	return &HelmDeployer{
		HelmDeploy:  runCtx.Cfg.Deploy.HelmDeploy,
		kubeContext: runCtx.KubeContext,
		kubeConfig:  runCtx.Opts.KubeConfig,
		namespace:   runCtx.Opts.Namespace,
		forceDeploy: runCtx.Opts.Force,
		labels:      labels,
	}
}

// Deploy deploys the build results to the Kubernetes cluster
func (h *HelmDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]string, error) {
	hv, err := h.binVer(ctx)
	if err != nil {
		return nil, fmt.Errorf(versionErrorString, err)
	}

	logrus.Infof("Deploying with helm v%s ...", hv)

	var dRes []Artifact
	nsMap := map[string]struct{}{}
	valuesSet := map[string]bool{}

	// Deploy every release
	for _, r := range h.Releases {
		results, err := h.deployRelease(ctx, out, r, builds, valuesSet, hv)
		if err != nil {
			releaseName, _ := util.ExpandEnvTemplate(r.Name, nil)
			return nil, fmt.Errorf("deploying %q: %w", releaseName, err)
		}

		// collect namespaces
		for _, r := range results {
			if trimmed := strings.TrimSpace(r.Namespace); trimmed != "" {
				nsMap[trimmed] = struct{}{}
			}
		}

		dRes = append(dRes, results...)
	}

	// Let's make sure that every image tag is set with `--set`.
	// Otherwise, templates have no way to use the images that were built.
	for _, b := range builds {
		if !valuesSet[b.Tag] {
			warnings.Printf("image [%s] is not used.", b.Tag)
			warnings.Printf("image [%s] is used instead.", b.ImageName)
			warnings.Printf("See helm sample for how to replace image names with their actual tags: https://github.com/GoogleContainerTools/skaffold/blob/master/examples/helm-deployment/skaffold.yaml")
		}
	}

	if err := labelDeployResults(h.labels, dRes); err != nil {
		return nil, fmt.Errorf("adding labels: %w", err)
	}

	// Collect namespaces in a string
	namespaces := make([]string, 0, len(nsMap))
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}

	return namespaces, nil
}

// Dependencies returns a list of files that the deployer depends on.
func (h *HelmDeployer) Dependencies() ([]string, error) {
	var deps []string

	for _, release := range h.Releases {
		r := release
		deps = append(deps, r.ValuesFiles...)

		if r.Remote {
			// chart path is only a dependency if it exists on the local filesystem
			continue
		}

		chartDepsDirs := []string{
			"charts",
			"tmpcharts",
		}

		lockFiles := []string{
			"Chart.lock",
			"requirements.lock",
		}

		// We can always add a dependency if it is not contained in our chartDepsDirs.
		// However, if the file is in our chartDepsDir, we can only include the file
		// if we are not running the helm dep build phase, as that modifies files inside
		// the chartDepsDir and results in an infinite build loop.
		// We additionally exclude ChartFile.lock (Helm 3) and requirements.lock (Helm 2)
		// since they also get modified on helm dep build phase
		isDep := func(path string, info walk.Dirent) (bool, error) {
			if info.IsDir() {
				return false, nil
			}
			if r.SkipBuildDependencies {
				return true, nil
			}

			for _, v := range chartDepsDirs {
				if strings.HasPrefix(path, filepath.Join(release.ChartPath, v)) {
					return false, nil
				}
			}

			for _, v := range lockFiles {
				if strings.EqualFold(info.Name(), v) {
					return false, nil
				}
			}

			return true, nil
		}

		if err := walk.From(release.ChartPath).When(isDep).AppendPaths(&deps); err != nil {
			return deps, fmt.Errorf("issue walking releases: %w", err)
		}
	}
	sort.Strings(deps)
	return deps, nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (h *HelmDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	hv, err := h.binVer(ctx)
	if err != nil {
		return fmt.Errorf(versionErrorString, err)
	}

	for _, r := range h.Releases {
		releaseName, err := util.ExpandEnvTemplate(r.Name, nil)
		if err != nil {
			return fmt.Errorf("cannot parse the release name template: %w", err)
		}

		var namespace string
		if h.namespace != "" {
			namespace = h.namespace
		} else if r.Namespace != "" {
			namespace = r.Namespace
		}

		args := []string{"delete", releaseName}
		if hv.LT(helm3Version) {
			args = append(args, "--purge")
		} else if namespace != "" {
			args = append(args, "--namespace", namespace)
		}
		if err := h.exec(ctx, out, false, args...); err != nil {
			return fmt.Errorf("deleting %q: %w", releaseName, err)
		}
	}
	return nil
}

// Render generates the Kubernetes manifests and writes them out
func (h *HelmDeployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, offline bool, filepath string) error {
	hv, err := h.binVer(ctx)
	if err != nil {
		return fmt.Errorf(versionErrorString, err)
	}

	renderedManifests := new(bytes.Buffer)

	for _, r := range h.Releases {
		args := []string{"template", r.ChartPath}

		if hv.GTE(helm3Version) {
			// Helm 3 requires the name to be before the chart path
			args = append(args[:1], append([]string{r.Name}, args[1:]...)...)
		} else {
			args = append(args, "--name", r.Name)
		}

		for _, vf := range r.ValuesFiles {
			args = append(args, "--values", vf)
		}

		params, err := pairParamsToArtifacts(builds, r.ArtifactOverrides)
		if err != nil {
			return fmt.Errorf("matching build results to chart values: %w", err)
		}

		for k, v := range params {
			var value string

			cfg := r.ImageStrategy.HelmImageConfig.HelmConventionConfig

			value, err = imageSetFromConfig(cfg, k, v.Tag)
			if err != nil {
				return err
			}

			args = append(args, "--set-string", value)
		}

		args, err = constructOverrideArgs(&r, builds, args, func(string) {})
		if err != nil {
			return err
		}

		if r.Namespace != "" {
			args = append(args, "--namespace", r.Namespace)
		}

		if err := h.exec(ctx, renderedManifests, false, args...); err != nil {
			return err
		}
	}

	return outputRenderedManifests(renderedManifests.String(), filepath, out)
}

// exec executes the helm command, writing combined stdout/stderr to the provided writer
func (h *HelmDeployer) exec(ctx context.Context, out io.Writer, useSecrets bool, args ...string) error {
	if args[0] != "version" {
		args = append([]string{"--kube-context", h.kubeContext}, args...)
		args = append(args, h.Flags.Global...)

		if h.kubeConfig != "" {
			args = append(args, "--kubeconfig", h.kubeConfig)
		}

		if useSecrets {
			args = append([]string{"secrets"}, args...)
		}
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(cmd)
}

// deployRelease deploys a single release
func (h *HelmDeployer) deployRelease(ctx context.Context, out io.Writer, r latest.HelmRelease, builds []build.Artifact, valuesSet map[string]bool, helmVersion semver.Version) ([]Artifact, error) {
	releaseName, err := util.ExpandEnvTemplate(r.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot parse the release name template: %w", err)
	}

	opts := installOpts{
		releaseName: releaseName,
		upgrade:     true,
		flags:       h.Flags.Upgrade,
		force:       h.forceDeploy,
		chartPath:   r.ChartPath,
		helmVersion: helmVersion,
	}

	if h.namespace != "" {
		opts.namespace = h.namespace
	} else if r.Namespace != "" {
		opts.namespace = r.Namespace
	}

	if err := h.exec(ctx, ioutil.Discard, false, getArgs(helmVersion, releaseName, opts.namespace)...); err != nil {
		color.Yellow.Fprintf(out, "Helm release %s not installed. Installing...\n", releaseName)

		opts.upgrade = false
		opts.flags = h.Flags.Install
	} else {
		if r.UpgradeOnChange != nil && !*r.UpgradeOnChange {
			logrus.Infof("Release %s already installed...", releaseName)
			return []Artifact{}, nil
		} else if r.UpgradeOnChange == nil && r.Remote {
			logrus.Infof("Release %s not upgraded as it is remote...", releaseName)
			return []Artifact{}, nil
		}
	}

	// Only build local dependencies, but allow a user to skip them.
	if !r.SkipBuildDependencies && !r.Remote {
		logrus.Infof("Building helm dependencies...")

		if err := h.exec(ctx, out, false, "dep", "build", r.ChartPath); err != nil {
			return nil, fmt.Errorf("building helm dependencies: %w", err)
		}
	}

	// Dump overrides to a YAML file to pass into helm
	if len(r.Overrides.Values) != 0 {
		overrides, err := yaml.Marshal(r.Overrides)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal overrides to create overrides values.yaml: %w", err)
		}

		if err := ioutil.WriteFile(constants.HelmOverridesFilename, overrides, 0666); err != nil {
			return nil, fmt.Errorf("cannot create file %q: %w", constants.HelmOverridesFilename, err)
		}

		defer func() {
			os.Remove(constants.HelmOverridesFilename)
		}()
	}

	if r.Packaged != nil {
		chartPath, err := h.packageChart(ctx, r)
		if err != nil {
			return nil, fmt.Errorf("cannot package chart: %w", err)
		}

		opts.chartPath = chartPath
	}

	args, err := installArgs(r, builds, valuesSet, opts)
	if err != nil {
		return nil, fmt.Errorf("release args: %w", err)
	}

	err = h.exec(ctx, out, r.UseHelmSecrets, args...)
	if err != nil {
		return nil, fmt.Errorf("install: %w", err)
	}

	b, err := h.getRelease(ctx, helmVersion, releaseName, opts.namespace)
	if err != nil {
		return nil, fmt.Errorf("get release: %w", err)
	}

	artifacts := parseReleaseInfo(opts.namespace, bufio.NewReader(&b))
	return artifacts, nil
}

// getRelease confirms that a release is visible to helm
func (h *HelmDeployer) getRelease(ctx context.Context, helmVersion semver.Version, releaseName string, namespace string) (bytes.Buffer, error) {
	// Retry, because under Helm 2, at least, a release may not be immediately visible
	opts := backoff.NewExponentialBackOff()
	opts.MaxElapsedTime = 4 * time.Second
	var b bytes.Buffer

	err := backoff.Retry(
		func() error {
			if err := h.exec(ctx, &b, false, getArgs(helmVersion, releaseName, namespace)...); err != nil {
				logrus.Debugf("unable to get release: %v (may retry):\n%s", err, b.String())
				return err
			}
			return nil
		}, opts)

	logrus.Debug(b.String())

	return b, err
}

// binVer returns the version of the helm binary found in PATH. May be cached.
func (h *HelmDeployer) binVer(ctx context.Context) (semver.Version, error) {
	// Return the cached version value if non-zero
	if h.bV.Major != 0 && h.bV.Minor != 0 {
		return h.bV, nil
	}

	var b bytes.Buffer
	// Only 3.0.0-beta doesn't support --client
	if err := h.exec(ctx, &b, false, "version", "--client"); err != nil {
		return semver.Version{}, fmt.Errorf("helm version command failed %q: %w", b.String(), err)
	}
	raw := b.String()
	matches := versionRegex.FindStringSubmatch(raw)
	if len(matches) == 0 {
		return semver.Version{}, fmt.Errorf("unable to parse output: %q", raw)
	}

	v, err := semver.Make(matches[1])
	if err != nil {
		return semver.Version{}, fmt.Errorf("semver make %q: %w", matches[1], err)
	}

	h.bV = v
	return h.bV, nil
}

// installOpts are options to be passed to "helm install"
type installOpts struct {
	flags       []string
	releaseName string
	namespace   string
	chartPath   string
	upgrade     bool
	force       bool
	helmVersion semver.Version
}

// installArgs calculates the correct arguments to "helm install"
func installArgs(r latest.HelmRelease, builds []build.Artifact, valuesSet map[string]bool, o installOpts) ([]string, error) {
	var args []string
	if o.upgrade {
		args = append(args, "upgrade", o.releaseName)
		args = append(args, o.flags...)

		if o.force {
			args = append(args, "--force")
		}

		if r.RecreatePods {
			args = append(args, "--recreate-pods")
		}
	} else {
		args = append(args, "install")
		if o.helmVersion.LT(helm3Version) {
			args = append(args, "--name")
		}
		args = append(args, o.releaseName)
		args = append(args, o.flags...)
	}

	// There are 2 strategies:
	// 1) Deploy chart directly from filesystem path or from repository
	//    (like stable/kubernetes-dashboard). Version only applies to a
	//    chart from repository.
	// 2) Package chart into a .tgz archive with specific version and then deploy
	//    that packaged chart. This way user can apply any version and appVersion
	//    for the chart.
	if r.Packaged == nil && r.Version != "" {
		args = append(args, "--version", r.Version)
	}

	args = append(args, o.chartPath)

	if o.namespace != "" {
		args = append(args, "--namespace", o.namespace)
	}

	params, err := pairParamsToArtifacts(builds, r.ArtifactOverrides)
	if err != nil {
		return nil, fmt.Errorf("matching build results to chart values: %w", err)
	}

	if len(r.Overrides.Values) != 0 {
		args = append(args, "-f", constants.HelmOverridesFilename)
	}

	for k, v := range params {
		var value string

		cfg := r.ImageStrategy.HelmImageConfig.HelmConventionConfig

		value, err = imageSetFromConfig(cfg, k, v.Tag)
		if err != nil {
			return nil, err
		}

		valuesSet[v.Tag] = true
		args = append(args, "--set-string", value)
	}

	args, err = constructOverrideArgs(&r, builds, args, func(k string) {
		valuesSet[k] = true
	})
	if err != nil {
		return nil, err
	}

	if r.Wait {
		args = append(args, "--wait")
	}

	return args, nil
}

// constructOverrideArgs creates the command line arguments for overrides
func constructOverrideArgs(r *latest.HelmRelease, builds []build.Artifact, args []string, record func(string)) ([]string, error) {
	sortedKeys := make([]string, 0, len(r.SetValues))
	for k := range r.SetValues {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		record(r.SetValues[k])
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, r.SetValues[k]))
	}

	for k, v := range r.SetFiles {
		record(v)
		args = append(args, "--set-file", fmt.Sprintf("%s=%s", k, v))
	}

	envMap := map[string]string{}
	for idx, b := range builds {
		suffix := ""
		if idx > 0 {
			suffix = strconv.Itoa(idx + 1)
		}

		for k, v := range envVarForImage(b.ImageName, b.Tag) {
			envMap[k+suffix] = v
		}
	}
	logrus.Debugf("EnvVarMap: %+v\n", envMap)

	sortedKeys = make([]string, 0, len(r.SetValueTemplates))
	for k := range r.SetValueTemplates {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		v, err := util.ExpandEnvTemplate(r.SetValueTemplates[k], envMap)
		if err != nil {
			return nil, err
		}

		record(v)
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, v))
	}

	for _, v := range r.ValuesFiles {
		exp, err := homedir.Expand(v)
		if err != nil {
			return nil, fmt.Errorf("unable to expand %q: %w", v, err)
		}

		exp, err = util.ExpandEnvTemplate(exp, envMap)
		if err != nil {
			return nil, err
		}

		args = append(args, "-f", exp)
	}
	return args, nil
}

// getArgs calculates the correct arguments to "helm get"
func getArgs(v semver.Version, releaseName string, namespace string) []string {
	args := []string{"get"}
	if v.GTE(helm3Version) {
		args = append(args, "all")
		if namespace != "" {
			args = append(args, "--namespace", namespace)
		}
	}
	return append(args, releaseName)
}

// envVarForImage creates an environment map for an image and digest tag (fqn)
func envVarForImage(imageName string, digest string) map[string]string {
	customMap := map[string]string{
		"IMAGE_NAME": imageName,
		"DIGEST":     digest, // The `DIGEST` name is kept for compatibility reasons
	}

	// Standardize access to Image reference fields in templates
	ref, err := docker.ParseReference(digest)
	if err == nil {
		customMap[constants.ImageRef.Repo] = ref.BaseName
		customMap[constants.ImageRef.Tag] = ref.Tag
		customMap[constants.ImageRef.Digest] = ref.Digest
	} else {
		logrus.Warnf("unable to extract values for %v, %v and %v from image %v due to error:\n%v", constants.ImageRef.Repo, constants.ImageRef.Tag, constants.ImageRef.Digest, digest, err)
	}

	if digest == "" {
		return customMap
	}

	// DIGEST_ALGO and DIGEST_HEX are deprecated and will contain nonsense values
	names := strings.SplitN(digest, ":", 2)
	if len(names) >= 2 {
		customMap["DIGEST_ALGO"] = names[0]
		customMap["DIGEST_HEX"] = names[1]
	} else {
		customMap["DIGEST_HEX"] = digest
	}
	return customMap
}

// packageChart packages the chart and returns the path to the resulting chart archive
func (h *HelmDeployer) packageChart(ctx context.Context, r latest.HelmRelease) (string, error) {
	// Allow a test to sneak a predictable path in
	tmpDir := h.pkgTmpDir

	if tmpDir == "" {
		t, err := ioutil.TempDir("", "skaffold-helm")
		if err != nil {
			return "", fmt.Errorf("tempdir: %w", err)
		}
		tmpDir = t
	}

	args := []string{"package", r.ChartPath, "--destination", tmpDir}

	if r.Packaged.Version != "" {
		v, err := util.ExpandEnvTemplate(r.Packaged.Version, nil)
		if err != nil {
			return "", fmt.Errorf("packaged.version template: %w", err)
		}
		args = append(args, "--version", v)
	}

	if r.Packaged.AppVersion != "" {
		av, err := util.ExpandEnvTemplate(r.Packaged.AppVersion, nil)
		if err != nil {
			return "", fmt.Errorf("packaged.appVersion template: %w", err)
		}
		args = append(args, "--app-version", av)
	}

	buf := &bytes.Buffer{}

	if err := h.exec(ctx, buf, false, args...); err != nil {
		return "", fmt.Errorf("package chart into a .tgz archive: %v: %w", args, err)
	}

	output := strings.TrimSpace(buf.String())
	idx := strings.Index(output, tmpDir)

	if idx == -1 {
		return "", fmt.Errorf("unable to find %s in output: %s", tmpDir, output)
	}

	return output[idx:], nil
}

// imageSetFromConfig calculates the --set-string value from the helm config
func imageSetFromConfig(cfg *latest.HelmConventionConfig, valueName string, tag string) (string, error) {
	if cfg == nil {
		return fmt.Sprintf("%s=%s", valueName, tag), nil
	}

	ref, err := docker.ParseReference(tag)
	if err != nil {
		return "", fmt.Errorf("cannot parse the image reference %q: %w", tag, err)
	}

	var imageTag string
	if ref.Digest != "" {
		imageTag = fmt.Sprintf("%s@%s", ref.Tag, ref.Digest)
	} else {
		imageTag = ref.Tag
	}

	if cfg.ExplicitRegistry {
		if ref.Domain == "" {
			return "", fmt.Errorf("image reference %s has no domain", tag)
		}
		return fmt.Sprintf("%[1]s.registry=%[2]s,%[1]s.repository=%[3]s,%[1]s.tag=%[4]s", valueName, ref.Domain, ref.Path, imageTag), nil
	}

	return fmt.Sprintf("%[1]s.repository=%[2]s,%[1]s.tag=%[3]s", valueName, ref.BaseName, imageTag), nil
}

// pairParamsToArtifacts associates parameters to the build artifact it creates
func pairParamsToArtifacts(builds []build.Artifact, params map[string]string) (map[string]build.Artifact, error) {
	imageToBuildResult := map[string]build.Artifact{}
	for _, b := range builds {
		imageToBuildResult[b.ImageName] = b
	}

	paramToBuildResult := map[string]build.Artifact{}

	for param, imageName := range params {
		b, ok := imageToBuildResult[imageName]
		if !ok {
			return nil, fmt.Errorf("no build present for %s", imageName)
		}

		paramToBuildResult[param] = b
	}

	return paramToBuildResult, nil
}

func IsHelmChart(path string) bool {
	return filepath.Base(path) == "Chart.yaml"
}
