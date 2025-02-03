/*
Copyright 2024 The Skaffold Authors

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

package gcbreposv2

import (
	"context"
	"fmt"
	"net/url"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv2"
	cloudbuildpb "cloud.google.com/go/cloudbuild/apiv2/cloudbuildpb"
	"github.com/googleapis/gax-go/v2"
)

type cloudBuildRepoClient interface {
	GetRepository(ctx context.Context, req *cloudbuildpb.GetRepositoryRequest, opts ...gax.CallOption) (*cloudbuildpb.Repository, error)
	FetchReadToken(ctx context.Context, req *cloudbuildpb.FetchReadTokenRequest, opts ...gax.CallOption) (*cloudbuildpb.FetchReadTokenResponse, error)
	Close() error
}

type Repo struct {
	// Original repo URI.
	URI string

	// URI with oauth2 format.
	CloneURI string
}

var RepositoryManagerClient = repositoryManagerClient

func GetRepoInfo(ctx context.Context, gcpProject, gcpRegion, gcpConnectionName, gcpRepoName string) (Repo, error) {
	cbRepoRef := fmt.Sprintf("projects/%v/locations/%v/connections/%v/repositories/%v", gcpProject, gcpRegion, gcpConnectionName, gcpRepoName)
	cbClient, err := RepositoryManagerClient(ctx)
	if err != nil {
		return Repo{}, fmt.Errorf("failed to create repository manager client: %w", err)
	}
	defer cbClient.Close()

	repoURI, err := getRepoURI(ctx, cbClient, cbRepoRef)
	if err != nil {
		return Repo{}, fmt.Errorf("failed to get remote URI for repository %v: %w", gcpRepoName, err)
	}

	readAccessToken, err := getRepoReadAccessToken(ctx, cbClient, cbRepoRef)
	if err != nil {
		return Repo{}, fmt.Errorf("failed to get repository read access token for repo %v: %w", gcpRepoName, err)
	}

	repoCloneURI, err := buildRepoURIWithToken(repoURI, readAccessToken)
	if err != nil {
		return Repo{}, fmt.Errorf("failed to clone repo %s: trouble building repo URI with token: %w", repoURI, err)
	}

	return Repo{
		URI:      repoURI,
		CloneURI: repoCloneURI,
	}, nil
}

func repositoryManagerClient(ctx context.Context) (cloudBuildRepoClient, error) {
	return cloudbuild.NewRepositoryManagerClient(ctx)
}

func getRepoURI(ctx context.Context, cbClient cloudBuildRepoClient, cbRepoRef string) (string, error) {
	req := &cloudbuildpb.GetRepositoryRequest{
		Name: cbRepoRef,
	}
	repoInfo, err := cbClient.GetRepository(ctx, req)
	if err != nil {
		return "", err
	}
	return repoInfo.GetRemoteUri(), nil
}

func getRepoReadAccessToken(ctx context.Context, cbClient cloudBuildRepoClient, cbRepoRef string) (string, error) {
	req := &cloudbuildpb.FetchReadTokenRequest{
		Repository: cbRepoRef,
	}
	resp, err := cbClient.FetchReadToken(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.GetToken(), nil
}

func buildRepoURIWithToken(repoURI, readAccessToken string) (string, error) {
	parsed, err := url.Parse(repoURI)
	if err != nil {
		return "", err
	}

	parsed.Host = fmt.Sprintf("oauth2:%v@%v", readAccessToken, parsed.Host)
	return url.PathUnescape(parsed.String())
}
