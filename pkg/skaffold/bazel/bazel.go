/*
Copyright 2018 Google LLC

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

package bazel

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

type BazelDependencyResolver struct{}

const sourceQuery = "kind('source file', deps('%s'))"

func (*BazelDependencyResolver) GetDependencies(a *v1alpha2.Artifact) ([]string, error) {
	cmd := exec.Command("bazel", "query", fmt.Sprintf(sourceQuery, a.BazelArtifact.BuildTarget), "--noimplicit_deps", "--order_output=no")
	stdout, stderr, err := util.RunCommand(cmd, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "stdout: %s stderr: %s", stdout, stderr)
	}
	labels := strings.Split(string(stdout), "\n")
	var deps []string
	for _, l := range labels {
		if strings.HasPrefix(l, "@") {
			continue
		}
		if strings.HasPrefix(l, "//external") {
			continue
		}
		if l == "" {
			continue
		}
		dep := strings.TrimPrefix(l, "//:")
		deps = append(deps, dep)
	}
	return deps, nil
}
