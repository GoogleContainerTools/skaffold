package image

import (
	"fmt"
	"strings"

	"github.com/buildpacks/imgutil"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/pkg/errors"
)

// resolvePlatformSpecificDigest resolves a multi-platform image reference to a platform-specific digest.
// If the image is a manifest list, it finds the manifest for the specified platform and returns its digest.
// If the image is a single-platform image, it validates the platform matches and returns a digest reference.
func resolvePlatformSpecificDigest(imageRef string, platform *imgutil.Platform, keychain authn.Keychain, registrySettings map[string]imgutil.RegistrySetting) (string, error) {
	// If platform is nil, return the reference unchanged
	if platform == nil {
		return imageRef, nil
	}

	// Parse the reference (could be digest or tag)
	ref, err := name.ParseReference(imageRef, name.WeakValidation)
	if err != nil {
		return "", errors.Wrapf(err, "parsing image reference %q", imageRef)
	}

	// Get registry settings for the reference
	reg := getRegistrySetting(imageRef, registrySettings)

	// Get authentication
	auth, err := keychain.Resolve(ref.Context().Registry)
	if err != nil {
		return "", errors.Wrapf(err, "resolving authentication for registry %q", ref.Context().Registry)
	}

	// Fetch the descriptor
	desc, err := remote.Get(ref, remote.WithAuth(auth), remote.WithTransport(imgutil.GetTransport(reg.Insecure)))
	if err != nil {
		return "", errors.Wrapf(err, "fetching descriptor for %q", imageRef)
	}

	// Check if it's a manifest list
	if desc.MediaType == types.OCIImageIndex || desc.MediaType == types.DockerManifestList {
		// Get the index
		index, err := desc.ImageIndex()
		if err != nil {
			return "", errors.Wrapf(err, "getting image index for %q", imageRef)
		}

		// Get the manifest list
		manifestList, err := index.IndexManifest()
		if err != nil {
			return "", errors.Wrapf(err, "getting manifest list for %q", imageRef)
		}

		// Find the platform-specific manifest
		for _, manifest := range manifestList.Manifests {
			if manifest.Platform != nil {
				manifestPlatform := &imgutil.Platform{
					OS:           manifest.Platform.OS,
					Architecture: manifest.Platform.Architecture,
					Variant:      manifest.Platform.Variant,
					OSVersion:    manifest.Platform.OSVersion,
				}

				if platformsMatch(platform, manifestPlatform) {
					// Create a new digest reference for the platform-specific manifest
					platformDigestRef, err := name.NewDigest(
						fmt.Sprintf("%s@%s", ref.Context().Name(), manifest.Digest.String()),
						name.WeakValidation,
					)
					if err != nil {
						return "", errors.Wrapf(err, "creating platform-specific digest reference")
					}
					return platformDigestRef.String(), nil
				}
			}
		}

		return "", errors.Errorf("no manifest found for platform %s/%s%s in manifest list %q",
			platform.OS,
			platform.Architecture,
			platformString(platform),
			imageRef)
	}

	// If it's a single manifest, validate that the platform matches
	img, err := desc.Image()
	if err != nil {
		return "", errors.Wrapf(err, "getting image for %q", imageRef)
	}

	configFile, err := img.ConfigFile()
	if err != nil {
		return "", errors.Wrapf(err, "getting config file for %q", imageRef)
	}

	// Create platform from image config
	imagePlatform := &imgutil.Platform{
		OS:           configFile.OS,
		Architecture: configFile.Architecture,
		Variant:      configFile.Variant,
		OSVersion:    configFile.OSVersion,
	}

	// Check if the image's platform matches the requested platform
	if !platformsMatch(platform, imagePlatform) {
		return "", errors.Errorf("image platform %s/%s%s does not match requested platform %s/%s%s for %q",
			configFile.OS,
			configFile.Architecture,
			platformString(imagePlatform),
			platform.OS,
			platform.Architecture,
			platformString(platform),
			imageRef)
	}

	// Platform matches - if input was a digest reference, return it unchanged
	// If input was a tag reference, return the digest reference for consistency
	if _, ok := ref.(name.Digest); ok {
		return imageRef, nil
	}

	// Convert tag reference to digest reference
	digest, err := img.Digest()
	if err != nil {
		return "", errors.Wrapf(err, "getting digest for image %q", imageRef)
	}

	digestRef, err := name.NewDigest(
		fmt.Sprintf("%s@%s", ref.Context().Name(), digest.String()),
		name.WeakValidation,
	)
	if err != nil {
		return "", errors.Wrapf(err, "creating digest reference for %q", imageRef)
	}

	return digestRef.String(), nil
}

// platformsMatch checks if two platforms match.
// OS and Architecture must match exactly.
// For Variant and OSVersion, if either is blank, it's considered a match.
func platformsMatch(p1, p2 *imgutil.Platform) bool {
	if p1 == nil || p2 == nil {
		return false
	}

	// OS and Architecture must match exactly
	if p1.OS != p2.OS || p1.Architecture != p2.Architecture {
		return false
	}

	// For Variant and OSVersion, if either is blank, consider it a match
	variantMatch := p1.Variant == "" || p2.Variant == "" || p1.Variant == p2.Variant
	osVersionMatch := p1.OSVersion == "" || p2.OSVersion == "" || p1.OSVersion == p2.OSVersion

	return variantMatch && osVersionMatch
}

// platformString returns a pretty-printed string representation of a platform's variant and OS version.
// Returns empty string if both are blank, otherwise returns "/variant:osversion" format.
func platformString(platform *imgutil.Platform) string {
	if platform == nil {
		return ""
	}

	var parts []string

	if platform.Variant != "" {
		parts = append(parts, platform.Variant)
	}

	if platform.OSVersion != "" {
		parts = append(parts, platform.OSVersion)
	}

	if len(parts) == 0 {
		return ""
	}

	result := "/" + parts[0]
	if len(parts) > 1 {
		result += ":" + parts[1]
	}

	return result
}

// getRegistrySetting returns the registry setting for a given repository name.
// It checks if any prefix in the settings map matches the repository name.
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
