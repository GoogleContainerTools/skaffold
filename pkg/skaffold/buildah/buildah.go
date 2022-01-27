package buildah

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/pkg/errors"
)

type Buildah struct {
	runtime *libimage.Runtime
	store   storage.Store
}

func New() (*Buildah, error) {
	store, err := getBuildStore()
	if err != nil {
		return nil, fmt.Errorf("getting build store: %w", err)
	}
	runtime, err := newLibImageRuntime(store)
	if err != nil {
		return nil, err
	}
	return &Buildah{
		runtime: runtime,
		store:   store,
	}, nil
}

var imageArchivePath = os.TempDir()

func (b *Buildah) ListImages(ctx context.Context, name string) (sums []*libimage.Image, err error) {
	return b.runtime.ListImages(ctx, []string{name}, &libimage.ListImagesOptions{})
	// if err != nil {
	// 	return nil, fmt.Errorf("libimage listing images: %w", err)
	// }
	// for _, img := range imgs {
	// 	sums = append(sums, imageSummary{
	// 		id:      img.ID(),
	// 		created: img.Created().Unix(),
	// 	})
	// }
	// return sums, nil
}

func (b *Buildah) Prune(ctx context.Context, ids []string, pruneChildren bool) ([]string, error) {
	reports, errs := b.runtime.RemoveImages(ctx, ids, &libimage.RemoveImagesOptions{Force: pruneChildren})
	if len(errs) > 0 {
		errorTexts := make([]string, len(errs))
		for _, err := range errs {
			if err != nil {
				errorTexts = append(errorTexts, err.Error())
			}
		}
		return nil, fmt.Errorf("removing images: %v", strings.Join(errorTexts, ";"))
	}

	var prunedIDs []string
	for _, report := range reports {
		prunedIDs = append(prunedIDs, report.ID)
	}
	return prunedIDs, nil
}

func (b *Buildah) DiskUsage(ctx context.Context) (uint64, error) {
	usages, err := b.runtime.DiskUsage(ctx)
	if err != nil {
		return 0, err
	}
	var total uint64
	for _, usage := range usages {
		total = total + uint64(usage.Size)
	}
	return total, nil
}

func (b *Buildah) GetImageID(ctx context.Context, tag string) (string, error) {
	image, err := b.findImage(tag)
	if err != nil {
		// dont return an error if the image wasnt found
		if errors.Cause(err) == storage.ErrImageUnknown {
			return "", nil
		}
		return "", fmt.Errorf("get buildah image: %w", err)
	}
	return image.ID(), nil
}

func (b *Buildah) Tag(ctx context.Context, tag string, imageID string) error {
	image, err := b.findImage(imageID)
	if err != nil {
		return err
	}
	return image.Tag(tag)
}

func (b *Buildah) TagWithImageID(ctx context.Context, tag string, imageID string) (string, error) {
	parsed, err := docker.ParseReference(tag)
	if err != nil {
		return "", err
	}

	image, err := b.findImage(imageID)
	if err != nil {
		return "", fmt.Errorf("get buildah image: %w", err)
	}
	uniqueTag := parsed.BaseName + ":" + strings.TrimPrefix(imageID, "sha256:")
	if err := image.Tag(uniqueTag); err != nil {
		return "", fmt.Errorf("tagging image: %w", err)
	}
	return uniqueTag, nil
}

func (b *Buildah) Push(ctx context.Context, ref string) (string, error) {
	digest, err := b.runtime.Push(ctx, ref, ref, &libimage.PushOptions{})
	if err != nil {
		return "", err
	}
	return string(digest), nil
}
func (l *Buildah) Pull(ctx context.Context, ref string) error {
	_, err := l.runtime.Pull(ctx, ref, config.PullPolicyMissing, &libimage.PullOptions{})
	return err
}

func (b *Buildah) Build(ctx context.Context, containerfilePath string, buildOptions define.BuildOptions) (string, error) {
	id, _, err := imagebuildah.BuildDockerfiles(ctx, b.store, buildOptions, containerfilePath)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (b *Buildah) ImageExists(ctx context.Context, ref string) bool {
	_, err := b.findImage(ref)
	if errors.Cause(err) == storage.ErrImageUnknown {
		return false
	}
	return true
}

// Save saves the image to disk as image archive and returns the path to it
// caller
func (b *Buildah) Save(ctx context.Context, images []string) (string, error) {
	path := filepath.Join(imageArchivePath, "skaffold-images.tar")

	return path, b.runtime.Save(ctx, images, "docker-archive", path, &libimage.SaveOptions{})
}

// findImage finds the ref in local storage
// ref can be a tag, digest, imageID or full reference
func (b *Buildah) findImage(ref string) (*libimage.Image, error) {
	image, _, err := b.runtime.LookupImage(ref, &libimage.LookupImageOptions{})
	if err != nil {
		return nil, err
	}
	return image, nil
}

// newLibImageRuntime returns a new libimage runtime with the default store
func newLibImageRuntime(store storage.Store) (*libimage.Runtime, error) {
	return libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{})
}

func getBuildStore() (storage.Store, error) {
	buildStoreOptions, err := storage.DefaultStoreOptions(unshare.IsRootless(), unshare.GetRootlessUID())
	if err != nil {
		return nil, fmt.Errorf("buildah store options: %w", err)
	}
	return storage.GetStore(buildStoreOptions)
}
