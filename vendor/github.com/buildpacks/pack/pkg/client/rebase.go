package client

import (
	"context"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/lifecycle/phase"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/buildpacks/lifecycle/platform/files"
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

	// If provided, directory to which report.toml will be copied
	ReportDestinationDir string

	// Pass-through force flag to lifecycle rebase command to skip target data
	// validated (will not have any effect if API < 0.12).
	Force bool

	InsecureRegistries []string

	// Image reference to use as the previous image for rebase.
	PreviousImage string
}

// Rebase updates the run image layers in an app image.
// This operation mutates the image specified in opts.
func (c *Client) Rebase(ctx context.Context, opts RebaseOptions) error {
	var flags = []string{"rebase"}
	imageRef, err := c.parseTagReference(opts.RepoName)
	if err != nil {
		return errors.Wrapf(err, "invalid image name '%s'", opts.RepoName)
	}

	repoName := opts.RepoName

	if opts.PreviousImage != "" {
		repoName = opts.PreviousImage
	}

	appImage, err := c.imageFetcher.Fetch(ctx, repoName, image.FetchOptions{Daemon: !opts.Publish, PullPolicy: opts.PullPolicy, InsecureRegistries: opts.InsecureRegistries})
	if err != nil {
		return err
	}

	appOS, err := appImage.OS()
	if err != nil {
		return errors.Wrapf(err, "getting app OS")
	}

	appArch, err := appImage.Architecture()
	if err != nil {
		return errors.Wrapf(err, "getting app architecture")
	}

	var md files.LayersMetadataCompat
	if ok, err := dist.GetLabel(appImage, platform.LifecycleMetadataLabel, &md); err != nil {
		return err
	} else if !ok {
		return errors.Errorf("could not find label %s on image", style.Symbol(platform.LifecycleMetadataLabel))
	}
	var runImageMD builder.RunImageMetadata
	if md.RunImage.Image != "" {
		runImageMD = builder.RunImageMetadata{
			Image:   md.RunImage.Image,
			Mirrors: md.RunImage.Mirrors,
		}
	} else if md.Stack != nil {
		runImageMD = builder.RunImageMetadata{
			Image:   md.Stack.RunImage.Image,
			Mirrors: md.Stack.RunImage.Mirrors,
		}
	}

	target := &dist.Target{OS: appOS, Arch: appArch}
	fetchOptions := image.FetchOptions{
		Daemon:             !opts.Publish,
		PullPolicy:         opts.PullPolicy,
		Target:             target,
		InsecureRegistries: opts.InsecureRegistries,
	}

	runImageName := c.resolveRunImage(
		opts.RunImage,
		imageRef.Context().RegistryStr(),
		"",
		runImageMD,
		opts.AdditionalMirrors,
		opts.Publish,
		fetchOptions,
	)

	if runImageName == "" {
		return errors.New("run image must be specified")
	}

	baseImage, err := c.imageFetcher.Fetch(ctx, runImageName, fetchOptions)
	if err != nil {
		return err
	}

	for _, reg := range opts.InsecureRegistries {
		flags = append(flags, "-insecure-registry", reg)
	}

	c.logger.Infof("Rebasing %s on run image %s", style.Symbol(appImage.Name()), style.Symbol(baseImage.Name()))
	rebaser := &phase.Rebaser{Logger: c.logger, PlatformAPI: build.SupportedPlatformAPIVersions.Latest(), Force: opts.Force}
	report, err := rebaser.Rebase(appImage, baseImage, opts.RepoName, nil)
	if err != nil {
		return err
	}

	appImageIdentifier, err := appImage.Identifier()
	if err != nil {
		return err
	}

	c.logger.Infof("Rebased Image: %s", style.Symbol(appImageIdentifier.String()))

	if opts.ReportDestinationDir != "" {
		reportPath := filepath.Join(opts.ReportDestinationDir, "report.toml")
		reportFile, err := os.OpenFile(reportPath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			c.logger.Warnf("unable to open %s for writing rebase report", reportPath)
			return err
		}

		defer reportFile.Close()
		err = toml.NewEncoder(reportFile).Encode(report)
		if err != nil {
			c.logger.Warnf("unable to write rebase report to %s", reportPath)
			return err
		}
	}
	return nil
}
