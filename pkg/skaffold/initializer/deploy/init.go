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

package deploy

import (
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/analyze"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

// Initializer detects a deployment type and is able to extract image names from it
type Initializer interface {
	// DeployConfig generates Deploy Config for skaffold configuration.
	DeployConfig() latest.DeployConfig
}

type emptyDeployInit struct {
}

func (e *emptyDeployInit) DeployConfig() latest.DeployConfig {
	return latest.DeployConfig{}
}

// NewInitializer if any helm charts are provided we use HelmInitializer, otherwise we use empty initializer.
func NewInitializer(h analyze.HelmChartInfo, c config.Config) Initializer {
	switch {
	case c.SkipDeploy:
		return &emptyDeployInit{}
	case len(h.Charts()) > 0:
		return newHelmInitializer(h.Charts())
	default:
		return &emptyDeployInit{}
	}
}
