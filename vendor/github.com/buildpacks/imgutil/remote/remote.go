package remote

import (
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
)

const maxRetries = 2

type Image struct {
	*imgutil.CNBImageCore
	repoName            string
	keychain            authn.Keychain
	addEmptyLayerOnSave bool
	registrySettings    map[string]imgutil.RegistrySetting
}

func (i *Image) Kind() string {
	return `remote`
}

func (i *Image) Name() string {
	return i.repoName
}

func (i *Image) Rename(name string) {
	i.repoName = name
}

func (i *Image) Found() bool {
	_, err := i.found()
	return err == nil
}

func (i *Image) found() (*v1.Descriptor, error) {
	reg := getRegistrySetting(i.repoName, i.registrySettings)
	ref, auth, err := referenceForRepoName(i.keychain, i.repoName, reg.Insecure)
	if err != nil {
		return nil, err
	}
	return remote.Head(ref, remote.WithAuth(auth), remote.WithTransport(imgutil.GetTransport(reg.Insecure)))
}

func (i *Image) Identifier() (imgutil.Identifier, error) {
	ref, err := name.ParseReference(i.repoName, name.WeakValidation)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing reference for image %q", i.repoName)
	}

	hash, err := i.Digest()
	if err != nil {
		return nil, errors.Wrapf(err, "getting digest for image %q", i.repoName)
	}

	digestRef, err := name.NewDigest(fmt.Sprintf("%s@%s", ref.Context().Name(), hash.String()), name.WeakValidation)
	if err != nil {
		return nil, errors.Wrap(err, "creating digest reference")
	}

	return DigestIdentifier{
		Digest: digestRef,
	}, nil
}

// Valid returns true if the (saved) image is valid.
func (i *Image) Valid() bool {
	return i.valid() == nil
}

func (i *Image) valid() error {
	reg := getRegistrySetting(i.repoName, i.registrySettings)
	ref, auth, err := referenceForRepoName(i.keychain, i.repoName, reg.Insecure)
	if err != nil {
		return err
	}
	desc, err := remote.Get(ref, remote.WithAuth(auth), remote.WithTransport(imgutil.GetTransport(reg.Insecure)))
	if err != nil {
		return err
	}
	if desc.MediaType == types.OCIImageIndex || desc.MediaType == types.DockerManifestList {
		index, err := desc.ImageIndex()
		if err != nil {
			return err
		}
		return validate.Index(index, validate.Fast)
	}
	img, err := desc.Image()
	if err != nil {
		return err
	}
	return validate.Image(img, validate.Fast)
}

func (i *Image) Delete() error {
	id, err := i.Identifier()
	if err != nil {
		return err
	}
	reg := getRegistrySetting(i.repoName, i.registrySettings)
	ref, auth, err := referenceForRepoName(i.keychain, id.String(), reg.Insecure)
	if err != nil {
		return err
	}
	return remote.Delete(ref, remote.WithAuth(auth), remote.WithTransport(imgutil.GetTransport(reg.Insecure)))
}

// extras

func (i *Image) CheckReadAccess() (bool, error) {
	var err error
	if _, err = i.found(); err == nil {
		return true, nil
	}
	var canRead bool
	if transportErr, ok := err.(*transport.Error); ok {
		if canRead = transportErr.StatusCode != http.StatusUnauthorized &&
			transportErr.StatusCode != http.StatusForbidden; canRead {
			err = nil
		}
	}
	return canRead, err
}

func (i *Image) CheckReadWriteAccess() (bool, error) {
	if canRead, err := i.CheckReadAccess(); !canRead {
		return false, err
	}
	reg := getRegistrySetting(i.repoName, i.registrySettings)
	ref, _, err := referenceForRepoName(i.keychain, i.repoName, reg.Insecure)
	if err != nil {
		return false, err
	}
	err = remote.CheckPushPermission(ref, i.keychain, http.DefaultTransport)
	if err != nil {
		return false, err
	}
	return true, nil
}

var _ imgutil.ImageIndex = (*ImageIndex)(nil)

type ImageIndex struct {
	*imgutil.CNBIndex
}
