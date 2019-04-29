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

package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/gcp"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/homedir"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	configFileDir = ".docker"
)

var (
	// DefaultAuthHelper is exposed so that other packages can override it for testing
	DefaultAuthHelper AuthConfigHelper
	configDir         = os.Getenv("DOCKER_CONFIG")
)

func init() {
	DefaultAuthHelper = credsHelper{}
	if configDir == "" {
		configDir = filepath.Join(homedir.Get(), configFileDir)
	}
}

// AuthConfigHelper exists for testing purposes since GetAuthConfig shells out
// to native store helpers.
// Ideally this shouldn't be public, but the LocalBuilder needs to use it.
type AuthConfigHelper interface {
	GetAuthConfig(registry string) (types.AuthConfig, error)
	GetAllAuthConfigs() (map[string]types.AuthConfig, error)
}

type credsHelper struct{}

func (credsHelper) GetAuthConfig(registry string) (types.AuthConfig, error) {
	cf, err := config.Load(configDir)
	if err != nil {
		return types.AuthConfig{}, errors.Wrap(err, "docker config")
	}

	gcp.AutoConfigureGCRCredentialHelper(cf, registry)

	return cf.GetAuthConfig(registry)
}

func (credsHelper) GetAllAuthConfigs() (map[string]types.AuthConfig, error) {
	cf, err := config.Load(configDir)
	if err != nil {
		return nil, errors.Wrap(err, "docker config")
	}

	// TODO(dgageot): this is really slow because it has to run all the credential helpers.
	// return cf.GetAllCredentials()
	return cf.GetCredentialsStore("").GetAll()
}

func (l *localDaemon) encodedRegistryAuth(ctx context.Context, a AuthConfigHelper, image string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", errors.Wrap(err, "parsing image name for registry")
	}

	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return "", err
	}

	configKey := repoInfo.Index.Name
	if repoInfo.Index.Official {
		configKey = l.officialRegistry(ctx)
	}

	ac, err := a.GetAuthConfig(configKey)
	if err != nil {
		return "", errors.Wrap(err, "getting auth config")
	}

	buf, err := json.Marshal(ac)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}

func (l *localDaemon) officialRegistry(ctx context.Context) string {
	serverAddress := registry.IndexServer

	// The daemon `/info` endpoint informs us of the default registry being used.
	info, err := l.apiClient.Info(ctx)
	switch {
	case err != nil:
		logrus.Warnf("failed to get default registry endpoint from daemon (%v). Using system default: %s\n", err, serverAddress)
	case info.IndexServerAddress == "":
		logrus.Warnf("empty registry endpoint from daemon. Using system default: %s\n", serverAddress)
	default:
		serverAddress = info.IndexServerAddress
	}

	return serverAddress
}
