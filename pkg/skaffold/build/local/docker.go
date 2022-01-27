package local

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

type localDocker struct {
	docker.LocalDaemon
}

func NewLocalDocker(client docker.LocalDaemon) *localDocker {
	return &localDocker{
		LocalDaemon: client,
	}
}

func (l *localDocker) ListImages(ctx context.Context, name string) (sums []imageSummary, err error) {
	imgs, err := l.ImageList(ctx, name)
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

func (l *localDocker) GetImageID(ctx context.Context, tag string) (string, error) {
	insp, _, err := l.ImageInspectWithRaw(ctx, tag)
	if err != nil {
		return "", err
	}
	return insp.ID, nil
}
