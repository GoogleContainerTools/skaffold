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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

var (
	meter = skaffoldMeter{
		OS:                runtime.GOOS,
		Arch:              runtime.GOARCH,
		EnumFlags:         map[string]string{},
		Builders:          map[string]int{},
		BuildDependencies: map[string]int{},
		SyncType:          map[string]bool{},
		DevIterations:     []devIteration{},
		StartTime:         time.Now(),
		Version:           version.Get().Version,
		ExitCode:          0,
		ErrorCode:         proto.StatusCode_OK,
	}
	MeteredCommands     = util.NewStringSet()
	doesBuild           = util.NewStringSet()
	doesDeploy          = util.NewStringSet()
	initExporter        = initCloudMonitoringExporterMetrics
	isOnline            bool
	shouldExportMetrics bool
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
		if shouldExportMetrics {
			r, err := http.Get("http://clients3.google.com/generate_204")
			if err == nil {
				r.Body.Close()
				isOnline = true
			}
		}
	}()
}

func InitMeterFromConfig(configs []*latest.SkaffoldConfig) {
	var platforms []string
	for _, config := range configs {
		pl := yamltags.GetYamlTag(config.Build.BuildType)
		if !util.StrSliceContains(platforms, pl) {
			platforms = append(platforms, pl)
		}
		for _, artifact := range config.Pipeline.Build.Artifacts {
			meter.Builders[yamltags.GetYamlTag(artifact.ArtifactType)]++
			if len(artifact.Dependencies) > 0 {
				meter.BuildDependencies[yamltags.GetYamlTag(artifact.ArtifactType)]++
			}
			if artifact.Sync != nil {
				meter.SyncType[yamltags.GetYamlTag(artifact.Sync)] = true
			}
		}
		meter.Deployers = append(meter.Deployers, yamltags.GetYamlKeys(config.Deploy.DeployType)...)
		if h := config.Deploy.HelmDeploy; h != nil {
			meter.HelmReleasesCount = len(h.Releases)
		}
		meter.BuildArtifacts += len(config.Pipeline.Build.Artifacts)
	}
	meter.PlatformType = strings.Join(platforms, ":")
	meter.ConfigCount = len(configs)
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

func AddFlag(flag *flag.Flag) {
	if flag.Changed {
		meter.EnumFlags[flag.Name] = flag.Value.String()
	}
}
