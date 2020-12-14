/*
Copyright 2020 The Skaffold Authors

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
	"strconv"

	"github.com/blang/semver"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// installOpts are options to be passed to "helm install"
type installOpts struct {
	flags        []string
	releaseName  string
	namespace    string
	chartPath    string
	upgrade      bool
	force        bool
	helmVersion  semver.Version
	postRenderer string
}

// constructOverrideArgs creates the command line arguments for overrides
func constructOverrideArgs(r *latest.HelmRelease, builds []build.Artifact, args []string, record func(string)) ([]string, error) {
	for _, k := range sortKeys(r.SetValues) {
		record(r.SetValues[k])
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, r.SetValues[k]))
	}

	for _, k := range sortKeys(r.SetFiles) {
		exp, err := homedir.Expand(r.SetFiles[k])
		if err != nil {
			return nil, fmt.Errorf("unable to expand %q: %w", r.SetFiles[k], err)
		}
		record(exp)
		args = append(args, "--set-file", fmt.Sprintf("%s=%s", k, exp))
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

	for _, k := range sortKeys(r.SetValueTemplates) {
		v, err := util.ExpandEnvTemplate(r.SetValueTemplates[k], envMap)
		if err != nil {
			return nil, err
		}
		expandedKey, err := util.ExpandEnvTemplate(k, envMap)
		if err != nil {
			return nil, err
		}

		record(v)
		args = append(args, "--set", fmt.Sprintf("%s=%s", expandedKey, v))
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
func getArgs(releaseName string, namespace string) []string {
	args := []string{"get", "all"}
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}
	return append(args, releaseName)
}

// installArgs calculates the correct arguments to "helm install"
func (h *Deployer) installArgs(r latest.HelmRelease, builds []build.Artifact, valuesSet map[string]bool, o installOpts) ([]string, error) {
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
		args = append(args, o.releaseName)
		args = append(args, o.flags...)
	}

	if o.postRenderer != "" {
		args = append(args, "--post-renderer")
		args = append(args, o.postRenderer)
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

	if r.CreateNamespace != nil && *r.CreateNamespace && !o.upgrade {
		if o.helmVersion.LT(helm32Version) {
			return nil, createNamespaceErr(h.bV.String())
		}
		args = append(args, "--create-namespace")
	}

	params, err := pairParamsToArtifacts(builds, r.ArtifactOverrides)
	if err != nil {
		return nil, err
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
