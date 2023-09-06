package remote

import (
	"crypto/tls"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/layer"
)

// NewImage returns a new Image that can be modified and saved to a Docker daemon.
func NewImage(repoName string, keychain authn.Keychain, ops ...ImageOption) (*Image, error) {
	imageOpts := &options{}
	for _, op := range ops {
		if err := op(imageOpts); err != nil {
			return nil, err
		}
	}

	platform := defaultPlatform()
	if (imageOpts.platform != imgutil.Platform{}) {
		platform = imageOpts.platform
	}

	image, err := emptyImage(platform)
	if err != nil {
		return nil, err
	}

	ri := &Image{
		keychain:            keychain,
		repoName:            repoName,
		image:               image,
		addEmptyLayerOnSave: imageOpts.addEmptyLayerOnSave,
		withHistory:         imageOpts.withHistory,
		registrySettings:    imageOpts.registrySettings,
	}

	if imageOpts.prevImageRepoName != "" {
		if err := processPreviousImageOption(ri, imageOpts.prevImageRepoName, platform); err != nil {
			return nil, err
		}
	}

	if imageOpts.baseImageRepoName != "" {
		if err := processBaseImageOption(ri, imageOpts.baseImageRepoName, platform); err != nil {
			return nil, err
		}
	}

	imgOS, err := ri.OS()
	if err != nil {
		return nil, err
	}
	if imgOS == "windows" {
		if err := prepareNewWindowsImage(ri); err != nil {
			return nil, err
		}
	}

	if imageOpts.createdAt.IsZero() {
		ri.createdAt = imgutil.NormalizedDateTime
	} else {
		ri.createdAt = imageOpts.createdAt
	}

	if imageOpts.config != nil {
		if ri.image, err = mutate.Config(ri.image, *imageOpts.config); err != nil {
			return nil, err
		}
	}

	ri.requestedMediaTypes = imageOpts.mediaTypes
	if err = ri.setUnderlyingImage(ri.image); err != nil { // update media types
		return nil, err
	}

	return ri, nil
}

func defaultPlatform() imgutil.Platform {
	return imgutil.Platform{
		OS:           "linux",
		Architecture: "amd64",
	}
}

func emptyImage(platform imgutil.Platform) (v1.Image, error) {
	cfg := &v1.ConfigFile{
		Architecture: platform.Architecture,
		History:      []v1.History{},
		OS:           platform.OS,
		OSVersion:    platform.OSVersion,
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: []v1.Hash{},
		},
	}

	return mutate.ConfigFile(empty.Image, cfg)
}

func prepareNewWindowsImage(ri *Image) error {
	// only append base layer to empty image
	cfgFile, err := ri.image.ConfigFile()
	if err != nil {
		return err
	}
	if len(cfgFile.RootFS.DiffIDs) > 0 {
		return nil
	}

	layerBytes, err := layer.WindowsBaseLayer()
	if err != nil {
		return err
	}

	windowsBaseLayer, err := tarball.LayerFromReader(layerBytes) // TODO: LayerFromReader is deprecated; LayerFromOpener or stream.NewLayer are suggested alternatives however the tests do not pass when they are used
	if err != nil {
		return err
	}

	image, err := mutate.AppendLayers(ri.image, windowsBaseLayer)
	if err != nil {
		return err
	}

	ri.image = image

	return nil
}

func processPreviousImageOption(ri *Image, prevImageRepoName string, platform imgutil.Platform) error {
	reg := getRegistry(prevImageRepoName, ri.registrySettings)

	prevImage, err := NewV1Image(prevImageRepoName, ri.keychain, WithV1DefaultPlatform(platform), WithV1RegistrySetting(reg.insecure, reg.insecureSkipVerify))
	if err != nil {
		return err
	}

	prevLayers, err := prevImage.Layers()
	if err != nil {
		return errors.Wrapf(err, "getting layers for previous image with repo name %q", prevImageRepoName)
	}

	configFile, err := prevImage.ConfigFile()
	if err != nil {
		return err
	}

	ri.prevLayers = prevLayers
	prevHistory := configFile.History
	if len(prevLayers) != len(prevHistory) {
		prevHistory = make([]v1.History, len(prevLayers))
	}
	ri.prevHistory = prevHistory

	return nil
}

