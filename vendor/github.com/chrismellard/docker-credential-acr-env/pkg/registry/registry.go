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
package registry

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/runtime/2019-08-15-preview/containerregistry"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
)

// GetRegistryRefreshTokenFromAADExchange retrieves an OAuth2 refresh token for the registry specified by serverURL
func GetRegistryRefreshTokenFromAADExchange(serverURL string, principalToken *adal.ServicePrincipalToken, tenantID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeOut)
	defer cancel()

	// If refreshing fails, don't try again, just fail.
	principalToken.MaxMSIRefreshAttempts = 1

	if err := principalToken.EnsureFreshWithContext(ctx); err != nil {
		return "", fmt.Errorf("error refreshing sp token - %w", err)
	}

	registryName, err := getRegistryURL(serverURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse server URL - %w", err)
	}
	refreshTokenClient := containerregistry.NewRefreshTokensClient(registryName.String())
	authorizer := autorest.NewBearerAuthorizer(principalToken)
	refreshTokenClient.Authorizer = authorizer
	rt, err := refreshTokenClient.GetFromExchange(ctx, "access_token", serverURL, tenantID, "", principalToken.Token().AccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to get refresh token for container registry - %w", err)
	}

	return *rt.RefreshToken, nil
}

// parseRegistryName parses a serverURL and returns the registry name (i.e. minus transport scheme)
func getRegistryURL(serverURL string) (*url.URL, error) {
	sURL, err := url.Parse(secureScheme + serverURL)
	if err != nil {
		return &url.URL{}, fmt.Errorf("failed to parse server URL - %w", err)
	}

	return sURL, nil
}
