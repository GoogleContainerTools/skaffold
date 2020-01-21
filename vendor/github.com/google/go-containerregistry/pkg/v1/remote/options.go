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

package remote

import (
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// Option is a functional option for remote operations.
type Option func(*options) error

type options struct {
	auth      authn.Authenticator
	keychain  authn.Keychain
	transport http.RoundTripper
	platform  v1.Platform
}

func makeOptions(target authn.Resource, opts ...Option) (*options, error) {
	o := &options{
		auth:      authn.Anonymous,
		transport: http.DefaultTransport,
		platform:  defaultPlatform,
	}

	for _, option := range opts {
		if err := option(o); err != nil {
			return nil, err
		}
	}

	if o.keychain != nil {
		auth, err := o.keychain.Resolve(target)
		if err != nil {
			return nil, err
		}
		if auth == authn.Anonymous {
			logs.Warn.Println("No matching credentials were found, falling back on anonymous")
		}
		o.auth = auth
	}

	// Wrap the transport in something that logs requests and responses.
	// It's expensive to generate the dumps, so skip it if we're writing
	// to nothing.
	if logs.Enabled(logs.Debug) {
		o.transport = transport.NewLogger(o.transport)
	}

	// Wrap the transport in something that can retry network flakes.
	o.transport = transport.NewRetry(o.transport)

	return o, nil
}

// WithTransport is a functional option for overriding the default transport
// for remote operations.
//
// The default transport its http.DefaultTransport.
func WithTransport(t http.RoundTripper) Option {
	return func(o *options) error {
		o.transport = t
		return nil
	}
}

// WithAuth is a functional option for overriding the default authenticator
// for remote operations.
//
// The default authenticator is authn.Anonymous.
func WithAuth(auth authn.Authenticator) Option {
	return func(o *options) error {
		o.auth = auth
		return nil
	}
}

// WithAuthFromKeychain is a functional option for overriding the default
// authenticator for remote operations, using an authn.Keychain to find
// credentials.
//
// The default authenticator is authn.Anonymous.
func WithAuthFromKeychain(keys authn.Keychain) Option {
	return func(o *options) error {
		o.keychain = keys
		return nil
	}
}

// WithPlatform is a functional option for overriding the default platform
// that Image and Descriptor.Image use for resolving an index to an image.
//
// The default platform is amd64/linux.
func WithPlatform(p v1.Platform) Option {
	return func(o *options) error {
		o.platform = p
		return nil
	}
}
