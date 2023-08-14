package containerengine

import "context"

type ImageManager interface {
	TagWithImageID(ctx context.Context, ref string, imageID string) (string, error)
}
