package podman

import (
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
)

// Builder is an artifact builder that uses podman
type Builder struct {
	localDocker docker.LocalDaemon
	pushImages  bool
}

// NewArtifactBuilder returns a new podman ArtifactBuilder
func NewArtifactBuilder(localDocker docker.LocalDaemon, pushImages bool) *Builder {

	return &Builder{
		localDocker: localDocker,
		pushImages:  pushImages,
	}
}
