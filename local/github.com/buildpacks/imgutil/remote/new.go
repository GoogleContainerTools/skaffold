package remote

import (
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
)

// NewImage returns a new image that can be modified and saved to an OCI image registry.
func NewImage(repoName string, keychain authn.Keychain, ops ...imgutil.ImageOption) (*Image, error) {
	options := &imgutil.ImageOptions{}
	for _, op := range ops {
		op(options)
	}

	options.Platform = processPlatformOption(options.Platform)

	var err error
	options.PreviousImage, err = processImageOption(options.PreviousImageRepoName, keychain, options.Platform, options.RegistrySettings)
	if err != nil {
		return nil, err
	}

	options.BaseImage, err = processImageOption(options.BaseImageRepoName, keychain, options.Platform, options.RegistrySettings)
	if err != nil {
		return nil, err
	}
	options.MediaTypes = imgutil.GetPreferredMediaTypes(*options)
	if options.BaseImage != nil {
		options.BaseImage, _, err = imgutil.EnsureMediaTypesAndLayers(options.BaseImage, options.MediaTypes, imgutil.PreserveLayers)
		if err != nil {
			return nil, err
		}
	}

	cnbImage, err := imgutil.NewCNBImage(*options)
	if err != nil {
		return nil, err
	}

	return &Image{
		CNBImageCore:        cnbImage,
		repoName:            repoName,
		keychain:            keychain,
		addEmptyLayerOnSave: options.AddEmptyLayerOnSave,
		registrySettings:    options.RegistrySettings,
	}, nil
}

func defaultPlatform() imgutil.Platform {
	return imgutil.Platform{
		OS:           "linux",
		Architecture: runtime.GOARCH,
	}
}

func processPlatformOption(requestedPlatform imgutil.Platform) imgutil.Platform {
	if (requestedPlatform != imgutil.Platform{}) {
		return requestedPlatform
	}
	return defaultPlatform()
}

func processImageOption(repoName string, keychain authn.Keychain, withPlatform imgutil.Platform, withRegistrySettings map[string]imgutil.RegistrySetting) (v1.Image, error) {
	if repoName == "" {
		return nil, nil
	}

	platform := v1.Platform{
		Architecture: withPlatform.Architecture,
		OS:           withPlatform.OS,
		Variant:      withPlatform.Variant,
		OSVersion:    withPlatform.OSVersion,
	}
	reg := getRegistrySetting(repoName, withRegistrySettings)
	ref, auth, err := referenceForRepoName(keychain, repoName, reg.Insecure)
	if err != nil {
		return nil, err
	}

	var image v1.Image
	for i := 0; i <= maxRetries; i++ {
		time.Sleep(100 * time.Duration(i) * time.Millisecond) // wait if retrying
		image, err = remote.Image(ref,
			remote.WithAuth(auth),
			remote.WithPlatform(platform),
			remote.WithTransport(imgutil.GetTransport(reg.Insecure)),
		)
		if err != nil {
			if err == io.EOF && i != maxRetries {
				continue // retry if EOF
			}
			if transportErr, ok := err.(*transport.Error); ok && len(transportErr.Errors) > 0 {
				switch transportErr.StatusCode {
				case http.StatusNotFound, http.StatusUnauthorized:
					return emptyImage(withPlatform)
				}
			}
			if strings.Contains(err.Error(), "no child with platform") {
				return emptyImage(withPlatform)
			}
			return nil, errors.Wrapf(err, "connect to repo store %q", repoName)
		}
		break
	}
	return image, nil
}

func getRegistrySetting(forRepoName string, givenSettings map[string]imgutil.RegistrySetting) imgutil.RegistrySetting {
	if givenSettings == nil {
		return imgutil.RegistrySetting{}
	}
	for prefix, r := range givenSettings {
		if strings.HasPrefix(forRepoName, prefix) {
			return r
		}
	}
	return imgutil.RegistrySetting{}
}

func referenceForRepoName(keychain authn.Keychain, ref string, insecure bool) (name.Reference, authn.Authenticator, error) {
	var auth authn.Authenticator
	opts := []name.Option{name.WeakValidation}
	if insecure {
		opts = append(opts, name.Insecure)
	}
	r, err := name.ParseReference(ref, opts...)
	if err != nil {
		return nil, nil, err
	}

	auth, err = keychain.Resolve(r.Context().Registry)
	if err != nil {
		return nil, nil, err
	}
	return r, auth, nil
}

func emptyImage(platform imgutil.Platform) (v1.Image, error) {
	cfg := &v1.ConfigFile{
		Architecture: platform.Architecture,
		History:      []v1.History{},
		OS:           platform.OS,
		Variant:      platform.Variant,
		OSVersion:    platform.OSVersion,
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: []v1.Hash{},
		},
	}

	return mutate.ConfigFile(empty.Image, cfg)
}

// NewV1Image returns a new v1.Image.
// It exists to provide library users (such as pack) an easy way to construct a v1.Image with configurable options
// (such as platform and insecure registry).
// FIXME: this function can be deprecated in favor of remote.NewImage as this now also implements the v1.Image interface
func NewV1Image(baseImageRepoName string, keychain authn.Keychain, ops ...func(*imgutil.ImageOptions)) (v1.Image, error) {
	options := &imgutil.ImageOptions{}
	for _, op := range ops {
		op(options)
	}
	options.Platform = processPlatformOption(options.Platform)
	return processImageOption(baseImageRepoName, keychain, options.Platform, options.RegistrySettings)
}
