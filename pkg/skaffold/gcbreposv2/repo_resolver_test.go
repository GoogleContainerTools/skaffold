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
	GitRepo       string
	ReadToken     string
	ErrorGetRepo  bool
	ErrorGetToken bool
}

const (
	gcpProject    = "my-project"
	gcpRegion     = "us-central1"
	gcbConnection = "gcb-repo-connection"
	gcbRepo       = "repo-1"
)

func (c GCBReposClientMock) GetRepository(ctx context.Context, req *cloudbuildpb.GetRepositoryRequest, opts ...gax.CallOption) (*cloudbuildpb.Repository, error) {
	if c.ErrorGetRepo {
		return nil, fmt.Errorf("failed to get repo")
	}

	validRepoIdentifier := fmt.Sprintf("projects/%v/locations/%v/connections/%v/repositories/%v", gcpProject, gcpRegion, gcbConnection, gcbRepo)
	if req.Name != validRepoIdentifier {
		return nil, fmt.Errorf("invalid request, expecting %v, got %v", validRepoIdentifier, req.Name)
	}

	return &cloudbuildpb.Repository{
		RemoteUri: c.GitRepo,
	}, nil
}

func (c GCBReposClientMock) FetchReadToken(ctx context.Context, req *cloudbuildpb.FetchReadTokenRequest, opts ...gax.CallOption) (*cloudbuildpb.FetchReadTokenResponse, error) {
	if c.ErrorGetToken {
		return nil, fmt.Errorf("failed to get token")
	}

	validRepoIdentifier := fmt.Sprintf("projects/%v/locations/%v/connections/%v/repositories/%v", gcpProject, gcpRegion, gcbConnection, gcbRepo)
	if req.Repository != validRepoIdentifier {
		return nil, fmt.Errorf("invalid request, expecting %v, got %v", validRepoIdentifier, req.Repository)
	}

	return &cloudbuildpb.FetchReadTokenResponse{
		Token: c.ReadToken,
	}, nil
}

func (c GCBReposClientMock) Close() error { return nil }

func TestGetRepoInfo(t *testing.T) {
	tests := []struct {
		description      string
		expectedRepoInfo Repo
		gcbMockClient    GCBReposClientMock
		shouldError      bool
		errorMsg         string
	}{
		{
			description: "repo info correct",
			expectedRepoInfo: Repo{
				URI:      "https://github.com/GoogleContainerTools/skaffold",
				CloneURI: "https://oauth2:token123@github.com/GoogleContainerTools/skaffold",
			},
			gcbMockClient: GCBReposClientMock{
				GitRepo:   "https://github.com/GoogleContainerTools/skaffold",
				ReadToken: "token123",
			},
		},
		{
			description:      "failed getting GCB repo info",
			expectedRepoInfo: Repo{},
			shouldError:      true,
			errorMsg:         fmt.Sprintf("failed to get remote URI for repository %v", gcbRepo),
			gcbMockClient: GCBReposClientMock{
				ErrorGetRepo: true,
			},
		},
		{
			description:      "failed getting GCB repo read token",
			expectedRepoInfo: Repo{},
			shouldError:      true,
			errorMsg:         fmt.Sprintf("failed to get repository read access token for repo %v", gcbRepo),
			gcbMockClient: GCBReposClientMock{
				ErrorGetToken: true,
			},
		},
		{
			description:      "failed to build clone repo URI",
			expectedRepoInfo: Repo{},
			shouldError:      true,
			errorMsg:         "failed to clone repo :not-valid: trouble building repo URI with token",
			gcbMockClient: GCBReposClientMock{
				GitRepo:   ":not-valid",
				ReadToken: "token123",
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			ctx := context.Background()
			t.Override(&RepositoryManagerClient, func(ctx context.Context) (cloudBuildRepoClient, error) {
				return test.gcbMockClient, nil
			})

			repoInfo, err := GetRepoInfo(ctx, gcpProject, gcpRegion, gcbConnection, gcbRepo)

			t.CheckError(test.shouldError, err)
			if test.shouldError {
				t.CheckErrorContains(test.errorMsg, err)
			}
			t.CheckDeepEqual(test.expectedRepoInfo, repoInfo)
		})
	}
}
