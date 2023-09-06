package image

import (
	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/local"
	"github.com/docker/docker/client"
)

const LocalKind = "docker"

type LocalHandler struct {
	docker client.CommonAPIClient
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
