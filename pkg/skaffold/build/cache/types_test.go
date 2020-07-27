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

package cache

import (
	"context"
	"errors"

	"github.com/docker/docker/client"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func mockHasher(value string) func(context.Context, *latest.Artifact) (string, error) {
	return func(context.Context, *latest.Artifact) (string, error) {
		return value, nil
	}
}

func failingHasher(errMessage string) func(context.Context, *latest.Artifact) (string, error) {
	return func(context.Context, *latest.Artifact) (string, error) {
		return "", errors.New(errMessage)
	}
}

func fakeLocalDaemon(api client.CommonAPIClient) docker.LocalDaemon {
	return docker.NewLocalDaemon(api, nil, &cacheConfig{})
}

type cacheConfig struct {
	Config
	cacheFile string
}

func (c *cacheConfig) Prune() bool          { return false }
func (c *cacheConfig) CacheArtifacts() bool { return true }
func (c *cacheConfig) CacheFile() string    { return c.cacheFile }
func (c *cacheConfig) DevMode() bool        { return false }
