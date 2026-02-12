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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	clitypes "github.com/docker/cli/cli/config/types"
	types "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/homedir"
	"github.com/docker/docker/registry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/google"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/gcp"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
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
	GetAuthConfig(ctx context.Context, registry string) (types.AuthConfig, error)
	GetAllAuthConfigs(ctx context.Context) (map[string]types.AuthConfig, error)
}

type credsHelper struct{}

func LoadDockerConfig(access bool) (*configfile.ConfigFile, error) {
	cf, err := config.Load(configDir)
	if err == nil && access {
		// proper check of config file to detect permissions errors
		if _, err = os.Stat(cf.Filename); err == nil {
			var file *os.File
			file, err = os.Open(cf.Filename)
			if err == nil {
				defer file.Close()
			}
		}
		if err != nil {
			log.Entry(context.TODO()).Warnf("cannot access docker config file %s: %v", configDir, err)
		}
	}
	return cf, err
}

func loadDockerConfig() (*configfile.ConfigFile, error) {
	cf, err := LoadDockerConfig(false)
	if err != nil {
		return nil, fmt.Errorf("docker config: %w", err)
	}
	gcp.AutoConfigureGCRCredentialHelper(cf)
	return cf, nil
}

func (h credsHelper) GetAuthConfig(ctx context.Context, registry string) (types.AuthConfig, error) {
	cf, err := loadDockerConfig()
	if err != nil {
		return types.AuthConfig{}, err
	}

	return h.loadCredentials(ctx, cf, registry)
}

func (h credsHelper) loadCredentials(ctx context.Context, cf *configfile.ConfigFile, registry string) (types.AuthConfig, error) {
	if helper := cf.CredentialHelpers[registry]; helper == "gcloud" {
		authCfg, err := h.getGoogleAuthConfig(ctx, registry)
		if err == nil {
			return authCfg, nil
		}
		log.Entry(context.TODO()).Debugf("error getting google authenticator, falling back to docker auth: %v", err)
	}

	var anonymous clitypes.AuthConfig
	auth, err := cf.GetAuthConfig(registry)
	if err != nil {
		return types.AuthConfig{}, err
	}

	// From go-containerrergistry logic, the ServerAddress is not considered when determining if returned auth is anonymous.
	anonymous.ServerAddress = auth.ServerAddress
	if auth != anonymous {
		return types.AuthConfig(auth), nil
	}

	if isGoogleRegistry(registry) {
		authCfg, err := h.getGoogleAuthConfig(ctx, registry)
		if err == nil {
			return authCfg, nil
		}
	}

	return types.AuthConfig(auth), nil
}

func (h credsHelper) getGoogleAuthConfig(ctx context.Context, registry string) (types.AuthConfig, error) {
	auth, err := google.NewEnvAuthenticator(ctx)
	if err != nil {
		return types.AuthConfig{}, err
	}

	if auth == authn.Anonymous {
		return types.AuthConfig{}, fmt.Errorf("error getting google authenticator")
	}

	cfg, err := auth.Authorization()
	if err != nil {
		return types.AuthConfig{}, err
	}

	bCfg, err := cfg.MarshalJSON()
	if err != nil {
		return types.AuthConfig{}, err
	}

	var authCfg types.AuthConfig
	err = json.Unmarshal(bCfg, &authCfg)
	if err != nil {
		return types.AuthConfig{}, err
	}

	// The docker library does the same when we request the credentials
	authCfg.ServerAddress = registry

	return authCfg, nil
}

// GetAllAuthConfigs retrieves all the auth configs.
// Because this can take a long time, we make sure it can be interrupted by the user.
func (h credsHelper) GetAllAuthConfigs(ctx context.Context) (map[string]types.AuthConfig, error) {
	type result struct {
		configs map[string]types.AuthConfig
		err     error
	}

	auth := make(chan result)

	go func() {
		configs, err := h.doGetAllAuthConfigs(ctx)
		auth <- result{configs, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-auth:
		return r.configs, r.err
	}
}

func (h credsHelper) doGetAllAuthConfigs(ctx context.Context) (map[string]types.AuthConfig, error) {
	credentials := make(map[string]types.AuthConfig)
	cf, err := loadDockerConfig()
	if err != nil {
		return nil, err
	}

	defaultCreds, err := cf.GetCredentialsStore("").GetAll()
	if err != nil {
		return nil, err
	}

	for registry, cred := range defaultCreds {
		credentials[registry] = types.AuthConfig(cred)
	}

	for registry := range cf.CredentialHelpers {
		authCfg, err := h.loadCredentials(ctx, cf, registry)
		if err != nil {
			log.Entry(context.TODO()).Debugf("failed to get credentials for registry %v: %v", registry, err)
			continue
		}
		credentials[registry] = authCfg
	}

	return credentials, nil
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

	ac, err := a.GetAuthConfig(ctx, configKey)
	if err != nil {
		return "", fmt.Errorf("getting auth config: %w", err)
	}

	return types.EncodeAuthConfig(ac)
}

func (l *localDaemon) officialRegistry(ctx context.Context) string {
	serverAddress := registry.IndexServer

	// The daemon `/info` endpoint informs us of the default registry being used.
	info, err := l.apiClient.Info(ctx)
	switch {
	case err != nil:
		log.Entry(ctx).Warnf("failed to get default registry endpoint from daemon (%v). Using system default: %s\n", err, serverAddress)
	case info.IndexServerAddress == "":
		log.Entry(ctx).Warnf("empty registry endpoint from daemon. Using system default: %s\n", serverAddress)
	default:
		serverAddress = info.IndexServerAddress
	}

	return serverAddress
}
