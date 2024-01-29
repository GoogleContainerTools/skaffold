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
	"testing"

	"cloud.google.com/go/cloudbuild/apiv2/cloudbuildpb"
	"github.com/googleapis/gax-go/v2"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type GCBReposClientMock struct {
	Region              string
	Project             string
	Connection          string
	Repo                string
	GitRepo             string
	ReadToken           string
	ShouldErrorGetRepo  bool
	ShouldErrorGetToken bool
}

func (c GCBReposClientMock) GetRepository(ctx context.Context, req *cloudbuildpb.GetRepositoryRequest, opts ...gax.CallOption) (*cloudbuildpb.Repository, error) {
	if c.ShouldErrorGetRepo {
		return nil, fmt.Errorf("failed to get repo")
	}

	if req.Name == fmt.Sprintf("projects/%v/locations/%v/connections/%v/repositories/%v", c.Project, c.Region, c.Connection, c.Repo) {
		return &cloudbuildpb.Repository{
			Name:      c.Repo,
			RemoteUri: c.GitRepo,
		}, nil
	}

	return nil, fmt.Errorf("invalid request")
}

func (c GCBReposClientMock) FetchReadToken(ctx context.Context, req *cloudbuildpb.FetchReadTokenRequest, opts ...gax.CallOption) (*cloudbuildpb.FetchReadTokenResponse, error) {
	if c.ShouldErrorGetToken {
		return nil, fmt.Errorf("failed to get token")
	}

	if req.Repository == fmt.Sprintf("projects/%v/locations/%v/connections/%v/repositories/%v", c.Project, c.Region, c.Connection, c.Repo) {
		return &cloudbuildpb.FetchReadTokenResponse{
			Token: c.ReadToken,
		}, nil
	}
	return nil, fmt.Errorf("invalid request")
}

func (c GCBReposClientMock) Close() error { return nil }

func TestGetRepoURI(t *testing.T) {
	tests := []struct {
		description         string
		gcpRegion           string
		gcpProject          string
		gcpConnection       string
		gcpRepo             string
		expectedGitRepo     string
		expectedToken       string
		readToken           string
		shouldErrorGetRepo  bool
		shouldErrorGetToken bool
		errorMsg            string
	}{
		{
			description:     "correct repo connection string",
			gcpRegion:       "us-central1",
			gcpProject:      "my-project",
			gcpConnection:   "gcb-repo-connection",
			gcpRepo:         "repo-1",
			expectedGitRepo: "https://github.com/org/repo-1",
		},
		{
			description:        "failed getting GCB repo info",
			gcpRepo:            "repo-1",
			shouldErrorGetRepo: true,
			errorMsg:           "failed to get remote uri for repository repo-1",
		},
		{
			description:         "failed getting GCB repo read token",
			gcpRepo:             "repo-1",
			shouldErrorGetToken: true,
			errorMsg:            "failed to get repository read access token for repo repo-1",
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ctx := context.Background()
			t.Override(&RepositoryManagerClient, func(ctx context.Context) (cloudBuildRepoClient, error) {
				return GCBReposClientMock{
					Region:              test.gcpRegion,
					Project:             test.gcpProject,
					Connection:          test.gcpConnection,
					Repo:                test.gcpRepo,
					GitRepo:             test.expectedGitRepo,
					ReadToken:           test.expectedToken,
					ShouldErrorGetRepo:  test.shouldErrorGetRepo,
					ShouldErrorGetToken: test.shouldErrorGetToken,
				}, nil
			})

			uri, accessToken, err := GetRepoInfo(ctx, test.gcpProject, test.gcpRegion, test.gcpConnection, test.gcpRepo)

			shouldError := test.shouldErrorGetRepo || test.shouldErrorGetToken
			if shouldError {
				t.CheckError(shouldError, err)
				t.CheckErrorContains(test.errorMsg, err)
			}

			t.CheckDeepEqual(test.expectedGitRepo, uri)
			t.CheckDeepEqual(test.expectedToken, accessToken)
		})
	}
}
