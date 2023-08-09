package client

import (
	"context"

	"github.com/buildpacks/lifecycle"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/build"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

// RebaseOptions is a configuration struct that controls image rebase behavior.
type RebaseOptions struct {
	// Name of image we wish to rebase.
	RepoName string

	// Flag to publish image to remote registry after rebase completion.
	Publish bool

	// Strategy for pulling images during rebase.
	PullPolicy image.PullPolicy

	// Image to rebase against. This image must have
	// the same StackID as the previous run image.
	RunImage string

	// A mapping from StackID to an array of mirrors.
	// This mapping used only if both RunImage is omitted and Publish is true.
	// AdditionalMirrors gives us inputs to recalculate the 'best' run image
	// based on the registry we are publishing to.
	AdditionalMirrors map[string][]string
}

// Rebase updates the run image layers in an app image.
// This operation mutates the image specified in opts.
func (c *Client) Rebase(ctx context.Context, opts RebaseOptions) error {
	imageRef, err := c.parseTagReference(opts.RepoName)
	if err != nil {
		return errors.Wrapf(err, "invalid image name '%s'", opts.RepoName)
	}

	appImage, err := c.imageFetcher.Fetch(ctx, opts.RepoName, image.FetchOptions{Daemon: !opts.Publish, PullPolicy: opts.PullPolicy})
	if err != nil {
		return err
	}

	var md platform.LayersMetadataCompat
	if ok, err := dist.GetLabel(appImage, platform.LayerMetadataLabel, &md); err != nil {
		return err
	} else if !ok {
		return errors.Errorf("could not find label %s on image", style.Symbol(platform.LayerMetadataLabel))
	}

	runImageName := c.resolveRunImage(
		opts.RunImage,
		imageRef.Context().RegistryStr(),
		"",
		builder.StackMetadata{
			RunImage: builder.RunImageMetadata{
				Image:   md.Stack.RunImage.Image,
				Mirrors: md.Stack.RunImage.Mirrors,
			},
		},
		opts.AdditionalMirrors,
		opts.Publish)

	if runImageName == "" {
		return errors.New("run image must be specified")
	}

	baseImage, err := c.imageFetcher.Fetch(ctx, runImageName, image.FetchOptions{Daemon: !opts.Publish, PullPolicy: opts.PullPolicy})
	if err != nil {
		return err
	}

	c.logger.Infof("Rebasing %s on run image %s", style.Symbol(appImage.Name()), style.Symbol(baseImage.Name()))
	rebaser := &lifecycle.Rebaser{Logger: c.logger, PlatformAPI: build.SupportedPlatformAPIVersions.Latest()}
	_, err = rebaser.Rebase(appImage, baseImage, nil)
	if err != nil {
		return err
	}

	appImageIdentifier, err := appImage.Identifier()
	if err != nil {
		return err
	}

	c.logger.Infof("Rebased Image: %s", style.Symbol(appImageIdentifier.String()))
	return nil
}
