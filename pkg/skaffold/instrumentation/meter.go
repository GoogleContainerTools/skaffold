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

package instrumentation

import (
	"net/http"
	"runtime"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringset"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yamltags"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

const (
	gke    = "gke"
	others = "others"
)

var (
	meter = skaffoldMeter{
		OS:                 runtime.GOOS,
		Arch:               runtime.GOARCH,
		EnumFlags:          map[string]string{},
		Builders:           map[string]int{},
		BuildDependencies:  map[string]int{},
		BuildWithPlatforms: map[string]int{},
		SyncType:           map[string]bool{},
		Hooks:              map[HookPhase]int{},
		DevIterations:      []devIteration{},
		StartTime:          time.Now(),
		Version:            version.Get().Version,
		ExitCode:           0,
		ErrorCode:          proto.StatusCode_OK,
	}
	MeteredCommands     = stringset.New()
	doesBuild           = stringset.New()
	doesDeploy          = stringset.New()
	initExporter        = initCloudMonitoringExporterMetrics
	isOnline            bool
	ShouldExportMetrics bool
)

func init() {
	MeteredCommands.Insert("apply", "build", "delete", "deploy", "dev", "debug", "filter", "generate_pipeline", "render", "run", "test")
	doesBuild.Insert("build", "render", "dev", "debug", "run")
	doesDeploy.Insert("apply", "deploy", "dev", "debug", "run")
}

// SetOnlineStatus issues a GET request to see if the user is online.
// http://clients3.google.com/generate_204 is a well-known URL that returns an empty page and HTTP status 204
// More info can be found here: https://www.chromium.org/chromium-os/chromiumos-design-docs/network-portal-detection
func SetOnlineStatus() {
	go func() {
		if ShouldExportMetrics {
			r, err := http.Get("http://clients3.google.com/generate_204")
			if err == nil {
				r.Body.Close()
				isOnline = true
			}
		}
	}()
}

func InitMeterFromConfig(configs []*latest.SkaffoldConfig, user, deployCtx string) {
	var platforms []string
	for _, config := range configs {
		pl := yamltags.GetYamlTag(config.Build.BuildType)
		if !stringslice.Contains(platforms, pl) {
			platforms = append(platforms, pl)
		}
		for _, artifact := range config.Pipeline.Build.Artifacts {
			meter.Builders[yamltags.GetYamlTag(artifact.ArtifactType)]++
			if len(artifact.Dependencies) > 0 {
				meter.BuildDependencies[yamltags.GetYamlTag(artifact.ArtifactType)]++
			}
			if len(config.Pipeline.Build.Platforms) > 0 || len(artifact.Platforms) > 0 {
				meter.BuildWithPlatforms[yamltags.GetYamlTag(artifact.ArtifactType)]++
			}
			meter.Hooks[HookPhases.PreBuild] += len(artifact.LifecycleHooks.PreHooks)
			meter.Hooks[HookPhases.PostBuild] += len(artifact.LifecycleHooks.PostHooks)

			if artifact.Sync != nil {
				meter.SyncType[yamltags.GetYamlTag(artifact.Sync)] = true
				meter.Hooks[HookPhases.PreSync] += len(artifact.Sync.LifecycleHooks.PreHooks)
				meter.Hooks[HookPhases.PostSync] += len(artifact.Sync.LifecycleHooks.PostHooks)
			}
		}
		meter.Deployers = append(meter.Deployers, yamltags.GetYamlKeys(config.Deploy.DeployType)...)
		if h := config.Deploy.LegacyHelmDeploy; h != nil {
			meter.HelmReleasesCount = len(h.Releases)
		}
		if k := config.Deploy.KubectlDeploy; k != nil {
			meter.Hooks[HookPhases.PreDeploy] += len(k.LifecycleHooks.PreHooks)
			meter.Hooks[HookPhases.PostDeploy] += len(k.LifecycleHooks.PostHooks)
		}
		meter.BuildArtifacts += len(config.Pipeline.Build.Artifacts)
	}
	meter.PlatformType = strings.Join(platforms, ":")
	meter.ConfigCount = len(configs)
	meter.User = strings.ToLower(user)
	meter.ClusterType = getClusterType(deployCtx)
}

func SetCommand(cmd string) {
	if MeteredCommands.Contains(cmd) {
		meter.Command = cmd
	}
}

func SetErrorCode(errorCode proto.StatusCode) {
	meter.ErrorCode = errorCode
}

func AddDevIteration(intent string) {
	meter.DevIterations = append(meter.DevIterations, devIteration{Intent: intent})
}

func AddDevIterationErr(i int, errorCode proto.StatusCode) {
	if len(meter.DevIterations) == i {
		meter.DevIterations = append(meter.DevIterations, devIteration{Intent: "error"})
	}
	meter.DevIterations[i].ErrorCode = errorCode
}

func AddResourceFilter(source string, typ string) {
	meter.ResourceFilters = append(meter.ResourceFilters, resourceFilter{Source: source, Type: typ})
}

func AddFlag(flag *flag.Flag) {
	if flag.Changed {
		meter.EnumFlags[flag.Name] = flag.Value.String()
	}
}

func AddResolvedBuildTargetPlatforms(platforms string) {
	platforms = strings.ReplaceAll(platforms, ",", ";")
	if !stringslice.Contains(meter.ResolvedBuildTargetPlatforms, platforms) {
		meter.ResolvedBuildTargetPlatforms = append(meter.ResolvedBuildTargetPlatforms, platforms)
	}
}

func AddCliBuildTargetPlatforms(platforms string) {
	platforms = strings.ReplaceAll(platforms, ",", ";")
	meter.CliBuildTargetPlatforms = platforms
}

func AddDeployNodePlatforms(platforms string) {
	platforms = strings.ReplaceAll(platforms, ",", ";")
	meter.DeployNodePlatforms = platforms
}

func getClusterType(deployCtx string) string {
	if strings.HasPrefix(deployCtx, "gke_") {
		return gke
	}
	// TODO (tejaldesai): Add minikube detection.
	return others
}
