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
	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
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
	repo         string
	version      string
}

// installArgs calculates the correct arguments to "helm install"
func (h *Deployer) installArgs(r latest.HelmRelease, builds []graph.Artifact, o installOpts) ([]string, error) {
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

	// There are 2 strategies:
	// 1) Deploy chart directly from filesystem path or from repository
	//    (like stable/kubernetes-dashboard). Version only applies to a
	//    chart from repository.
	// 2) Package chart into a .tgz archive with specific version and then deploy
	//    that packaged chart. This way user can apply any version and appVersion
	//    for the chart.
	if r.Packaged == nil && o.version != "" {
		args = append(args, "--version", o.version)
	}

	args = append(args, o.chartPath)

	if o.postRenderer != "" {
		args = append(args, "--post-renderer")
		args = append(args, o.postRenderer)
	}

	if o.namespace != "" {
		args = append(args, "--namespace", o.namespace)
	}

	if o.repo != "" {
		args = append(args, "--repo")
		args = append(args, o.repo)
	}

	if r.CreateNamespace != nil && *r.CreateNamespace && !o.upgrade {
		if o.helmVersion.LT(helm32Version) {
			return nil, helm.CreateNamespaceErr(h.bV.String())
		}
		args = append(args, "--create-namespace")
	}

	args, err := helm.ConstructOverrideArgs(&r, builds, args, nil)
	if err != nil {
		return nil, err
	}

	if len(r.Overrides.Values) != 0 {
		args = append(args, "-f", constants.HelmOverridesFilename)
	}

	if r.Wait {
		args = append(args, "--wait")
	}

	return args, nil
}
