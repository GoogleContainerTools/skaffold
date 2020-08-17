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
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

// KptDeployer deploys workflows with kpt CLI
type KptDeployer struct {
}

func NewKptDeployer(runCtx *runcontext.RunContext, labels map[string]string) *KptDeployer {
	return &KptDeployer{}
}

func (k *KptDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]string, error) {
	return nil, nil
}

func (k *KptDeployer) Dependencies() ([]string, error) {
	return nil, nil
}

func (k *KptDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

func (k *KptDeployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, offline bool, filepath string) error {
	return nil
}
