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

package deploy

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
)

// Deployer is the Deploy API of skaffold and responsible for deploying
// the build results to a Kubernetes cluster
type Deployer interface {
	// Deploy should ensure that the build results are deployed to the Kubernetes
	// cluster. Returns the list of impacted namespaces.
	Deploy(context.Context, io.Writer, []build.Artifact) ([]string, error)

	// Dependencies returns a list of files that the deployer depends on.
	// In dev mode, a redeploy will be triggered
	Dependencies() ([]string, error)

	// Cleanup deletes what was deployed by calling Deploy.
	Cleanup(context.Context, io.Writer) error

	// Render generates the Kubernetes manifests replacing the build results and
	// writes them to the given file path
	Render(context.Context, io.Writer, []build.Artifact, bool, string) error
}
