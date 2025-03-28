// Package image implements functions for manipulating images
package image

import (
	"github.com/buildpacks/imgutil"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/authn"
)

// Handler wraps initialization of an [imgutil] image.
//
// [imgutil]: github.com/buildpacks/imgutil
//
//go:generate mockgen -package testmock -destination ../phase/testmock/image_handler.go github.com/buildpacks/lifecycle/image Handler
type Handler interface {
	InitImage(imageRef string) (imgutil.Image, error)
	Kind() string
}

// NewHandler creates a new Handler according to the arguments provided, following these rules:
// - WHEN layoutDir is defined and useLayout is true then it returns a LayoutHandler
// - WHEN a docker client is provided then it returns a LocalHandler
// - WHEN an auth.Keychain is provided then it returns a RemoteHandler
// - Otherwise nil is returned
func NewHandler(docker client.CommonAPIClient, keychain authn.Keychain, layoutDir string, useLayout bool, insecureRegistries []string) Handler {
	if layoutDir != "" && useLayout {
		return &LayoutHandler{
			layoutDir: layoutDir,
		}
	}
	if docker != nil {
		return &LocalHandler{
			docker: docker,
		}
	}
	if keychain != nil {
		return &RemoteHandler{
			keychain:           keychain,
			insecureRegistries: insecureRegistries,
		}
	}
	return nil
}
