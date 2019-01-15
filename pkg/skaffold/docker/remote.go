/*
Copyright 2018 The Skaffold Authors

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
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/pkg/errors"
)

func AddTag(src, target string) error {
	srcRef, err := name.ParseReference(src, name.WeakValidation)
	if err != nil {
		return errors.Wrap(err, "getting source reference")
	}

	auth, err := authn.DefaultKeychain.Resolve(srcRef.Context().Registry)
	if err != nil {
		return err
	}

	targetRef, err := name.ParseReference(target, name.WeakValidation)
	if err != nil {
		return errors.Wrap(err, "getting target reference")
	}

	return addTag(srcRef, targetRef, auth, http.DefaultTransport)
}

func addTag(ref name.Reference, targetRef name.Reference, auth authn.Authenticator, t http.RoundTripper) error {
	tr, err := transport.New(ref.Context().Registry, auth, t, []string{targetRef.Scope(transport.PushScope)})
	if err != nil {
		return err
	}

	img, err := remote.Image(ref, remote.WithAuth(auth), remote.WithTransport(tr))
	if err != nil {
		return err
	}

	return remote.Write(targetRef, img, auth, t)
}

func RemoteDigest(identifier string) (string, error) {
	img, err := remoteImage(identifier)
	if err != nil {
		return "", errors.Wrap(err, "getting image")
	}

	h, err := img.Digest()
	if err != nil {
		return "", errors.Wrap(err, "getting digest")
	}

	return h.String(), nil
}

func retrieveRemoteConfig(identifier string) (*v1.ConfigFile, error) {
	img, err := remoteImage(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "getting image")
	}

	return img.ConfigFile()
}

func remoteImage(identifier string) (v1.Image, error) {
	ref, err := name.ParseReference(identifier, name.WeakValidation)
	if err != nil {
		return nil, errors.Wrap(err, "parsing initial ref")
	}

	auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		return nil, errors.Wrap(err, "getting default keychain auth")
	}

	return remote.Image(ref, remote.WithAuth(auth), remote.WithTransport(http.DefaultTransport))
}
