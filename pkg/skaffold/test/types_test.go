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

package test

import (
	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type testConfig struct {
	Config

	workingDir string
	tests      []*latest.TestCase
}

func (c *testConfig) WorkingDir() string                  { return c.workingDir }
func (c *testConfig) Prune() bool                         { return false }
func (c *testConfig) GetKubeContext() string              { return "" }
func (c *testConfig) MinikubeProfile() string             { return "" }
func (c *testConfig) InsecureRegistries() map[string]bool { return nil }
func (c *testConfig) Pipeline() latest.Pipeline {
	return latest.Pipeline{
		Test: c.tests,
	}
}

func testCase(image string, files ...string) *latest.TestCase {
	return &latest.TestCase{
		ImageName:      image,
		StructureTests: files,
	}
}

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, &testConfig{})
}
