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
	"context"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
)

// Keychain exports an instance of the google Keychain.
var Keychain authn.Keychain = &googleKeychain{}

type googleKeychain struct {
	once sync.Once
	auth authn.Authenticator
}

// Resolve implements authn.Keychain a la docker-credential-gcr.
//
// This behaves similarly to the GCR credential helper, but reuses tokens until
// they expire.
//
// We can't easily add this behavior to our credential helper implementation
// of authn.Authenticator because the credential helper protocol doesn't include
// expiration information, see here:
// https://godoc.org/github.com/docker/docker-credential-helpers/credentials#Credentials
//
// In addition to being a performance optimization, the reuse of these access
// tokens works around a bug in gcloud. It appears that attempting to invoke
// `gcloud config config-helper` multiple times too quickly will fail:
// https://github.com/GoogleCloudPlatform/docker-credential-gcr/issues/54
//
// We could upstream this behavior into docker-credential-gcr by parsing
// gcloud's output and persisting its tokens across invocations, but then
// we have to deal with invalidating caches across multiple runs (no fun).
//
// In general, we don't worry about that here because we expect to use the same
// gcloud configuration in the scope of this one process.
func (gk *googleKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	return gk.ResolveContext(context.Background(), target)
}

// ResolveContext implements authn.ContextKeychain.
func (gk *googleKeychain) ResolveContext(ctx context.Context, target authn.Resource) (authn.Authenticator, error) {
	// Only authenticate GCR and AR so it works with authn.NewMultiKeychain to fallback.
	if !isGoogle(target.RegistryStr()) {
		return authn.Anonymous, nil
	}

	gk.once.Do(func() {
		gk.auth = resolve(ctx)
	})

	return gk.auth, nil
}

func resolve(ctx context.Context) authn.Authenticator {
	auth, envErr := NewEnvAuthenticator(ctx)
	if envErr == nil && auth != authn.Anonymous {
		logs.Debug.Println("google.Keychain: using Application Default Credentials")
		return auth
	}

	auth, gErr := NewGcloudAuthenticator(ctx)
	if gErr == nil && auth != authn.Anonymous {
		logs.Debug.Println("google.Keychain: using gcloud fallback")
		return auth
	}

	logs.Debug.Println("Failed to get any Google credentials, falling back to Anonymous")
	if envErr != nil {
		logs.Debug.Printf("Google env error: %v", envErr)
	}
	if gErr != nil {
		logs.Debug.Printf("gcloud error: %v", gErr)
	}
	return authn.Anonymous
}

func isGoogle(host string) bool {
	return host == "gcr.io" ||
		strings.HasSuffix(host, ".gcr.io") ||
		strings.HasSuffix(host, ".pkg.dev") ||
		strings.HasSuffix(host, ".google.com")
}
