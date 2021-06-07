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
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// DeployerMux forwards all method calls to the deployers it contains.
// When encountering an error, it aborts and returns the error. Otherwise,
// it collects the results and returns it in bulk.
type DeployerMux []Deployer

func (m DeployerMux) Deploy(ctx context.Context, w io.Writer, as []graph.Artifact) ([]string, error) {
	seenNamespaces := util.NewStringSet()

	for i, deployer := range m {
		eventV2.DeployInProgress(i)
		w = output.WithEventContext(w, constants.Deploy, strconv.Itoa(i), "skaffold")
		ctx, endTrace := instrumentation.StartTrace(ctx, "Deploy")

		namespaces, err := deployer.Deploy(ctx, w, as)
		if err != nil {
			eventV2.DeployFailed(i, err)
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}
		seenNamespaces.Insert(namespaces...)

		eventV2.DeploySucceeded(i)
		endTrace()
	}

	return seenNamespaces.ToList(), nil
}

func (m DeployerMux) Dependencies() ([]string, error) {
	deps := util.NewStringSet()
	for _, deployer := range m {
		result, err := deployer.Dependencies()
		if err != nil {
			return nil, err
		}
		deps.Insert(result...)
	}
	return deps.ToList(), nil
}

func (m DeployerMux) Cleanup(ctx context.Context, w io.Writer) error {
	for _, deployer := range m {
		ctx, endTrace := instrumentation.StartTrace(ctx, "Cleanup")
		if err := deployer.Cleanup(ctx, w); err != nil {
			return err
		}
		endTrace()
	}
	return nil
}

func (m DeployerMux) Render(ctx context.Context, w io.Writer, as []graph.Artifact, offline bool, filepath string) error {
	resources, buf := []string{}, &bytes.Buffer{}
	for _, deployer := range m {
		ctx, endTrace := instrumentation.StartTrace(ctx, "Render")
		buf.Reset()
		if err := deployer.Render(ctx, buf, as, offline, "" /* never write to files */); err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return err
		}
		resources = append(resources, buf.String())
		endTrace()
	}

	allResources := strings.Join(resources, "\n---\n")
	return manifest.Write(strings.TrimSpace(allResources), filepath, w)
}
