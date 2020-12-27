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

package types

import (
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type Config interface {
	docker.Config

	GetPipelines() []latest.Pipeline
	GetWorkingDir() string
	GlobalConfig() string
	ConfigurationFile() string
	DefaultRepo() *string
	SkipRender() bool
}

// Artifact contains all information about a completed deployment
type Artifact struct {
	Obj       runtime.Object
	Namespace string
}
