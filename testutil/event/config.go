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

package event

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
)

func InitializeState(pipes []latest_v1.Pipeline) {
	cfg := config{
		pipes: pipes,
	}
	event.InitializeState(cfg)
	eventV2.InitializeState(cfg)
}

type config struct {
	pipes []latest_v1.Pipeline
}

func (c config) AutoBuild() bool                    { return true }
func (c config) AutoDeploy() bool                   { return true }
func (c config) AutoSync() bool                     { return true }
func (c config) GetPipelines() []latest_v1.Pipeline { return c.pipes }
func (c config) GetKubeContext() string             { return "temp" }
