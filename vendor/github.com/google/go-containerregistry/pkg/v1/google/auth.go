// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
	"golang.org/x/oauth2"
	googauth "golang.org/x/oauth2/google"
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// GetGcloudCmd is exposed so we can test this.
var GetGcloudCmd = func() *exec.Cmd {
	// This is odd, but basically what docker-credential-gcr does.
	//
	// config-helper is undocumented, but it's purportedly the only supported way
	// of accessing tokens (`gcloud auth print-access-token` is discouraged).
	//
	// --force-auth-refresh means we are getting a token that is valid for about
	// an hour (we reuse it until it's expired).
	return exec.Command("gcloud", "config", "config-helper", "--force-auth-refresh", "--format=json(credential)")
}

// NewEnvAuthenticator returns an authn.Authenticator that generates access
// tokens from the environment we're running in.
//
// See: https://godoc.org/golang.org/x/oauth2/google#FindDefaultCredentials
func NewEnvAuthenticator() (authn.Authenticator, error) {
	ts, err := googauth.DefaultTokenSource(context.Background(), cloudPlatformScope)
	if err != nil {
		return nil, err
	}

	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	return &tokenSourceAuth{oauth2.ReuseTokenSource(token, ts)}, nil
}

// NewGcloudAuthenticator returns an oauth2.TokenSource that generates access
// tokens by shelling out to the gcloud sdk.
func NewGcloudAuthenticator() (authn.Authenticator, error) {
	if _, err := exec.LookPath("gcloud"); err != nil {
		// gcloud is not available, fall back to anonymous
		logs.Warn.Println("gcloud binary not found")
		return authn.Anonymous, nil
	}

	ts := gcloudSource{GetGcloudCmd}

	// Attempt to fetch a token to ensure gcloud is installed and we can run it.
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	return &tokenSourceAuth{oauth2.ReuseTokenSource(token, ts)}, nil
}

// NewJSONKeyAuthenticator returns a Basic authenticator which uses Service Account
// as a way of authenticating with Google Container Registry.
// More information: https://cloud.google.com/container-registry/docs/advanced-authentication#json_key_file
func NewJSONKeyAuthenticator(serviceAccountJSON string) authn.Authenticator {
	return &authn.Basic{
		Username: "_json_key",
		Password: serviceAccountJSON,
	}
}

// NewTokenAuthenticator returns an oauth2.TokenSource that generates access
// tokens by using the Google SDK to produce JWT tokens from a Service Account.
// More information: https://godoc.org/golang.org/x/oauth2/google#JWTAccessTokenSourceFromJSON
func NewTokenAuthenticator(serviceAccountJSON string, scope string) (authn.Authenticator, error) {
	ts, err := googauth.JWTAccessTokenSourceFromJSON([]byte(serviceAccountJSON), scope)
	if err != nil {
		return nil, err
	}

	return &tokenSourceAuth{oauth2.ReuseTokenSource(nil, ts)}, nil
}

// NewTokenSourceAuthenticator converts an oauth2.TokenSource into an authn.Authenticator.
func NewTokenSourceAuthenticator(ts oauth2.TokenSource) authn.Authenticator {
	return &tokenSourceAuth{ts}
}

// tokenSourceAuth turns an oauth2.TokenSource into an authn.Authenticator.
type tokenSourceAuth struct {
	oauth2.TokenSource
}

// Authorization implements authn.Authenticator.
func (tsa *tokenSourceAuth) Authorization() (*authn.AuthConfig, error) {
	token, err := tsa.Token()
	if err != nil {
		return nil, err
	}

	return &authn.AuthConfig{
		Username: "_token",
		Password: token.AccessToken,
	}, nil
}

// gcloudOutput represents the output of the gcloud command we invoke.
//
// `gcloud config config-helper --format=json(credential)` looks something like:
//
// {
//   "credential": {
//     "access_token": "ya29.abunchofnonsense",
//     "token_expiry": "2018-12-02T04:08:13Z"
//   }
// }
type gcloudOutput struct {
	Credential struct {
		AccessToken string `json:"access_token"`
		TokenExpiry string `json:"token_expiry"`
	} `json:"credential"`
}

type gcloudSource struct {
	// This is passed in so that we mock out gcloud and test Token.
	exec func() *exec.Cmd
}

// Token implements oauath2.TokenSource.
func (gs gcloudSource) Token() (*oauth2.Token, error) {
	cmd := gs.exec()
	var out bytes.Buffer
	cmd.Stdout = &out

	// Don't attempt to interpret stderr, just pass it through.
	cmd.Stderr = logs.Warn.Writer()

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error executing `gcloud config config-helper`: %w", err)
	}

	creds := gcloudOutput{}
	if err := json.Unmarshal(out.Bytes(), &creds); err != nil {
		return nil, fmt.Errorf("failed to parse `gcloud config config-helper` output: %w", err)
	}

	expiry, err := time.Parse(time.RFC3339, creds.Credential.TokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gcloud token expiry: %w", err)
	}

	token := oauth2.Token{
		AccessToken: creds.Credential.AccessToken,
		Expiry:      expiry,
	}

	return &token, nil
}
