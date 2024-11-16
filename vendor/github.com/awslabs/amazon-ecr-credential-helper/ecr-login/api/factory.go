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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/aws/smithy-go/middleware"
	"github.com/aws/smithy-go/transport/http"
	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login/cache"
	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login/version"
)

// Options makes the constructors more configurable
type Options struct {
	Config   aws.Config
	CacheDir string
}

// ClientFactory is a factory for creating clients to interact with ECR
type ClientFactory interface {
	NewClient(awsConfig aws.Config) Client
	NewClientWithOptions(opts Options) Client
	NewClientFromRegion(region string) Client
	NewClientWithFipsEndpoint(region string) (Client, error)
	NewClientWithDefaults() Client
}

// DefaultClientFactory is a default implementation of the ClientFactory
type DefaultClientFactory struct{}

var userAgentLoadOption = config.WithAPIOptions([]func(*middleware.Stack) error{
	http.AddHeaderValue("User-Agent", "amazon-ecr-credential-helper/"+version.Version),
})

// NewClientWithDefaults creates the client and defaults region
func (defaultClientFactory DefaultClientFactory) NewClientWithDefaults() Client {
	awsConfig, err := config.LoadDefaultConfig(context.TODO(), userAgentLoadOption)
	if err != nil {
		panic(err)
	}

	return defaultClientFactory.NewClientWithOptions(Options{Config: awsConfig})
}

// NewClientWithFipsEndpoint overrides the default ECR service endpoint in a given region to use the FIPS endpoint
func (defaultClientFactory DefaultClientFactory) NewClientWithFipsEndpoint(region string) (Client, error) {
	awsConfig, err := config.LoadDefaultConfig(
		context.TODO(),
		userAgentLoadOption,
		config.WithRegion(region),
		config.WithEndpointDiscovery(aws.EndpointDiscoveryEnabled),
	)
	if err != nil {
		return nil, err
	}

	return defaultClientFactory.NewClientWithOptions(Options{Config: awsConfig}), nil
}

// NewClientFromRegion uses the region to create the client
func (defaultClientFactory DefaultClientFactory) NewClientFromRegion(region string) Client {
	awsConfig, err := config.LoadDefaultConfig(
		context.TODO(),
		userAgentLoadOption,
		config.WithRegion(region),
	)
	if err != nil {
		panic(err)
	}

	return defaultClientFactory.NewClientWithOptions(Options{
		Config: awsConfig,
	})
}

// NewClient Create new client with AWS Config
func (defaultClientFactory DefaultClientFactory) NewClient(awsConfig aws.Config) Client {
	return defaultClientFactory.NewClientWithOptions(Options{Config: awsConfig})
}

// NewClientWithOptions Create new client with Options
func (defaultClientFactory DefaultClientFactory) NewClientWithOptions(opts Options) Client {
	// The ECR Public API is only available in us-east-1 today
	publicConfig := opts.Config.Copy()
	publicConfig.Region = "us-east-1"
	return &defaultClient{
		ecrClient:       ecr.NewFromConfig(opts.Config),
		ecrPublicClient: ecrpublic.NewFromConfig(publicConfig),
		credentialCache: cache.BuildCredentialsCache(opts.Config, opts.CacheDir),
	}
}
