/*
Copyright 2021 The Skaffold Authors

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

package hooks

import (
	"context"
	"io"
)

type Runner interface {
	RunPreHooks(ctx context.Context, out io.Writer) error
	RunPostHooks(ctx context.Context, out io.Writer) error
}

type phase string

var phases = struct {
	PreBuild   phase
	PostBuild  phase
	PreSync    phase
	PostSync   phase
	PreDeploy  phase
	PostDeploy phase
}{
	PreBuild:   "pre-build",
	PostBuild:  "post-build",
	PreSync:    "pre-sync",
	PostSync:   "post-sync",
	PreDeploy:  "pre-deploy",
	PostDeploy: "post-deploy",
}