func getRegistry(repoName string, registrySettings map[string]registrySetting) registrySetting {
	for prefix, r := range registrySettings {
		if strings.HasPrefix(repoName, prefix) {
			return r
		}
	}
	return registrySetting{}
}

// NewV1Image returns a new v1.Image
func NewV1Image(baseImageRepoName string, keychain authn.Keychain, ops ...V1ImageOption) (v1.Image, error) {
	imageOpts := &v1Options{}
	for _, op := range ops {
		if err := op(imageOpts); err != nil {
			return nil, err
		}
	}

	platform := defaultPlatform()
	if (imageOpts.platform != imgutil.Platform{}) {
		platform = imageOpts.platform
	}

	reg := registrySetting{}
	if (imageOpts.registrySetting != registrySetting{}) {
		reg = imageOpts.registrySetting
	}

	baseImage, err := newV1Image(keychain, baseImageRepoName, platform, reg)
	if err != nil {
		return nil, err
	}
	return baseImage, nil
}

func newV1Image(keychain authn.Keychain, repoName string, platform imgutil.Platform, reg registrySetting) (v1.Image, error) {
	ref, auth, err := referenceForRepoName(keychain, repoName, reg.insecure)
	if err != nil {
		return nil, err
	}

	v1Platform := v1.Platform{
		Architecture: platform.Architecture,
		OS:           platform.OS,
		OSVersion:    platform.OSVersion,
	}

	opts := []remote.Option{remote.WithAuth(auth), remote.WithPlatform(v1Platform)}
	// #nosec G402
	if reg.insecureSkipVerify {
		opts = append(opts, remote.WithTransport(&http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}))
	} else {
		opts = append(opts, remote.WithTransport(http.DefaultTransport))
	}

	var image v1.Image
	for i := 0; i <= maxRetries; i++ {
		time.Sleep(100 * time.Duration(i) * time.Millisecond) // wait if retrying
		image, err = remote.Image(ref, opts...)
		if err != nil {
			if err == io.EOF && i != maxRetries {
				continue // retry if EOF
			}
			if transportErr, ok := err.(*transport.Error); ok && len(transportErr.Errors) > 0 {
				switch transportErr.StatusCode {
				case http.StatusNotFound, http.StatusUnauthorized:
					return emptyImage(platform)
				}
			}
			if strings.Contains(err.Error(), "no child with platform") {
				return emptyImage(platform)
			}
			return nil, errors.Wrapf(err, "connect to repo store %q", repoName)
		}
		break
	}

	return image, nil
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

func processBaseImageOption(ri *Image, baseImageRepoName string, platform imgutil.Platform) error {
	reg := getRegistry(baseImageRepoName, ri.registrySettings)
	var err error
	ri.image, err = NewV1Image(baseImageRepoName, ri.keychain, WithV1DefaultPlatform(platform), WithV1RegistrySetting(reg.insecure, reg.insecureSkipVerify))
	if err != nil {
		return err
	}
	if !ri.withHistory {
		return nil
	}
	if ri.image, err = imgutil.OverrideHistoryIfNeeded(ri.image); err != nil {
		return err
	}
	return nil
}

// setUnderlyingImage wraps the provided v1.Image into a layout.Image and sets it as the underlying image for the receiving layout.Image
func (i *Image) setUnderlyingImage(base v1.Image) error {
	manifest, err := base.Manifest()
	if err != nil {
		return err
	}
	if i.requestedMediaTypesMatch(manifest) {
		i.image = base
		return nil
	}
	// provided v1.Image media types differ from requested, override them
	newBase, err := imgutil.OverrideMediaTypes(base, i.requestedMediaTypes)
	if err != nil {
		return err
	}
	i.image = newBase
	return nil
}

// requestedMediaTypesMatch returns true if the manifest and config file use the requested media types
func (i *Image) requestedMediaTypesMatch(manifest *v1.Manifest) bool {
	return manifest.MediaType == i.requestedMediaTypes.ManifestType() &&
		manifest.Config.MediaType == i.requestedMediaTypes.ConfigType()
}
