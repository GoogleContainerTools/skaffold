// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/sirupsen/logrus"

	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login/cache"
)

const (
	proxyEndpointScheme = "https://"
	programName         = "docker-credential-ecr-login"
	ecrPublicName       = "public.ecr.aws"
	ecrPublicEndpoint   = proxyEndpointScheme + ecrPublicName
)

var ecrPattern = regexp.MustCompile(`^(\d{12})\.dkr[\.\-]ecr(\-fips)?\.([a-zA-Z0-9][a-zA-Z0-9-_]*)\.(amazonaws\.(?:com(?:\.cn)?|eu)|on\.(?:aws|amazonwebservices\.com\.cn)|sc2s\.sgov\.gov|c2s\.ic\.gov|cloud\.adc-e\.uk|csp\.hci\.ic\.gov)$`)

type Service string

const (
	ServiceECR       Service = "ecr"
	ServiceECRPublic Service = "ecr-public"
)

// Registry in ECR
type Registry struct {
	Service Service
	ID      string
	FIPS    bool
	Region  string
}

// ExtractRegistry returns the ECR registry behind a given service endpoint
func ExtractRegistry(input string) (*Registry, error) {
	if strings.HasPrefix(input, proxyEndpointScheme) {
		input = strings.TrimPrefix(input, proxyEndpointScheme)
	}
	serverURL, err := url.Parse(proxyEndpointScheme + input)
	if err != nil {
		return nil, err
	}
	if serverURL.Hostname() == ecrPublicName {
		return &Registry{
			Service: ServiceECRPublic,
		}, nil
	}
	matches := ecrPattern.FindStringSubmatch(serverURL.Hostname())
	if len(matches) == 0 {
		return nil, fmt.Errorf(programName + " can only be used with Amazon Elastic Container Registry.")
	} else if len(matches) < 3 {
		return nil, fmt.Errorf("%q is not a valid repository URI for Amazon Elastic Container Registry.", input)
	}
	return &Registry{
		Service: ServiceECR,
		ID:      matches[1],
		FIPS:    matches[2] == "-fips",
		Region:  matches[3],
	}, nil
}

// Client used for calling ECR service
type Client interface {
	GetCredentials(serverURL string) (*Auth, error)
	GetCredentialsByRegistryID(registryID string) (*Auth, error)
	ListCredentials() ([]*Auth, error)
}

// Auth credentials returned by ECR service to allow docker login
type Auth struct {
	ProxyEndpoint string
	Username      string
	Password      string
}

type defaultClient struct {
	ecrClient       ECRAPI
	ecrPublicClient ECRPublicAPI
	credentialCache cache.CredentialsCache
}

type ECRAPI interface {
	GetAuthorizationToken(context.Context, *ecr.GetAuthorizationTokenInput, ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error)
}

type ECRPublicAPI interface {
	GetAuthorizationToken(context.Context, *ecrpublic.GetAuthorizationTokenInput, ...func(*ecrpublic.Options)) (*ecrpublic.GetAuthorizationTokenOutput, error)
}

// GetCredentials returns username, password, and proxyEndpoint
func (c *defaultClient) GetCredentials(serverURL string) (*Auth, error) {
	registry, err := ExtractRegistry(serverURL)
	if err != nil {
		return nil, err
	}
	logrus.
		WithField("service", registry.Service).
		WithField("registry", registry.ID).
		WithField("region", registry.Region).
		WithField("serverURL", serverURL).
		Debug("Retrieving credentials")
	switch registry.Service {
	case ServiceECR:
		return c.GetCredentialsByRegistryID(registry.ID)
	case ServiceECRPublic:
		return c.GetPublicCredentials()
	}
	return nil, fmt.Errorf("unknown service %q", registry.Service)
}

// GetCredentialsByRegistryID returns username, password, and proxyEndpoint
func (c *defaultClient) GetCredentialsByRegistryID(registryID string) (*Auth, error) {
	cachedEntry := c.credentialCache.Get(registryID)
	if cachedEntry != nil {
		if cachedEntry.IsValid(time.Now()) {
			logrus.WithField("registry", registryID).Debug("Using cached token")
			return extractToken(cachedEntry.AuthorizationToken, cachedEntry.ProxyEndpoint)
		}
		logrus.
			WithField("requestedAt", cachedEntry.RequestedAt).
			WithField("expiresAt", cachedEntry.ExpiresAt).
			Debug("Cached token is no longer valid")
	}

	auth, err := c.getAuthorizationToken(registryID)

	// if we have a cached token, fall back to avoid failing the request. This may result an expired token
	// being returned, but if there is a 500 or timeout from the service side, we'd like to attempt to re-use an
	// old token. We invalidate tokens prior to their expiration date to help mitigate this scenario.
	if err != nil && cachedEntry != nil {
		logrus.WithError(err).Info("Got error fetching authorization token. Falling back to cached token.")
		return extractToken(cachedEntry.AuthorizationToken, cachedEntry.ProxyEndpoint)
	}
	return auth, err
}

