package util

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

func ConvertToV1Platform(platform specs.Platform) *v1.Platform {
	return &v1.Platform{Architecture: platform.Architecture, OS: platform.OS, OSVersion: platform.OSVersion, OSFeatures: platform.OSFeatures, Variant: platform.Variant}
}
