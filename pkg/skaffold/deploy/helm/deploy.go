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

package helm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/blang/semver"
	backoff "github.com/cenkalti/backoff/v4"
	shell "github.com/kballard/go-shellquote"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
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
	helm3Version  = semver.MustParse("3.0.0-beta.0")
	helm32Version = semver.MustParse("3.2.0")

	// helm31Version represents the version cut-off for helm3.1 post-renderer behavior
	helm31Version = semver.MustParse("3.1.0")

	// error to throw when helm version can't be determined
	versionErrorString = "failed to determine binary version: %w"

	// osExecutable allows for replacing the skaffold binary for testing purposes
	osExecutable = os.Executable
)

// Deployer deploys workflows using the helm CLI
type Deployer struct {
	*latest.HelmDeploy

	kubeContext string
	kubeConfig  string
	namespace   string
	configFile  string

	// packaging temporary directory, used for predictable test output
	pkgTmpDir string

	labels map[string]string

	forceDeploy bool
	enableDebug bool

	// bV is the helm binary version
	bV semver.Version
}

// NewDeployer returns a configured Deployer.  Returns an error if current version of helm is less than 3.0.0.
func NewDeployer(cfg kubectl.Config, labels map[string]string, h *latest.HelmDeploy) (*Deployer, error) {
	hv, err := binVer()
	if err != nil {
		return nil, versionGetErr(err)
	}

	if hv.LT(helm3Version) {
		return nil, minVersionErr()
	}

	return &Deployer{
		HelmDeploy:  h,
		kubeContext: cfg.GetKubeContext(),
		kubeConfig:  cfg.GetKubeConfig(),
		namespace:   cfg.GetKubeNamespace(),
		forceDeploy: cfg.ForceDeploy(),
		configFile:  cfg.ConfigurationFile(),
		labels:      labels,
		bV:          hv,
		enableDebug: cfg.Mode() == config.RunModes.Debug,
	}, nil
}

// Deploy deploys the build results to the Kubernetes cluster
func (h *Deployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]string, error) {
	logrus.Infof("Deploying with helm v%s ...", h.bV)

	var dRes []types.Artifact
	nsMap := map[string]struct{}{}
	valuesSet := map[string]bool{}

	// Deploy every release
	for _, r := range h.Releases {
		releaseName, err := util.ExpandEnvTemplateOrFail(r.Name, nil)
		if err != nil {
			return nil, userErr(fmt.Sprintf("cannot expand release name %q", r.Name), err)
		}
		results, err := h.deployRelease(ctx, out, releaseName, r, builds, valuesSet, h.bV)
		if err != nil {
			return nil, userErr(fmt.Sprintf("deploying %q", releaseName), err)
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

	if err := label.Apply(ctx, h.labels, dRes); err != nil {
		return nil, helmLabelErr(fmt.Errorf("adding labels: %w", err))
	}

	// Collect namespaces in a string
	var namespaces []string
	for ns := range nsMap {
		namespaces = append(namespaces, ns)
	}
	return namespaces, nil
}

// Dependencies returns a list of files that the deployer depends on.
func (h *Deployer) Dependencies() ([]string, error) {
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
		}

		// We can always add a dependency if it is not contained in our chartDepsDirs.
		// However, if the file is in our chartDepsDir, we can only include the file
		// if we are not running the helm dep build phase, as that modifies files inside
		// the chartDepsDir and results in an infinite build loop.
		// We additionally exclude ChartFile.lock,
		// since it also gets modified during a `helm dep build`.
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
			return deps, userErr("issue walking releases", err)
		}
	}
	sort.Strings(deps)
	return deps, nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (h *Deployer) Cleanup(ctx context.Context, out io.Writer) error {
	for _, r := range h.Releases {
		releaseName, err := util.ExpandEnvTemplateOrFail(r.Name, nil)
		if err != nil {
			return fmt.Errorf("cannot parse the release name template: %w", err)
		}

		namespace, err := h.releaseNamespace(r)
		if err != nil {
			return err
		}

		args := []string{"delete", releaseName}
		if namespace != "" {
			args = append(args, "--namespace", namespace)
		}
		if err := h.exec(ctx, out, false, nil, args...); err != nil {
			return deployerr.CleanupErr(err)
		}
	}
	return nil
}

// Render generates the Kubernetes manifests and writes them out
func (h *Deployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, offline bool, filepath string) error {
	renderedManifests := new(bytes.Buffer)

	for _, r := range h.Releases {
		args := []string{"template", r.ChartPath}

		args = append(args[:1], append([]string{r.Name}, args[1:]...)...)

		params, err := pairParamsToArtifacts(builds, r.ArtifactOverrides)
		if err != nil {
			return err
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
			return userErr("construct override args", err)
		}

		namespace, err := h.releaseNamespace(r)
		if err != nil {
			return err
		}
		if namespace != "" {
			args = append(args, "--namespace", namespace)
		}

		outBuffer := new(bytes.Buffer)
		if err := h.exec(ctx, outBuffer, false, nil, args...); err != nil {
			return userErr("std out err", fmt.Errorf(outBuffer.String()))
		}
		renderedManifests.Write(outBuffer.Bytes())
	}

	return manifest.Write(renderedManifests.String(), filepath, out)
}

