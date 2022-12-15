/*
Copyright 2021 The Skaffold Authors

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
	"context"
	"fmt"
	"io"
	"time"

	deployutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
)

// Apply sends Kubernetes manifests to the cluster.
func (r *SkaffoldRunner) Apply(ctx context.Context, out io.Writer) error {
	var manifests manifest.ManifestList
	var err error
	manifests, err = deployutil.GetManifestsFromHydratedManifests(ctx, r.runCtx.HydratedManifests())
	manifestsByConfig := manifest.NewManifestListByConfig()
	manifestsByConfig.Add(r.deployer.ConfigName(), manifests)

	if err != nil {
		return fmt.Errorf("getting manifests from hydrated manifests: %w", err)
	}
	if err := r.applyResources(ctx, out, nil, nil, manifestsByConfig); err != nil {
		return err
	}

	statusCheckOut, postStatusCheckFn, err := deployutil.WithStatusCheckLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	postStatusCheckFn()
	if err != nil {
		return err
	}
	sErr := r.deployer.GetStatusMonitor().Check(ctx, statusCheckOut)
	return sErr
}

func (r *SkaffoldRunner) applyResources(ctx context.Context, out io.Writer, artifacts, _ []graph.Artifact, list manifest.ManifestListByConfig) error {
	deployOut, postDeployFn, err := deployutil.WithLogFile(time.Now().Format(deployutil.TimeFormat)+".log", out, r.runCtx.Muted())
	if err != nil {
		return err
	}

	event.DeployInProgress()
	ctx, endTrace := instrumentation.StartTrace(ctx, "applyResources_Deploying")
	defer endTrace()
	err = r.deployer.Deploy(ctx, deployOut, artifacts, list)
	postDeployFn()
	if err != nil {
		event.DeployFailed(err)
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	r.deployManifests = list
	event.DeployComplete()
	return nil
}
