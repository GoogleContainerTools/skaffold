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
	"strings"
	"sync"

	"github.com/docker/cli/cli/config"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/google"
)

var primaryKeychain = &Keychain{
	configDir: configDir,
}

// Keychain stores an authenticator per registry.
type Keychain struct {
	configDir  string
	byRegistry map[string]*lockedAuthenticator
	lock       sync.Mutex
}

// Resolve retrieves the authenticator for a given resource.
func (a *Keychain) Resolve(res authn.Resource) (authn.Authenticator, error) {
	registry := res.RegistryStr()

	a.lock.Lock()
	defer a.lock.Unlock()

	// Get existing authenticator
	if auth, present := a.byRegistry[registry]; present {
		return auth, nil
	}

	// Create a new authenticator
	auth := &lockedAuthenticator{
		delegate: a.newAuthenticator(res),
	}

	if a.byRegistry == nil {
		a.byRegistry = map[string]*lockedAuthenticator{}
	}
	a.byRegistry[registry] = auth

	return auth, nil
}

// lockedAuthenticator is an authn.Authenticator that can
// be used safely from multiple go routines.
type lockedAuthenticator struct {
	delegate authn.Authenticator
	lock     sync.Mutex
}

func (a *lockedAuthenticator) Authorization() (*authn.AuthConfig, error) {
	a.lock.Lock()
	authorization, err := a.delegate.Authorization()
	a.lock.Unlock()
	return authorization, err
}

// Create a new authenticator for a given reference
// 1. If `gcloud` is configured, we use google.NewGcloudAuthenticator(). It is more efficient because it reuses tokens.
// 2. If something else is configured, we use that authenticator
// 3. If nothing is configured, we check if `gcloud` can be used
// 4. Default to anonymous
func (a *Keychain) newAuthenticator(res authn.Resource) authn.Authenticator {
	registry := res.RegistryStr()

	// 1. Use google.NewGcloudAuthenticator() authenticator if `gcloud` is configured
	cfg, err := config.Load(a.configDir)
	if err == nil && cfg.CredentialHelpers[registry] == "gcloud" {
		if auth, err := google.NewGcloudAuthenticator(); err == nil {
			return auth
		}
	}

	// 2. Use whatever `non anonymous` credential helper is configured
	auth, err := authn.DefaultKeychain.Resolve(res)
	if err == nil && auth != authn.Anonymous {
		return auth
	}

	// 3. Try gcloud for *.gcr.io
	if registry == "gcr.io" || strings.HasSuffix(registry, ".gcr.io") {
		if auth, err := google.NewGcloudAuthenticator(); err == nil {
			return auth
		}
	}

	// 4. Default to anonymous
	return authn.Anonymous
}
