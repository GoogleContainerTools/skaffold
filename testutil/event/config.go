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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func InitializeState(pipes []latest.Pipeline) {
	cfg := config{
		pipes: pipes,
	}
	event.InitializeState(cfg)
}

type config struct {
	pipes []latest.Pipeline
}

func (c config) AutoBuild() bool                 { return true }
func (c config) AutoDeploy() bool                { return true }
func (c config) AutoSync() bool                  { return true }
func (c config) GetPipelines() []latest.Pipeline { return c.pipes }
func (c config) GetKubeContext() string          { return "temp" }
