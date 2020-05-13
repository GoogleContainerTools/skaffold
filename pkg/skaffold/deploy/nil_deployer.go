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
)

// NilDeployer is an empty deployer which does not deploy to the cluster
type NilDeployer struct {
}

// Labels returns empty set of labels since deploy is skipped.
func (n *NilDeployer) Labels() map[string]string {
	return map[string]string{}
}

// Deploy returns an empty result with no error
func (n *NilDeployer) Deploy(_ context.Context, _ io.Writer, _ []build.Artifact, _ []Labeller) *Result {
	return &Result{}
}

// Dependencies returns an empty list of files
func (n *NilDeployer) Dependencies() ([]string, error) {
	return []string{}, nil
}

// Cleanup perform no cleanup since there is no deployment
func (n *NilDeployer) Cleanup(context.Context, io.Writer) error {
	return nil
}

// Render renders nothing.
func (n *NilDeployer) Render(context.Context, io.Writer, []build.Artifact, []Labeller, string) error {
	return nil
}
