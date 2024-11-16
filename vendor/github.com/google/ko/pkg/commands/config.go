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

package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	ecr "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands/options"
	"github.com/google/ko/pkg/publish"
)

var (
	amazonKeychain authn.Keychain = authn.NewKeychainFromHelper(ecr.NewECRHelper(ecr.WithLogger(io.Discard)))
	azureKeychain  authn.Keychain = authn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper())
	keychain                      = authn.NewMultiKeychain(
		amazonKeychain,
		authn.DefaultKeychain,
		google.Keychain,
		github.Keychain,
		azureKeychain,
	)
)

// getBaseImage returns a function that determines the base image for a given import path.
func getBaseImage(bo *options.BuildOptions) build.GetBase {
	userAgent := ua()
	if bo.UserAgent != "" {
		userAgent = bo.UserAgent
	}

	ropt := []remote.Option{
		remote.WithAuthFromKeychain(keychain),
		remote.WithUserAgent(userAgent),
	}
	puller, err := remote.NewPuller(ropt...)
	if err != nil {
		// This can't really happen.
		panic(err)
	}
	ropt = append(ropt, remote.Reuse(puller))

	cache, err := newCache(puller)
	if err != nil {
		log.Printf("Image cache init failed: %v", err)
	}

	fetch := func(ctx context.Context, ref name.Reference) (build.Result, error) {
		ropt = append(ropt, remote.WithContext(ctx))

		desc, err := remote.Get(ref, ropt...)
		if err != nil {
			return nil, err
		}
		if desc.MediaType.IsIndex() {
			return desc.ImageIndex()
		}
		return desc.Image()
	}

	return func(ctx context.Context, s string) (name.Reference, build.Result, error) {
		s = strings.TrimPrefix(s, build.StrictScheme)
		// Viper configuration file keys are case insensitive, and are
		// returned as all lowercase.  This means that import paths with
		// uppercase must be normalized for matching here, e.g.
		//    github.com/GoogleCloudPlatform/foo/cmd/bar
		// comes through as:
		//    github.com/googlecloudplatform/foo/cmd/bar
		baseImage, ok := bo.BaseImageOverrides[strings.ToLower(s)]
		if !ok || baseImage == "" {
			baseImage = bo.BaseImage
		}
		var nameOpts []name.Option
		if bo.InsecureRegistry {
			nameOpts = append(nameOpts, name.Insecure)
		}
		ref, err := name.ParseReference(baseImage, nameOpts...)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing base image (%q): %w", baseImage, err)
		}

		var result build.Result

		// For ko.local, look in the daemon.
		if ref.Context().RegistryStr() == publish.LocalDomain {
			result, err = daemon.Image(ref)
			if err != nil {
				return nil, nil, fmt.Errorf("loading %s from daemon: %w", ref, err)
			}
		} else {
			result, err = cache.get(ctx, ref, fetch)
			if err != nil {
				// We don't expect this to fail, usually, but the cache should also not be fatal.
				// Log it so people can complain about it and we can fix the cache.
				log.Printf("cache.get(%q) failed with %v", ref.String(), err)

				result, err = fetch(ctx, ref)
				if err != nil {
					return nil, nil, fmt.Errorf("pulling %s: %w", ref, err)
				}
			}
		}

		if _, ok := ref.(name.Digest); ok {
			log.Printf("Using base %s for %s", ref, s)
		} else {
			dig, err := result.Digest()
			if err != nil {
				return ref, result, err
			}
			log.Printf("Using base %s@%s for %s", ref, dig, s)
		}

		return ref, result, nil
	}
}

func getTimeFromEnv(env string) (*v1.Time, error) {
	epoch := os.Getenv(env)
	if epoch == "" {
		return nil, nil
	}

	seconds, err := strconv.ParseInt(epoch, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("the environment variable %s should be the number of seconds since January 1st 1970, 00:00 UTC, got: %w", env, err)
	}
	return &v1.Time{Time: time.Unix(seconds, 0)}, nil
}

func getCreationTime() (*v1.Time, error) {
	return getTimeFromEnv("SOURCE_DATE_EPOCH")
}

func getKoDataCreationTime() (*v1.Time, error) {
	return getTimeFromEnv("KO_DATA_DATE_EPOCH")
}

type baseFactory func(context.Context, name.Reference) (build.Result, error)
