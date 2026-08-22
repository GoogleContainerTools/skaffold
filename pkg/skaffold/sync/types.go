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

package sync

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/logger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

type syncMap map[string][]string

type Item struct {
	Image    string
	Artifact *latest.Artifact
	Copy     map[string][]string
	Delete   map[string][]string
}

type Syncer interface {
	Sync(context.Context, io.Writer, *Item) error
}

// DeploymentAwareSyncer extends Syncer with the ability to track deployed artifacts.
// By tracking which artifacts each deployer deployed
// the syncer can skip sync operations for images it didn't deploy.
type DeploymentAwareSyncer interface {
	Syncer
	RegisterDeployedArtifacts([]graph.Artifact)
	DeployedArtifacts() []graph.Artifact
}

type PodSyncer struct {
	kubectl    *pkgkubectl.CLI
	namespaces *[]string
	formatter  logger.Formatter
	// deployedArtifacts holds the artifacts deployed by the associated deployer.
	// Used to filter sync operations so this syncer only handles its own images.
	deployedArtifacts []graph.Artifact
}

func NewPodSyncer(cli *pkgkubectl.CLI, namespaces *[]string, formatter logger.Formatter, deployedArtifacts []graph.Artifact) *PodSyncer {
	return &PodSyncer{
		kubectl:           cli,
		namespaces:        namespaces,
		formatter:         formatter,
		deployedArtifacts: deployedArtifacts,
	}
}

type NoopSyncer struct{}

func (s *NoopSyncer) Sync(context.Context, io.Writer, *Item) error {
	return nil
}

func (s *NoopSyncer) RegisterDeployedArtifacts([]graph.Artifact) {}

func (s *NoopSyncer) DeployedArtifacts() []graph.Artifact {
	return nil
}

func (i *Item) HasChanges() bool {
	return i != nil && (len(i.Copy) > 0 || len(i.Delete) > 0)
}
