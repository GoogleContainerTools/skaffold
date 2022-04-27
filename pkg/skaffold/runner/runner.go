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

package runner

import (
	"context"
	"errors"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/util"
)

// ErrorConfigurationChanged is a special error that's returned when the skaffold configuration was changed.
var ErrorConfigurationChanged = errors.New("configuration changed")

// Runner is responsible for running the skaffold build, test and deploy config.
type Runner interface {
	Apply(context.Context, io.Writer) error
	ApplyDefaultRepo(tag string) (string, error)
	Build(context.Context, io.Writer, []*latest.Artifact) ([]graph.Artifact, error)
	Cleanup(context.Context, io.Writer, bool) error
	Dev(context.Context, io.Writer, []*latest.Artifact) error
	// Deploy and DeployAndLog: Do they need the `graph.Artifact` and could use render output.
	Deploy(context.Context, io.Writer, []graph.Artifact, manifest.ManifestList) error
	DeployAndLog(context.Context, io.Writer, []graph.Artifact, manifest.ManifestList) error
	GeneratePipeline(context.Context, io.Writer, []util.VersionedConfig, []string, string) error
	HasBuilt() bool
	HasDeployed() bool
	Prune(context.Context, io.Writer) error

	// "output" arg is only used in Render v1.
	Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, output string) (manifest.ManifestList, error)
	Test(context.Context, io.Writer, []graph.Artifact) error
}
