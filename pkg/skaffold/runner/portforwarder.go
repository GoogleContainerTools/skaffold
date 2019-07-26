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

package runner

import (
	"fmt"
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func (r *SkaffoldRunner) createForwarder(out io.Writer) {
	label := createLabelQuery(r.runCtx.Cfg.Deploy.HelmDeploy, r.defaultLabeller.K8sManagedByLabelKeyValueString())
	r.forwarderManager = portforward.NewForwarderManager(out, r.imageList, r.runCtx.Namespaces, label, r.runCtx.Opts.PortForward, r.portForwardResources)
}

func createLabelQuery(helmDeploy *latest.HelmDeploy, label string) string {
	if helmDeploy == nil {
		return label
	}
	names := make([]string, len(helmDeploy.Releases))
	for i, release := range helmDeploy.Releases {
		names[i] = release.Name
	}
	return fmt.Sprintf("release in (%s)", strings.Join(names, ","))
}
