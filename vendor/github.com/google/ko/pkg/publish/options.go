// Copyright 2018 ko Build Authors All Rights Reserved.
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

package publish

import (
	"crypto/tls"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
)

type staticKeychain struct {
	auth authn.Authenticator
}

func (s staticKeychain) Resolve(authn.Resource) (authn.Authenticator, error) {
	return s.auth, nil
}

// WithTransport is a functional option for overriding the default transport
// on a default publisher.
func WithTransport(t http.RoundTripper) Option {
	return func(i *defaultOpener) error {
		i.t = t
		return nil
	}
}

// WithUserAgent is a functional option for overriding the User-Agent
// on a default publisher.
func WithUserAgent(ua string) Option {
	return func(i *defaultOpener) error {
		i.userAgent = ua
		return nil
	}
}

// WithAuth is a functional option for overriding the default authenticator
// on a default publisher.
func WithAuth(a authn.Authenticator) Option {
	return func(i *defaultOpener) error {
		i.keychain = staticKeychain{a}
		return nil
	}
}

// WithAuthFromKeychain is a functional option for overriding the default
// authenticator on a default publisher using an authn.Keychain
func WithAuthFromKeychain(keys authn.Keychain) Option {
	return func(i *defaultOpener) error {
		i.keychain = keys
		return nil
	}
}

// WithNamer is a functional option for overriding the image naming behavior
// in our default publisher.
func WithNamer(n Namer) Option {
	return func(i *defaultOpener) error {
		i.namer = n
		return nil
	}
}

// WithTags is a functional option for overriding the image tags
func WithTags(tags []string) Option {
	return func(i *defaultOpener) error {
		i.tags = tags
		return nil
	}
}

// WithTagOnly is a functional option for resolving images into tag-only references
func WithTagOnly(tagOnly bool) Option {
	return func(i *defaultOpener) error {
		i.tagOnly = tagOnly
		return nil
	}
}

func Insecure(b bool) Option {
	return func(i *defaultOpener) error {
		i.insecure = b
		t, ok := i.t.(*http.Transport)
		if !ok {
			return nil
		}
		t = t.Clone()
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{} //nolint: gosec
		}
		t.TLSClientConfig.InsecureSkipVerify = b //nolint: gosec
		i.t = t

		return nil
	}
}

// WithJobs limits the number of concurrent pushes.
func WithJobs(jobs int) Option {
	return func(i *defaultOpener) error {
		i.jobs = jobs
		return nil
	}
}
