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
	"os"
	"strings"

	"github.com/blang/semver"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/gcs"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/helm"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// installOpts are options to be passed to "helm install"
type installOpts struct {
	flags       []string
	releaseName string
	namespace   string
	chartPath   string
	upgrade     bool
	force       bool
	helmVersion semver.Version
	repo        string
	version     string
}

// installArgs calculates the correct arguments to "helm install"
func (h *Deployer) installArgs(r latest.HelmRelease, builds []graph.Artifact, o installOpts) ([]string, error) {
	var args []string
	if o.upgrade {
		args = append(args, "upgrade", o.releaseName)
		processedFlags, err := processGCSFlags(o.flags)
		if err != nil {
			return nil, err
		}
		args = append(args, processedFlags...)

		if o.force {
			args = append(args, "--force")
		}

		if r.RecreatePods {
			args = append(args, "--recreate-pods")
		}
	} else {
		args = append(args, "install")
		args = append(args, o.releaseName)
		processedFlags, err := processGCSFlags(o.flags)
		if err != nil {
			return nil, err
		}
		args = append(args, processedFlags...)
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

	if o.namespace != "" {
		args = append(args, "--namespace", o.namespace)
	}

	if o.repo != "" {
		args = append(args, "--repo")
		args = append(args, o.repo)
	}

	if r.CreateNamespace != nil && *r.CreateNamespace && !o.upgrade {
		if o.helmVersion.LT(helm32Version) {
			return nil, helm.CreateNamespaceErr(h.helmVersion.String())
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

// extractValueFileFromGCSFunc is a function variable that can be mocked in tests
var extractValueFileFromGCSFunc = func(gcsPath, tempDir string, gcs gcs.Gsutil) (string, error) {
	return helm.ExtractValueFileFromGCS(gcsPath, tempDir, gcs)
}

// processGCSFlags processes helm flags to handle gs:// URLs in --values flags
func processGCSFlags(flags []string) ([]string, error) {
	if len(flags) == 0 {
		return flags, nil
	}

	var processedFlags []string
	gcs := gcs.NewGsutil()

	for i := 0; i < len(flags); i++ {
		flag := flags[i]

		// Check for --values flag with equals sign (--values=gs://...)
		if strings.HasPrefix(flag, "--values=") {
			value := strings.TrimPrefix(flag, "--values=")
			if strings.HasPrefix(value, "gs://") {
				tempDir, err := os.MkdirTemp("", "helm_values_from_gcs")
				if err != nil {
					return nil, fmt.Errorf("failed to create temp directory: %w", err)
				}
				processedValue, err := extractValueFileFromGCSFunc(value, tempDir, gcs)
				if err != nil {
					return nil, err
				}
				processedFlags = append(processedFlags, "--values="+processedValue)
			} else {
				processedFlags = append(processedFlags, flag)
			}
		} else if flag == "--values" || flag == "-f" {
			// Check for --values flag with separate argument (--values gs://... or -f gs://...)
			if i+1 < len(flags) {
				nextFlag := flags[i+1]
				if strings.HasPrefix(nextFlag, "gs://") {
					tempDir, err := os.MkdirTemp("", "helm_values_from_gcs")
					if err != nil {
						return nil, fmt.Errorf("failed to create temp directory: %w", err)
					}
					processedValue, err := extractValueFileFromGCSFunc(nextFlag, tempDir, gcs)
					if err != nil {
						return nil, err
					}
					processedFlags = append(processedFlags, flag, processedValue)
					i++ // Skip the next flag since we processed it
				} else {
					processedFlags = append(processedFlags, flag, nextFlag)
					i++
				}
			} else {
				processedFlags = append(processedFlags, flag)
			}
		} else {
			processedFlags = append(processedFlags, flag)
		}
	}

	return processedFlags, nil
}