// deployRelease deploys a single release
func (h *Deployer) deployRelease(ctx context.Context, out io.Writer, releaseName string, r latest.HelmRelease, builds []build.Artifact, valuesSet map[string]bool, helmVersion semver.Version) ([]types.Artifact, error) {
	var err error
	opts := installOpts{
		releaseName: releaseName,
		upgrade:     true,
		flags:       h.Flags.Upgrade,
		force:       h.forceDeploy,
		chartPath:   r.ChartPath,
		helmVersion: helmVersion,
	}

	var installEnv []string
	if h.enableDebug {
		if h.bV.LT(helm31Version) {
			return nil, fmt.Errorf("debug requires at least Helm 3.1 (current: %v)", h.bV)
		}
		var binary string
		if binary, err = osExecutable(); err != nil {
			return nil, fmt.Errorf("cannot locate this Skaffold binary: %w", err)
		}
		opts.postRenderer = binary

		var buildsFile string
		if len(builds) > 0 {
			var cleanup func()
			buildsFile, cleanup, err = writeBuildArtifacts(builds)
			if err != nil {
				return nil, fmt.Errorf("could not write build-artifacts: %w", err)
			}
			defer cleanup()
		}

		cmdLine := h.generateSkaffoldDebugFilter(buildsFile)

		// need to include current environment, specifically for HOME to lookup ~/.kube/config
		env := util.EnvSliceToMap(util.OSEnviron(), "=")
		env["SKAFFOLD_CMDLINE"] = shell.Join(cmdLine...)
		env["SKAFFOLD_FILENAME"] = h.configFile
		installEnv = util.EnvMapToSlice(env, "=")
	}

	opts.namespace, err = h.releaseNamespace(r)
	if err != nil {
		return nil, err
	}

	if err := h.exec(ctx, ioutil.Discard, false, nil, getArgs(releaseName, opts.namespace)...); err != nil {
		color.Yellow.Fprintf(out, "Helm release %s not installed. Installing...\n", releaseName)

		opts.upgrade = false
		opts.flags = h.Flags.Install
	} else {
		if r.UpgradeOnChange != nil && !*r.UpgradeOnChange {
			logrus.Infof("Release %s already installed...", releaseName)
			return []types.Artifact{}, nil
		} else if r.UpgradeOnChange == nil && r.Remote {
			logrus.Infof("Release %s not upgraded as it is remote...", releaseName)
			return []types.Artifact{}, nil
		}
	}

	// Only build local dependencies, but allow a user to skip them.
	if !r.SkipBuildDependencies && !r.Remote {
		logrus.Infof("Building helm dependencies...")

		if err := h.exec(ctx, out, false, nil, "dep", "build", r.ChartPath); err != nil {
			return nil, userErr("building helm dependencies", err)
		}
	}

	// Dump overrides to a YAML file to pass into helm
	if len(r.Overrides.Values) != 0 {
		overrides, err := yaml.Marshal(r.Overrides)
		if err != nil {
			return nil, userErr("cannot marshal overrides to create overrides values.yaml", err)
		}

		if err := ioutil.WriteFile(constants.HelmOverridesFilename, overrides, 0666); err != nil {
			return nil, userErr(fmt.Sprintf("cannot create file %q", constants.HelmOverridesFilename), err)
		}

		defer func() {
			os.Remove(constants.HelmOverridesFilename)
		}()
	}

	if r.Packaged != nil {
		chartPath, err := h.packageChart(ctx, r)
		if err != nil {
			return nil, userErr("cannot package chart", err)
		}

		opts.chartPath = chartPath
	}

	args, err := h.installArgs(r, builds, valuesSet, opts)
	if err != nil {
		return nil, userErr("release args", err)
	}

	err = h.exec(ctx, out, r.UseHelmSecrets, installEnv, args...)
	if err != nil {
		return nil, userErr("install", err)
	}

	b, err := h.getRelease(ctx, releaseName, opts.namespace)
	if err != nil {
		return nil, userErr("get release", err)
	}

	artifacts := parseReleaseInfo(opts.namespace, bufio.NewReader(&b))
	return artifacts, nil
}

// getRelease confirms that a release is visible to helm
func (h *Deployer) getRelease(ctx context.Context, releaseName string, namespace string) (bytes.Buffer, error) {
	// Retry, because sometimes a release may not be immediately visible
	opts := backoff.NewExponentialBackOff()
	opts.MaxElapsedTime = 4 * time.Second
	var b bytes.Buffer

	err := backoff.Retry(
		func() error {
			if err := h.exec(ctx, &b, false, nil, getArgs(releaseName, namespace)...); err != nil {
				logrus.Debugf("unable to get release: %v (may retry):\n%s", err, b.String())
				return err
			}
			return nil
		}, opts)

	logrus.Debug(b.String())

	return b, err
}

// packageChart packages the chart and returns the path to the resulting chart archive
func (h *Deployer) packageChart(ctx context.Context, r latest.HelmRelease) (string, error) {
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

	if err := h.exec(ctx, buf, false, nil, args...); err != nil {
		return "", fmt.Errorf("package chart into a .tgz archive: %v: %w", args, err)
	}

	output := strings.TrimSpace(buf.String())
	idx := strings.Index(output, tmpDir)

	if idx == -1 {
		return "", fmt.Errorf("unable to find %s in output: %s", tmpDir, output)
	}

	return output[idx:], nil
}
