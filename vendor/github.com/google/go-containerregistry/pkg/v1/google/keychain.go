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

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

// Keychain exports an instance of the google Keychain.
var Keychain authn.Keychain = &googleKeychain{}

type googleKeychain struct{}

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
func (gk *googleKeychain) Resolve(reg name.Registry) (authn.Authenticator, error) {
	// Only authenticate GCR so it works with authn.NewMultiKeychain to fallback.
	if !strings.HasSuffix(reg.String(), "gcr.io") {
		return authn.Anonymous, nil
	}

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
