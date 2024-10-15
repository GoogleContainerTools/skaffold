package image

import (
	"fmt"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/remote"
	"github.com/google/go-containerregistry/pkg/authn"
)

// RegistryHandler takes care of the registry settings and checks
//
//go:generate mockgen -package testmock -destination ../phase/testmock/registry_handler.go github.com/buildpacks/lifecycle/image RegistryHandler
type RegistryHandler interface {
	EnsureReadAccess(imageRefs ...string) error
	EnsureWriteAccess(imageRefs ...string) error
}

// DefaultRegistryHandler is the struct that implements the RegistryHandler methods
type DefaultRegistryHandler struct {
	keychain         authn.Keychain
	insecureRegistry []string
}

// NewRegistryHandler creates a new DefaultRegistryHandler
func NewRegistryHandler(keychain authn.Keychain, insecureRegistries []string) *DefaultRegistryHandler {
	return &DefaultRegistryHandler{
		keychain:         keychain,
		insecureRegistry: insecureRegistries,
	}
}

// EnsureReadAccess ensures that we can read from the registry
func (rv *DefaultRegistryHandler) EnsureReadAccess(imageRefs ...string) error {
	for _, imageRef := range imageRefs {
		if err := verifyReadAccess(imageRef, rv.keychain, GetInsecureOptions(rv.insecureRegistry)); err != nil {
			return err
		}
	}
	return nil
}

// EnsureWriteAccess ensures that we can write to the registry
func (rv *DefaultRegistryHandler) EnsureWriteAccess(imageRefs ...string) error {
	for _, imageRef := range imageRefs {
		if err := verifyReadWriteAccess(imageRef, rv.keychain, GetInsecureOptions(rv.insecureRegistry)); err != nil {
			return err
		}
	}
	return nil
}

// GetInsecureOptions returns a list of WithRegistrySetting imageOptions matching the specified imageRef prefix
/*
TODO: This is a temporary solution in order to get insecure registries in other components too
TODO: Ideally we should fix the `imgutil.options` struct visibility in order to mock and test the `remote.WithRegistrySetting`
TODO: function correctly and use the RegistryHandler everywhere it is needed.
*/
func GetInsecureOptions(insecureRegistries []string) []imgutil.ImageOption {
	var opts []imgutil.ImageOption
	for _, insecureRegistry := range insecureRegistries {
		opts = append(opts, remote.WithRegistrySetting(insecureRegistry, true))
	}
	return opts
}

func verifyReadAccess(imageRef string, keychain authn.Keychain, opts []imgutil.ImageOption) error {
	if imageRef == "" {
		return nil
	}

	img, _ := remote.NewImage(imageRef, keychain, opts...)
	canRead, err := img.CheckReadAccess()
	if !canRead {
		return fmt.Errorf("failed to ensure registry read access to %s: %w", imageRef, err)
	}

	return nil
}

func verifyReadWriteAccess(imageRef string, keychain authn.Keychain, opts []imgutil.ImageOption) error {
	if imageRef == "" {
		return nil
	}

	img, _ := remote.NewImage(imageRef, keychain, opts...)
	canReadWrite, err := img.CheckReadWriteAccess()
	if !canReadWrite {
		return fmt.Errorf("failed to ensure registry read/write access to %s: %w", imageRef, err)
	}
	return nil
}
