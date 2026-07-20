package image

import (
	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/local"
	"github.com/moby/moby/client"
)

const LocalKind = "docker"

type LocalHandler struct {
	docker client.APIClient
}

func (h *LocalHandler) InitImage(imageRef string) (imgutil.Image, error) {
	if imageRef == "" {
		return nil, nil
	}

	return local.NewImage(
		imageRef,
		h.docker,
		local.FromBaseImage(imageRef),
	)
}

func (h *LocalHandler) Kind() string {
	return LocalKind
}
