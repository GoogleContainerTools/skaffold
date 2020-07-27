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

package local

import (
	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, &localConfig{})
}

func fakeLocalDaemonWithExtraEnv(extraEnv []string) docker.LocalDaemon {
	return docker.NewLocalDaemon(&testutil.FakeAPIClient{}, extraEnv, &localConfig{})
}

type localConfig struct {
	Config

	local latest.LocalBuild
}

func (c *localConfig) GlobalConfig() string    { return "" }
func (c *localConfig) MinikubeProfile() string { return "" }
func (c *localConfig) GetKubeContext() string  { return "" }
func (c *localConfig) Prune() bool             { return true }
func (c *localConfig) SuppressLogs() []string  { return nil }
func (c *localConfig) Pipeline() latest.Pipeline {
	var pipeline latest.Pipeline
	pipeline.Build.BuildType.LocalBuild = &c.local
	return pipeline
}
