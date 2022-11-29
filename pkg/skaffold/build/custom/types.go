/*
Copyright 2019 The Skaffold Authors

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

package custom

import "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"

// Builder is a builder for custom artifacts
type Builder struct {
	localDocker   docker.LocalDaemon
	cfg           docker.Config
	pushImages    bool
	skipTest      bool
	additionalEnv []string
}

// NewArtifactBuilder returns a new custom artifact builder
func NewArtifactBuilder(localDocker docker.LocalDaemon, cfg docker.Config, pushImages bool, skipTest bool, additionalEnv []string) *Builder {
	return &Builder{
		localDocker:   localDocker,
		cfg:           cfg,
		pushImages:    pushImages,
		skipTest:      skipTest,
		additionalEnv: additionalEnv,
	}
}
