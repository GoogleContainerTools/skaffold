// Copyright 2022 Google LLC All Rights Reserved.
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

// Package github provides a keychain for the GitHub Container Registry.
package github

import (
	"net/url"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
)

const ghcrHostname = "ghcr.io"

// Keychain exports an instance of the GitHub Keychain.
//
// This keychain matches on requests for ghcr.io and provides the value of the
// environment variable $GITHUB_TOKEN, if it's set.
var Keychain authn.Keychain = githubKeychain{}

type githubKeychain struct{}

func (githubKeychain) Resolve(r authn.Resource) (authn.Authenticator, error) {
	serverURL, err := url.Parse("https://" + r.String())
	if err != nil {
		return authn.Anonymous, nil
	}
	if serverURL.Hostname() == ghcrHostname {
		username := os.Getenv("GITHUB_ACTOR")
		if username == "" {
			username = "unset"
		}
		if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
			return githubAuthenticator{username, tok}, nil
		}
	}
	return authn.Anonymous, nil
}

type githubAuthenticator struct{ username, password string }

func (g githubAuthenticator) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{
		Username: g.username,
		Password: g.password,
	}, nil
}
