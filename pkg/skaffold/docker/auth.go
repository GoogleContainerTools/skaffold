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
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/homedir"
	"github.com/docker/docker/registry"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/gcp"
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

func loadDockerConfig() (*configfile.ConfigFile, error) {
	cf, err := config.Load(configDir)
	if err != nil {
		return nil, fmt.Errorf("docker config: %w", err)
	}

	gcp.AutoConfigureGCRCredentialHelper(cf)

	return cf, nil
}

func (credsHelper) GetAuthConfig(registry string) (types.AuthConfig, error) {
	cf, err := loadDockerConfig()
	if err != nil {
		return types.AuthConfig{}, err
	}

	auth, err := cf.GetAuthConfig(registry)
	if err != nil {
		return types.AuthConfig{}, err
	}

	return types.AuthConfig(auth), nil
}

func (credsHelper) GetAllAuthConfigs() (map[string]types.AuthConfig, error) {
	cf, err := loadDockerConfig()
	if err != nil {
		return nil, err
	}

	credentials, err := cf.GetAllCredentials()
	if err != nil {
		return nil, err
	}

	authConfigs := make(map[string]types.AuthConfig, len(credentials))
	for k, auth := range credentials {
		authConfigs[k] = types.AuthConfig(auth)
	}

	return authConfigs, nil
}

func (l *localDaemon) encodedRegistryAuth(ctx context.Context, a AuthConfigHelper, image string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("parsing image name for registry: %w", err)
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
		return "", fmt.Errorf("getting auth config: %w", err)
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
