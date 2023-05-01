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
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
)

// WithTransport is a functional option for overriding the default transport
// on a remote image
func WithTransport(t http.RoundTripper) Option {
	return func(l *lister) error {
		l.transport = t
		return nil
	}
}

// WithAuth is a functional option for overriding the default authenticator
// on a remote image
func WithAuth(auth authn.Authenticator) Option {
	return func(l *lister) error {
		l.auth = auth
		return nil
	}
}

// WithAuthFromKeychain is a functional option for overriding the default
// authenticator on a remote image using an authn.Keychain
func WithAuthFromKeychain(keys authn.Keychain) Option {
	return func(l *lister) error {
		auth, err := keys.Resolve(l.repo.Registry)
		if err != nil {
			return err
		}
		l.auth = auth
		return nil
	}
}

// WithContext is a functional option for overriding the default
// context.Context for HTTP request to list remote images
func WithContext(ctx context.Context) Option {
	return func(l *lister) error {
		l.ctx = ctx
		return nil
	}
}

// WithUserAgent adds the given string to the User-Agent header for any HTTP
// requests. This header will also include "go-containerregistry/${version}".
//
// If you want to completely overwrite the User-Agent header, use WithTransport.
func WithUserAgent(ua string) Option {
	return func(l *lister) error {
		l.userAgent = ua
		return nil
	}
}
