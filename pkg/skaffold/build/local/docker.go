package local

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

type localDocker struct {
	client docker.LocalDaemon
}

func NewLocalDocker(client docker.LocalDaemon) *localDocker {
	return &localDocker{
		client: client,
	}
}

func (l *localDocker) ListImages(ctx context.Context, name string) (sums []imageSummary, err error) {
	imgs, err := l.client.ImageList(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("docker listing images: %w", err)
	}
	for _, img := range imgs {
		sums = append(sums, imageSummary{
			id:      img.ID,
			created: img.Created,
		})
	}
	return sums, nil
}

func (l *localDocker) Prune(ctx context.Context, ids []string, pruneChildren bool) ([]string, error) {
	return l.client.Prune(ctx, ids, pruneChildren)
}

func (l *localDocker) DiskUsage(ctx context.Context) (uint64, error) {
	return l.client.DiskUsage(ctx)
}

func (l *localDocker) GetImageID(ctx context.Context, tag string) (string, error) {
	insp, _, err := l.client.ImageInspectWithRaw(ctx, tag)
	if err != nil {
		return "", err
	}
	return insp.ID, nil
}

func (l *localDocker) TagImage(ctx context.Context, tag string, imageID string) error {
	return l.client.Tag(ctx, imageID, tag)
}

func (l *localDocker) TagImageWithImageID(ctx context.Context, tag string, imageID string) (string, error) {
	return l.client.TagWithImageID(ctx, tag, imageID)
}

func (l *localDocker) Push(ctx context.Context, w io.Writer, ref string) (string, error) {
	return l.client.Push(ctx, w, ref)
}
func (l *localDocker) Pull(ctx context.Context, ref string) error {
	return l.client.Pull(ctx, io.Discard, ref)
}

func (l *localDocker) ImageExists(ctx context.Context, ref string) bool {
	return l.client.ImageExists(ctx, ref)
}