func (c *defaultClient) GetPublicCredentials() (*Auth, error) {
	cachedEntry := c.credentialCache.GetPublic()
	if cachedEntry != nil {
		if cachedEntry.IsValid(time.Now()) {
			logrus.WithField("registry", ecrPublicName).Debug("Using cached token")
			return extractToken(cachedEntry.AuthorizationToken, cachedEntry.ProxyEndpoint)
		}
		logrus.
			WithField("requestedAt", cachedEntry.RequestedAt).
			WithField("expiresAt", cachedEntry.ExpiresAt).
			Debug("Cached token is no longer valid")
	}

	auth, err := c.getPublicAuthorizationToken()
	// if we have a cached token, fall back to avoid failing the request. This may result an expired token
	// being returned, but if there is a 500 or timeout from the service side, we'd like to attempt to re-use an
	// old token. We invalidate tokens prior to their expiration date to help mitigate this scenario.
	if err != nil && cachedEntry != nil {
		logrus.WithError(err).Info("Got error fetching authorization token. Falling back to cached token.")
		return extractToken(cachedEntry.AuthorizationToken, cachedEntry.ProxyEndpoint)
	}
	return auth, err
}

func (c *defaultClient) ListCredentials() ([]*Auth, error) {
	// prime the cache with default authorization tokens
	_, err := c.GetCredentialsByRegistryID("")
	if err != nil {
		logrus.WithError(err).Debug("couldn't get authorization token for default registry")
	}
	_, err = c.GetPublicCredentials()
	if err != nil {
		logrus.WithError(err).Debug("couldn't get authorization token for public registry")
	}

	auths := make([]*Auth, 0)
	for _, authEntry := range c.credentialCache.List() {
		auth, err := extractToken(authEntry.AuthorizationToken, authEntry.ProxyEndpoint)
		if err != nil {
			logrus.WithError(err).Debug("Could not extract token")
		} else {
			auths = append(auths, auth)
		}
	}

	return auths, nil
}

func (c *defaultClient) getAuthorizationToken(registryID string) (*Auth, error) {
	var input *ecr.GetAuthorizationTokenInput
	if registryID == "" {
		logrus.Debug("Calling ECR.GetAuthorizationToken for default registry")
		input = &ecr.GetAuthorizationTokenInput{}
	} else {
		logrus.WithField("registry", registryID).Debug("Calling ECR.GetAuthorizationToken")
		input = &ecr.GetAuthorizationTokenInput{
			RegistryIds: []string{registryID},
		}
	}

	output, err := c.ecrClient.GetAuthorizationToken(context.TODO(), input)
	if err != nil || output == nil {
		if err == nil {
			if registryID == "" {
				err = fmt.Errorf("missing AuthorizationData in ECR response for default registry")
			} else {
				err = fmt.Errorf("missing AuthorizationData in ECR response for %s", registryID)
			}
		}
		return nil, fmt.Errorf("ecr: Failed to get authorization token: %w", err)
	}

	for _, authData := range output.AuthorizationData {
		if authData.ProxyEndpoint != nil && authData.AuthorizationToken != nil {
			authEntry := cache.AuthEntry{
				AuthorizationToken: aws.ToString(authData.AuthorizationToken),
				RequestedAt:        time.Now(),
				ExpiresAt:          aws.ToTime(authData.ExpiresAt),
				ProxyEndpoint:      aws.ToString(authData.ProxyEndpoint),
				Service:            cache.ServiceECR,
			}
			registry, err := ExtractRegistry(authEntry.ProxyEndpoint)
			if err != nil {
				return nil, fmt.Errorf("Invalid ProxyEndpoint returned by ECR: %s", authEntry.ProxyEndpoint)
			}
			auth, err := extractToken(authEntry.AuthorizationToken, authEntry.ProxyEndpoint)
			if err != nil {
				return nil, err
			}
			c.credentialCache.Set(registry.ID, &authEntry)
			return auth, nil
		}
	}
	if registryID == "" {
		return nil, fmt.Errorf("No AuthorizationToken found for default registry")
	}
	return nil, fmt.Errorf("No AuthorizationToken found for %s", registryID)
}

func (c *defaultClient) getPublicAuthorizationToken() (*Auth, error) {
	var input *ecrpublic.GetAuthorizationTokenInput

	output, err := c.ecrPublicClient.GetAuthorizationToken(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("ecr: failed to get authorization token: %w", err)
	}
	if output == nil || output.AuthorizationData == nil {
		return nil, fmt.Errorf("ecr: missing AuthorizationData in ECR Public response")
	}
	authData := output.AuthorizationData
	token, err := extractToken(aws.ToString(authData.AuthorizationToken), ecrPublicEndpoint)
	if err != nil {
		return nil, err
	}
	authEntry := cache.AuthEntry{
		AuthorizationToken: aws.ToString(authData.AuthorizationToken),
		RequestedAt:        time.Now(),
		ExpiresAt:          aws.ToTime(authData.ExpiresAt),
		ProxyEndpoint:      ecrPublicEndpoint,
		Service:            cache.ServiceECRPublic,
	}
	c.credentialCache.Set(ecrPublicName, &authEntry)
	return token, nil
}

func extractToken(token string, proxyEndpoint string) (*Auth, error) {
	decodedToken, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	parts := strings.SplitN(string(decodedToken), ":", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid token: expected two parts, got %d", len(parts))
	}

	return &Auth{
		Username:      parts[0],
		Password:      parts[1],
		ProxyEndpoint: proxyEndpoint,
	}, nil
}
