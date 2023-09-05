package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/layer"
)

// NewImage returns a new Image that can be modified and saved to a registry.
func NewImage(repoName string, dockerClient DockerClient, ops ...ImageOption) (*Image, error) {
	imageOpts := &options{}
	for _, op := range ops {
		if err := op(imageOpts); err != nil {
			return nil, err
		}
	}

	platform, err := defaultPlatform(dockerClient)
	if err != nil {
		return nil, err
	}

	if (imageOpts.platform != imgutil.Platform{}) {
		if err := validatePlatformOption(platform, imageOpts.platform); err != nil {
			return nil, err
		}
		platform = imageOpts.platform
	}

	inspect := defaultInspect(platform)

	image := &Image{
		docker:           dockerClient,
		repoName:         repoName,
		inspect:          inspect,
		history:          make([]v1.History, len(inspect.RootFS.Layers)),
		layerPaths:       make([]string, len(inspect.RootFS.Layers)),
		downloadBaseOnce: &sync.Once{},
		withHistory:      imageOpts.withHistory,
	}

	if imageOpts.prevImageRepoName != "" {
		if err := processPreviousImageOption(image, imageOpts.prevImageRepoName, platform, dockerClient); err != nil {
			return nil, err
		}
	}

	if imageOpts.baseImageRepoName != "" {
		if err := processBaseImageOption(image, imageOpts.baseImageRepoName, platform, dockerClient); err != nil {
			return nil, err
		}
	}

	if image.inspect.Os == "windows" {
		if err := prepareNewWindowsImage(image); err != nil {
			return nil, err
		}
	}

	if imageOpts.createdAt.IsZero() {
		image.createdAt = imgutil.NormalizedDateTime
	} else {
		image.createdAt = imageOpts.createdAt
	}

	if imageOpts.config != nil {
		image.inspect.Config = imageOpts.config
	}

	return image, nil
}

func defaultPlatform(dockerClient DockerClient) (imgutil.Platform, error) {
	daemonInfo, err := dockerClient.Info(context.Background())
	if err != nil {
		return imgutil.Platform{}, err
	}

	return imgutil.Platform{
		OS:           daemonInfo.OSType,
		Architecture: "amd64",
	}, nil
}

func validatePlatformOption(defaultPlatform imgutil.Platform, optionPlatform imgutil.Platform) error {
	if optionPlatform.OS != "" && optionPlatform.OS != defaultPlatform.OS {
		return fmt.Errorf("invalid os: platform os %q must match the daemon os %q", optionPlatform.OS, defaultPlatform.OS)
	}

	return nil
}

func defaultInspect(platform imgutil.Platform) types.ImageInspect {
	return types.ImageInspect{
		Os:           platform.OS,
		Architecture: platform.Architecture,
		OsVersion:    platform.OSVersion,
		Config:       &container.Config{},
	}
}

func processPreviousImageOption(image *Image, prevImageRepoName string, platform imgutil.Platform, dockerClient DockerClient) error {
	inspect, err := inspectOptionalImage(dockerClient, prevImageRepoName, platform)
	if err != nil {
		return err
	}

	history, err := historyOptionalImage(dockerClient, prevImageRepoName)
	if err != nil {
		return err
	}

	v1History := toV1History(history)
	if len(history) != len(inspect.RootFS.Layers) {
		v1History = make([]v1.History, len(inspect.RootFS.Layers))
	}

	prevImage, err := NewImage(prevImageRepoName, dockerClient, FromBaseImage(prevImageRepoName))
	if err != nil {
		return errors.Wrapf(err, "getting previous image %q", prevImageRepoName)
	}

	image.prevImage = prevImage
	image.prevImage.history = v1History

	return nil
}

func inspectOptionalImage(docker DockerClient, imageName string, platform imgutil.Platform) (types.ImageInspect, error) {
	var (
		err     error
		inspect types.ImageInspect
	)
	if inspect, _, err = docker.ImageInspectWithRaw(context.Background(), imageName); err != nil {
		if client.IsErrNotFound(err) {
			return defaultInspect(platform), nil
		}

		return types.ImageInspect{}, errors.Wrapf(err, "verifying image %q", imageName)
	}
	return inspect, nil
}

func historyOptionalImage(docker DockerClient, imageName string) ([]image.HistoryResponseItem, error) {
	var (
		history []image.HistoryResponseItem
		err     error
	)
	if history, err = docker.ImageHistory(context.Background(), imageName); err != nil {
		if client.IsErrNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting history for image: %w", err)
	}
	return history, nil
}

func processBaseImageOption(image *Image, baseImageRepoName string, platform imgutil.Platform, dockerClient DockerClient) error {
	inspect, err := inspectOptionalImage(dockerClient, baseImageRepoName, platform)
	if err != nil {
		return err
	}

	history, err := historyOptionalImage(dockerClient, baseImageRepoName)
	if err != nil {
		return err
	}

	v1History := imgutil.NormalizedHistory(toV1History(history), len(inspect.RootFS.Layers))

	image.inspect = inspect
	image.history = v1History
	image.layerPaths = make([]string, len(image.inspect.RootFS.Layers))

	return nil
}

func toV1History(history []image.HistoryResponseItem) []v1.History {
	v1History := make([]v1.History, len(history))
	for offset, h := range history {
		// the daemon reports history in reverse order, so build up the array backwards
		v1History[len(v1History)-offset-1] = v1.History{
			Created:   v1.Time{Time: time.Unix(h.Created, 0)},
			CreatedBy: h.CreatedBy,
			Comment:   h.Comment,
		}
	}
	return v1History
}

func prepareNewWindowsImage(image *Image) error {
	// only append base layer to empty image
	if len(image.inspect.RootFS.Layers) > 0 {
		return nil
	}

	layerReader, err := layer.WindowsBaseLayer()
	if err != nil {
		return err
	}

	layerFile, err := ioutil.TempFile("", "imgutil.local.image.windowsbaselayer")
	if err != nil {
		return errors.Wrap(err, "creating temp file")
	}
	defer layerFile.Close()

	hasher := sha256.New()

	multiWriter := io.MultiWriter(layerFile, hasher)

	if _, err := io.Copy(multiWriter, layerReader); err != nil {
		return errors.Wrap(err, "copying base layer")
	}

	diffID := "sha256:" + hex.EncodeToString(hasher.Sum(nil))

	if err := image.AddLayerWithDiffIDAndHistory(layerFile.Name(), diffID, v1.History{}); err != nil {
		return errors.Wrap(err, "adding base layer to image")
	}

	return nil
}
