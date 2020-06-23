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

package buildpacks

import (
	"context"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/list"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// GetDependencies returns dependencies listed for a buildpack artifact
func GetDependencies(ctx context.Context, workspace string, a *latest.BuildpackArtifact) ([]string, error) {
	// TODO(dgageot): Support project.toml include/exclude.
	return list.Files(workspace, a.Dependencies.Paths, a.Dependencies.Ignore)
}
