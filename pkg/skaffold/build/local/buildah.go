package local

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/podman"
)

// TODO: this isn't a nice workaround for creating small interfaces for abstracting docker.LocalDaemon.
// But since docker.LocalDaemon uses docker native types, I think this is the best way
// Would be better to create a global Runtime Interface, with self defined types
// so every runtime can implement it

type localBuildah struct {
	*podman.Buildah
}

func NewLocalBuildah(client *podman.Buildah) *localBuildah {
	return &localBuildah{Buildah: client}
}

func (l *localBuildah) ListImages(ctx context.Context, name string) (sums []imageSummary, err error) {
	imgs, err := l.Buildah.ListImages(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("buildah listing images: %w", err)
	}
	for _, img := range imgs {
		sums = append(sums, imageSummary{
			id:      img.ID(),
			created: img.Created().Unix(),
		})
	}
	return sums, nil
}

func (l *localBuildah) Push(ctx context.Context, w io.Writer, ref string) (string, error) {
	return l.Buildah.Push(ctx, ref)
}
func (l *localBuildah) Pull(ctx context.Context, w io.Writer, ref string) error {
	return l.Buildah.Pull(ctx, ref)
}
