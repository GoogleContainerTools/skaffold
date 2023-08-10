package image

import (
	"github.com/buildpacks/imgutil/remote"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
)

type RegistryInputs interface {
	ReadableRegistryImages() []string
	WriteableRegistryImages() []string
}

// ValidateDestinationTags ensures all tags are valid
// daemon - when false (exporting to a registry), ensures all tags are on the same registry
func ValidateDestinationTags(daemon bool, repoNames ...string) error {
	var (
		reg        string
		registries = map[string]struct{}{}
	)

	for _, repoName := range repoNames {
		ref, err := name.ParseReference(repoName, name.WeakValidation)
		if err != nil {
			return err
		}
		reg = ref.Context().RegistryStr()
		registries[reg] = struct{}{}
	}

	if !daemon && len(registries) != 1 {
		return errors.New("writing to multiple registries is unsupported")
	}

	return nil
}

func VerifyRegistryAccess(regInputs RegistryInputs, keychain authn.Keychain) error {
	for _, imageRef := range regInputs.ReadableRegistryImages() {
		err := verifyReadAccess(imageRef, keychain)
		if err != nil {
			return err
		}
	}
	for _, imageRef := range regInputs.WriteableRegistryImages() {
		err := verifyReadWriteAccess(imageRef, keychain)
		if err != nil {
			return err
		}
	}
	return nil
}

func verifyReadAccess(imageRef string, keychain authn.Keychain) error {
	img, _ := remote.NewImage(imageRef, keychain)
	if !img.CheckReadAccess() {
		return errors.Errorf("ensure registry read access to %s", imageRef)
	}
	return nil
}

func verifyReadWriteAccess(imageRef string, keychain authn.Keychain) error {
	img, _ := remote.NewImage(imageRef, keychain)
	if !img.CheckReadWriteAccess() {
		return errors.Errorf("ensure registry read/write access to %s", imageRef)
	}
	return nil
}
