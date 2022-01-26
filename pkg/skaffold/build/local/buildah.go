package local

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/buildah"
)

// TODO: this isn't a nice workaround for creating small interfaces for abstracting docker.LocalDaemon.
// But since docker.LocalDaemon uses docker native types, I think this is the best way
// Would be better to create a global Runtime Interface, with self defined types
// so every runtime can implement it

type localBuildah struct {
	client *buildah.Buildah
}

func NewLocalBuildah(client *buildah.Buildah) *localBuildah {
	return &localBuildah{
		client: client,
	}
}

func (l *localBuildah) ListImages(ctx context.Context, name string) (sums []imageSummary, err error) {
	imgs, err := l.client.ListImages(ctx, name)
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

func (l *localBuildah) Prune(ctx context.Context, ids []string, pruneChildren bool) ([]string, error) {
	return l.client.Prune(ctx, ids, pruneChildren)
}

func (l *localBuildah) DiskUsage(ctx context.Context) (uint64, error) {
	return l.client.DiskUsage(ctx)
}

func (l *localBuildah) GetImageID(ctx context.Context, tag string) (string, error) {
	return l.client.GetImageID(ctx, tag)
}

func (l *localBuildah) TagImage(ctx context.Context, tag string, imageID string) error {
	return l.client.TagImage(ctx, imageID, tag)
}

func (l *localBuildah) TagImageWithImageID(ctx context.Context, tag string, imageID string) (string, error) {
	return l.client.TagImageWithImageID(ctx, tag, imageID)
}

func (l *localBuildah) Push(ctx context.Context, w io.Writer, ref string) (string, error) {
	return l.client.Push(ctx, w, ref)
}
func (l *localBuildah) Pull(ctx context.Context, ref string) error {
	return l.client.Pull(ctx, ref)
}

func (l *localBuildah) ImageExists(ctx context.Context, ref string) bool {
	return l.client.ImageExists(ctx, ref)
}
