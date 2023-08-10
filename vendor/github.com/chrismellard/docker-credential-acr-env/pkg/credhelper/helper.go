/*
Copyright © 2020 Chris Mellard chris.mellard@icloud.com

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
package credhelper

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/chrismellard/docker-credential-acr-env/pkg/registry"
	"github.com/chrismellard/docker-credential-acr-env/pkg/token"
	"github.com/docker/docker-credential-helpers/credentials"
)

var acrRE = regexp.MustCompile(`.*\.azurecr\.io|.*\.azurecr\.cn|.*\.azurecr\.de|.*\.azurecr\.us`)

const (
	mcrHostname   = "mcr.microsoft.com"
	tokenUsername = "<token>"
)

type ACRCredHelper struct {
}

func NewACRCredentialsHelper() credentials.Helper {
	return &ACRCredHelper{}
}

func (a ACRCredHelper) Add(_ *credentials.Credentials) error {
	return errors.New("list is unimplemented")
}

func (a ACRCredHelper) Delete(_ string) error {
	return errors.New("list is unimplemented")
}

func isACRRegistry(input string) bool {
	serverURL, err := url.Parse("https://" + input)
	if err != nil {
		return false
	}
	if serverURL.Hostname() == mcrHostname {
		return true
	}
	matches := acrRE.FindStringSubmatch(serverURL.Hostname())
	if len(matches) == 0 {
		return false
	}
	return true
}

func (a ACRCredHelper) Get(serverURL string) (string, string, error) {
	if !isACRRegistry(serverURL) {
		return "", "", errors.New("serverURL does not refer to Azure Container Registry")
	}

	spToken, settings, err := token.GetServicePrincipalTokenFromEnvironment()
	if err != nil {
		return "", "", fmt.Errorf("failed to acquire sp token %w", err)
	}
	refreshToken, err := registry.GetRegistryRefreshTokenFromAADExchange(serverURL, spToken, settings.Values[auth.TenantID])
	if err != nil {
		return "", "", fmt.Errorf("failed to acquire refresh token %w", err)
	}
	return tokenUsername, refreshToken, nil
}

func (a ACRCredHelper) List() (map[string]string, error) {
	return nil, errors.New("list is unimplemented")
}
