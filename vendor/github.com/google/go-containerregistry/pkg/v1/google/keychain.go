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
	"fmt"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
)

// Keychain exports an instance of the google Keychain.
var Keychain authn.Keychain = &googleKeychain{}

type googleKeychain struct {
	once sync.Once
	auth authn.Authenticator
	err  error
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
	// Only authenticate GCR and AR so it works with authn.NewMultiKeychain to fallback.
	host := target.RegistryStr()
	if host != "gcr.io" && !strings.HasSuffix(host, ".gcr.io") && !strings.HasSuffix(host, ".pkg.dev") {
		return authn.Anonymous, nil
	}

	gk.once.Do(func() {
		gk.auth, gk.err = resolve()
	})

	return gk.auth, gk.err
}

func resolve() (authn.Authenticator, error) {
	auth, envErr := NewEnvAuthenticator()
	if envErr == nil {
		return auth, nil
	}

	auth, gErr := NewGcloudAuthenticator()
	if gErr == nil {
		return auth, nil
	}

	return nil, fmt.Errorf("failed to create token source from env: %v or gcloud: %v", envErr, gErr)
}
