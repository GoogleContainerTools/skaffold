/*
Copyright 2022 The Skaffold Authors

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

package platform

import (
	"context"
	"fmt"
	"sort"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	coreV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

var (
	// for testing
	getClusterPlatforms = GetClusterPlatforms
	getHostMatcher      = func() Matcher { return Host }
)

type Resolver struct {
	platformsByImageName map[string]Matcher
}

func (r Resolver) GetPlatforms(imageName string) Matcher {
	if r.platformsByImageName == nil {
		return Matcher{}
	}
	return r.platformsByImageName[imageName]
}

func NewResolver(ctx context.Context, pipelines []latest.Pipeline, cliPlatformsSelection []string, runMode config.RunMode, kubeContext string) (Resolver, error) {
	r := Resolver{}
	r.platformsByImageName = make(map[string]Matcher)

	var fromCli, fromClusterNodes Matcher
	var err error
	fromCli, err = Parse(cliPlatformsSelection)
	if err != nil {
		return r, fmt.Errorf("failed to parse platforms: %w", err)
	}
	log.Entry(ctx).Debugf("CLI platforms provided: %q", fromCli)
	instrumentation.AddCliBuildTargetPlatforms(fromCli.String())

	if runMode == config.RunModes.Dev || runMode == config.RunModes.Debug || runMode == config.RunModes.Run {
		fromClusterNodes, err = getClusterPlatforms(ctx, kubeContext, runMode == config.RunModes.Dev || runMode == config.RunModes.Debug)
		if err != nil {
			log.Entry(ctx).Debugf("failed to get cluster node details: %v", err)
			log.Entry(ctx).Warnln("failed to detect active kubernetes cluster node platform. Specify the correct build platform in the `skaffold.yaml` file or using the `--platform` flag")
		}
	}
	log.Entry(ctx).Debugf("platforms detected from active kubernetes cluster nodes: %q", fromClusterNodes)
	instrumentation.AddDeployNodePlatforms(fromClusterNodes.String())
	for _, pipeline := range pipelines {
		platforms := fromCli
		if platforms.IsEmpty() {
			platforms, err = Parse(pipeline.Build.Platforms)
			if err != nil {
				return r, fmt.Errorf("failed to parse platforms: %w", err)
			}
		}
		if fromClusterNodes.IsNotEmpty() {
			if platforms.IsEmpty() {
				platforms = fromClusterNodes
			} else if p := platforms.Intersect(fromClusterNodes); p.IsNotEmpty() {
				platforms = p
			} else {
				log.Entry(ctx).Warnf("build target platforms %q do not match active kubernetes cluster node platforms %q", platforms, fromClusterNodes)
			}
		}
		instrumentation.AddResolvedBuildTargetPlatforms(platforms.String())
		for _, artifact := range pipeline.Build.Artifacts {
			pl := platforms
			constraints, err := Parse(artifact.Platforms)
			if err != nil {
				return r, fmt.Errorf("failed to parse platforms: %w", err)
			}
			if constraints.IsNotEmpty() {
				if pl.IsEmpty() {
					pl = constraints
				} else if pl = pl.Intersect(constraints); pl.IsEmpty() {
					return r, fmt.Errorf("build target platforms %q do not match platform constraints %q defined for artifact %q", platforms, artifact.Platforms, artifact.ImageName)
				}
			}

			r.platformsByImageName[artifact.ImageName] = pl
			log.Entry(ctx).Debugf("platforms selected for artifact %q: %q", artifact.ImageName, pl)
		}
	}
	return r, nil
}

// GetClusterPlatforms returns the platforms for the active kubernetes cluster.
// For `dev` or `debug` if there are multiple cluster node platforms, then it returns only one selection, preferably matching the host platform.
func GetClusterPlatforms(ctx context.Context, kContext string, isDevOrDebug bool) (Matcher, error) {
	client, err := kubernetesclient.Client(kContext)
	if err != nil {
		return Matcher{}, fmt.Errorf("failed to determine kubernetes cluster node platforms: %w", err)
	}
	nodes, err := client.CoreV1().Nodes().List(ctx, coreV1.ListOptions{})
	if nodes == nil || err != nil {
		return Matcher{}, fmt.Errorf("failed to determine kubernetes cluster node platforms: %w", err)
	}
	set := make(map[string]v1.Platform)
	for _, n := range nodes.Items {
		pl := v1.Platform{
			Architecture: n.Status.NodeInfo.Architecture,
			OS:           n.Status.NodeInfo.OperatingSystem,
		}
		set[Format(pl)] = pl
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys) // sort keys to have a deterministic selection
	var m Matcher
	for _, key := range keys {
		m.Platforms = append(m.Platforms, set[key])
	}

	if !isDevOrDebug || len(m.Platforms) <= 1 {
		return m, nil
	}

	// If there's more than 1 node platform type and we're running `dev` or `debug`,
	// then select the platform type that matches the host platform.
	// If there's no match then just return the first node platform type found.
	filtered := m.Intersect(getHostMatcher())
	if len(filtered.Platforms) == 0 {
		m.Platforms = m.Platforms[0:1] // guaranteed to have at least 1 element
		return m, nil
	}
	return filtered, nil
}
