/*
Copyright 2025 The Skaffold Authors

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
	"fmt"
	"os"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

func (h Helm) depBuildArgs(chartPath string) []string {
	args := []string{"dep", "build", chartPath}
	args = append(args, h.config.Flags.DepBuild...)
	return args
}

func createOverridesValuesFile(r latest.HelmRelease) (string, error) {
	if len(r.Overrides.Values) == 0 {
		return "", nil
	}

	overrides, err := yaml.Marshal(r.Overrides)
	if err != nil {
		return "", helm.UserErr("cannot marshal overrides to create overrides values.yaml", err)
	}

	if err := os.WriteFile(constants.HelmOverridesFilename, overrides, 0o666); err != nil {
		return "", helm.UserErr(fmt.Sprintf("cannot create file %q", constants.HelmOverridesFilename), err)
	}

	defer func() {
		os.Remove(constants.HelmOverridesFilename)
	}()

	return constants.HelmOverridesFilename, nil
}

func (h Helm) templateArgs(releaseName string, release latest.HelmRelease, builds []graph.Artifact, namespace string, additionalArgs []string) ([]string, error) {
	args := []string{"template", releaseName, helm.ChartSource(release)}
	args = append(args, h.config.Flags.Template...)
	args = append(args, additionalArgs...)

	overrideArgs, overrideArgsErr := helm.ConstructOverrideArgs(&release, builds, args, h.manifestOverrides)
	if overrideArgsErr != nil {
		return nil, helm.UserErr("construct override args", overrideArgsErr)
	}
	args = overrideArgs

	overridesFile, overridesFileErr := createOverridesValuesFile(release)
	if overridesFileErr != nil {
		return nil, overridesFileErr
	}
	if overridesFile != "" {
		args = append(args, "-f", overridesFile)
	}

	if release.Packaged == nil && release.Version != "" {
		args = append(args, "--version", release.Version)
	}

	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}
	if release.Repo != "" {
		args = append(args, "--repo")
		args = append(args, release.Repo)
	}
	if release.SkipTests {
		args = append(args, "--skip-tests")
	}

	return args, nil
}
