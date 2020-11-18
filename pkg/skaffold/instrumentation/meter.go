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
	"runtime"
	"time"

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
	EnumFlags      map[string]interface{}
	Builders       map[string]bool
	SyncType       map[string]bool
	DevIterations  map[string]int
	StartTime      time.Time
	ErrorCode      proto.StatusCode
}

var (
	meter = skaffoldMeter{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Builders:  map[string]bool{},
		SyncType:  map[string]bool{},
		StartTime: time.Now(),
		Version:   version.Get().Version,
		ExitCode:  0,
		ErrorCode: proto.StatusCode_OK,
	}
)

func InitMeter(runCtx *runcontext.RunContext, config *latest.SkaffoldConfig) {
	meter.Command = runCtx.Opts.Command
	meter.PlatformType = yamltags.GetYamlTag(config.Build.BuildType)
	for _, artifact := range config.Pipeline.Build.Artifacts {
		meter.Builders[yamltags.GetYamlTag(artifact.ArtifactType)] = true
		if artifact.Sync != nil {
			meter.SyncType[yamltags.GetYamlTag(artifact.Sync)] = true
		}
	}
	meter.BuildArtifacts = len(config.Pipeline.Build.Artifacts)
}
