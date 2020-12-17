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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mitchellh/go-homedir"
	flag "github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
	"github.com/GoogleContainerTools/skaffold/proto"
)

type skaffoldMeter struct {
	ExitCode       int
	BuildArtifacts int
	Command        string
	Version        string
	OS             string
	Arch           string
	PlatformType   string
	Deployers      []string
	EnumFlags      map[string]*flag.Flag
	Builders       map[string]bool
	SyncType       map[string]bool
	DevIterations  []devIteration
	StartTime      time.Time
	Duration       time.Duration
	ErrorCode      proto.StatusCode
}

type devIteration struct {
	intent    string
	errorCode proto.StatusCode
}

var (
	meter = skaffoldMeter{
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		EnumFlags:     map[string]*flag.Flag{},
		Builders:      map[string]bool{},
		SyncType:      map[string]bool{},
		DevIterations: []devIteration{},
		StartTime:     time.Now(),
		Version:       version.Get().Version,
		ExitCode:      0,
		ErrorCode:     proto.StatusCode_OK,
	}
	skipExport = os.Getenv("SKAFFOLD_EXPORT_METRICS")
)

func InitMeter(runCtx *runcontext.RunContext, configs []*latest.SkaffoldConfig) {
	meter.Command = runCtx.Opts.Command
	meter.PlatformType = yamltags.GetYamlTag(configs[0].Build.BuildType)
	for _, config := range configs {
		for _, artifact := range config.Pipeline.Build.Artifacts {
			meter.Builders[yamltags.GetYamlTag(artifact.ArtifactType)] = true
			if artifact.Sync != nil {
				meter.SyncType[yamltags.GetYamlTag(artifact.Sync)] = true
			}
		}
		meter.Deployers = append(meter.Deployers, yamltags.GetYamlTags(config.Deploy.DeployType)...)
		meter.BuildArtifacts += len(config.Pipeline.Build.Artifacts)
	}
}

func SetErrorCode(errorCode proto.StatusCode) {
	meter.ErrorCode = errorCode
}

func AddDevIteration(intent string) {
	meter.DevIterations = append(meter.DevIterations, devIteration{intent: intent})
}

func AddDevIterationErr(i int, errorCode proto.StatusCode) {
	if len(meter.DevIterations) == i {
		meter.DevIterations = append(meter.DevIterations, devIteration{intent: "error"})
	}
	meter.DevIterations[i].errorCode = errorCode
}

func AddFlag(flag *flag.Flag) {
	meter.EnumFlags[flag.Name] = flag
}

func ExportMetrics(exitCode int) error {
	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("retrieving home directory: %w", err)
	}
	meter.ExitCode = exitCode
	meter.Duration = time.Since(meter.StartTime)
	return exportMetrics(filepath.Join(home, constants.DefaultSkaffoldDir, constants.DefaultMetricFile), meter)
}

func exportMetrics(filename string, meter skaffoldMeter) error {
	if skipExport != "true" || meter.Command == "" {
		return nil
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var meters []skaffoldMeter
	err = json.Unmarshal(b, &meters)
	if err != nil {
		meters = []skaffoldMeter{}
	}
	meters = append(meters, meter)
	b, _ = json.Marshal(meters)
	return ioutil.WriteFile(filename, b, 0666)
}
