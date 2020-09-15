package pack

import (
	"context"

	"github.com/buildpacks/pack/config"

	"github.com/buildpacks/lifecycle"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/style"
)

type RebaseOptions struct {
	RepoName          string
	Publish           bool
	PullPolicy        config.PullPolicy
	RunImage          string
	AdditionalMirrors map[string][]string
}

func (c *Client) Rebase(ctx context.Context, opts RebaseOptions) error {
	imageRef, err := c.parseTagReference(opts.RepoName)
	if err != nil {
		return errors.Wrapf(err, "invalid image name '%s'", opts.RepoName)
	}

	appImage, err := c.imageFetcher.Fetch(ctx, opts.RepoName, !opts.Publish, opts.PullPolicy)
	if err != nil {
		return err
	}

	var md lifecycle.LayersMetadataCompat
	if ok, err := dist.GetLabel(appImage, lifecycle.LayerMetadataLabel, &md); err != nil {
		return err
	} else if !ok {
		return errors.Errorf("could not find label %s on image", style.Symbol(lifecycle.LayerMetadataLabel))
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

	baseImage, err := c.imageFetcher.Fetch(ctx, runImageName, !opts.Publish, opts.PullPolicy)
	if err != nil {
		return err
	}

	c.logger.Infof("Rebasing %s on run image %s", style.Symbol(appImage.Name()), style.Symbol(baseImage.Name()))
	rebaser := &lifecycle.Rebaser{Logger: c.logger}
	err = rebaser.Rebase(appImage, baseImage, nil)
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
