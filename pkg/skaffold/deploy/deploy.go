/*
Copyright 2018 Google LLC

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
	"fmt"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
)

// Result is currently unused, but a stub for results that might be returned
// from a Deployer.Run()
type Result struct{}

// Deployer is the Deploy API of skaffold and responsible for deploying
// the build results to a Kubernetes cluster
type Deployer interface {
	// Run should ensure that the build results are deployed to the Kubernetes
	// cluster.
	Run(*build.BuildResult) (*Result, error)
}

func JoinTagsToBuildResult(b []build.Build, params map[string]string) (map[string]build.Build, error) {
	imageToBuildResult := map[string]build.Build{}
	for _, build := range b {
		imageToBuildResult[build.ImageName] = build
	}

	paramToBuildResult := map[string]build.Build{}
	for param, imageName := range params {
		build, ok := imageToBuildResult[imageName]
		if !ok {
			return nil, fmt.Errorf("No build present for %s", imageName)
		}
		paramToBuildResult[param] = build
	}
	return paramToBuildResult, nil
}
