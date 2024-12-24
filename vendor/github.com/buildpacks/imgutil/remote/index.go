package remote

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/buildpacks/imgutil"
)

// NewIndex returns a new ImageIndex from the registry that can be modified and saved to the local file system.
func NewIndex(repoName string, ops ...imgutil.IndexOption) (*imgutil.CNBIndex, error) {
	options := &imgutil.IndexOptions{}
	for _, op := range ops {
		if err := op(options); err != nil {
			return nil, err
		}
	}

	var err error

	if options.BaseIndex == nil && options.BaseIndexRepoName != "" { // options.BaseIndex supersedes options.BaseIndexRepoName
		options.BaseIndex, err = newV1Index(
			options.BaseIndexRepoName,
			options.Keychain,
			options.Insecure,
		)
		if err != nil {
			return nil, err
		}
	}

	return imgutil.NewCNBIndex(repoName, *options)
}

func newV1Index(repoName string, keychain authn.Keychain, insecure bool) (v1.ImageIndex, error) {
	ref, err := name.ParseReference(repoName, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	desc, err := remote.Get(
		ref,
		remote.WithAuthFromKeychain(keychain),
		remote.WithTransport(imgutil.GetTransport(insecure)),
	)
	if err != nil {
		return nil, err
	}
	return desc.ImageIndex()
}
