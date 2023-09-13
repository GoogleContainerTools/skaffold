package image

import (
	"github.com/buildpacks/imgutil"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/authn"
)

type Handler interface {
	InitImage(imageRef string) (imgutil.Image, error)
	Kind() string
}

// NewHandler creates a new Handler according to the arguments provided, following these rules:
// - WHEN layoutDir is defined and useLayout is true then it returns a LayoutHandler
// - WHEN a docker client is provided then it returns a LocalHandler
// - WHEN an auth.Keychain is provided then it returns a RemoteHandler
// - Otherwise nil is returned
func NewHandler(docker client.CommonAPIClient, keychain authn.Keychain, layoutDir string, useLayout bool) Handler {
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
			keychain: keychain,
		}
	}
	return nil
}
