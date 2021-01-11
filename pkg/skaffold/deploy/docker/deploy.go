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

package docker

import (
	"context"
	"errors"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	deploy "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/types"
	dockerutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type Deployer struct {
	client dockerutil.LocalDaemon
}

type Config interface {
	deploy.Config
}

func NewDeployer(cfg Config, labels map[string]string, d *latest.DockerDeploy) (*Deployer, error) {
	client, err := dockerutil.NewAPIClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Deployer{
		client: client,
	}, nil
}

func (d *Deployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]string, error) {
	// TODO(nkubala): implement
	return nil, errors.New("not implemented")
}

func (d *Deployer) Dependencies() ([]string, error) {
	// TODO(nkubala): implement
	return nil, errors.New("not implemented")
}

func (d *Deployer) Cleanup(context.Context, io.Writer) error {
	// TODO(nkubala): implement
	return errors.New("not implemented")
}

func (d *Deployer) Render(context.Context, io.Writer, []build.Artifact, bool, string) error {
	// TODO(nkubala): implement
	return errors.New("not implemented")
}
