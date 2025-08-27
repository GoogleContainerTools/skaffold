package client

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/buildpacks/pack/internal/name"

	"github.com/Masterminds/semver"
	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	pubbldr "github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

// CreateBuilderOptions is a configuration object used to change the behavior of
// CreateBuilder.
type CreateBuilderOptions struct {
	// The base directory to use to resolve relative assets
	RelativeBaseDir string

	// Name of the builder.
	BuilderName string

	// BuildConfigEnv for Builder
	BuildConfigEnv map[string]string

	// Map of labels to add to the Buildpack
	Labels map[string]string

	// Configuration that defines the functionality a builder provides.
	Config pubbldr.Config

	// Skip building image locally, directly publish to a registry.
	// Requires BuilderName to be a valid registry location.
	Publish bool

	// Append [os]-[arch] suffix to the image tag when publishing a multi-arch to a registry
	// Requires Publish to be true
	AppendImageNameSuffix bool

	// Buildpack registry name. Defines where all registry buildpacks will be pulled from.
	Registry string

	// Strategy for updating images before a build.
	PullPolicy image.PullPolicy

	// List of modules to be flattened
	Flatten buildpack.FlattenModuleInfos

	// Target platforms to build builder images for
	Targets []dist.Target
}

// CreateBuilder creates and saves a builder image to a registry with the provided options.
// If any configuration is invalid, it will error and exit without creating any images.
func (c *Client) CreateBuilder(ctx context.Context, opts CreateBuilderOptions) error {
	targets, err := c.processBuilderCreateTargets(ctx, opts)
	if err != nil {
		return err
	}

	if len(targets) == 0 {
		_, err = c.createBuilderTarget(ctx, opts, nil, false)
		if err != nil {
			return err
		}
	} else {
		var digests []string
		multiArch := len(targets) > 1 && opts.Publish

		for _, target := range targets {
			digest, err := c.createBuilderTarget(ctx, opts, &target, multiArch)
			if err != nil {
				return err
			}
			digests = append(digests, digest)
		}

		if multiArch && len(digests) > 1 {
			return c.CreateManifest(ctx, CreateManifestOptions{
				IndexRepoName: opts.BuilderName,
				RepoNames:     digests,
				Publish:       true,
			})
		}
	}

	return nil
}

func (c *Client) createBuilderTarget(ctx context.Context, opts CreateBuilderOptions, target *dist.Target, multiArch bool) (string, error) {
	if err := c.validateConfig(ctx, opts, target); err != nil {
		return "", err
	}

	bldr, err := c.createBaseBuilder(ctx, opts, target, multiArch)
	if err != nil {
		return "", errors.Wrap(err, "failed to create builder")
	}

	if err := c.addBuildpacksToBuilder(ctx, opts, bldr); err != nil {
		return "", errors.Wrap(err, "failed to add buildpacks to builder")
	}

	if err := c.addExtensionsToBuilder(ctx, opts, bldr); err != nil {
		return "", errors.Wrap(err, "failed to add extensions to builder")
	}

	bldr.SetOrder(opts.Config.Order)
	bldr.SetOrderExtensions(opts.Config.OrderExtensions)

	if opts.Config.Stack.ID != "" {
		bldr.SetStack(opts.Config.Stack)
	}
	bldr.SetRunImage(opts.Config.Run)
	bldr.SetBuildConfigEnv(opts.BuildConfigEnv)

	err = bldr.Save(c.logger, builder.CreatorMetadata{Version: c.version})
	if err != nil {
		return "", err
	}

	if multiArch {
		// We need to keep the identifier to create the image index
		id, err := bldr.Image().Identifier()
		if err != nil {
			return "", errors.Wrapf(err, "determining image manifest digest")
		}
		return id.String(), nil
	}
	return "", nil
}

func (c *Client) validateConfig(ctx context.Context, opts CreateBuilderOptions, target *dist.Target) error {
	if err := pubbldr.ValidateConfig(opts.Config); err != nil {
		return errors.Wrap(err, "invalid builder config")
	}

	if err := c.validateRunImageConfig(ctx, opts, target); err != nil {
		return errors.Wrap(err, "invalid run image config")
	}

	return nil
}

func (c *Client) validateRunImageConfig(ctx context.Context, opts CreateBuilderOptions, target *dist.Target) error {
	var runImages []imgutil.Image
	for _, r := range opts.Config.Run.Images {
		for _, i := range append([]string{r.Image}, r.Mirrors...) {
			if !opts.Publish {
				img, err := c.imageFetcher.Fetch(ctx, i, image.FetchOptions{Daemon: true, PullPolicy: opts.PullPolicy, Target: target})
				if err != nil {
					if errors.Cause(err) != image.ErrNotFound {
						return errors.Wrap(err, "failed to fetch image")
					}
				} else {
					runImages = append(runImages, img)
					continue
				}
			}

			img, err := c.imageFetcher.Fetch(ctx, i, image.FetchOptions{Daemon: false, PullPolicy: opts.PullPolicy, Target: target})
			if err != nil {
				if errors.Cause(err) != image.ErrNotFound {
					return errors.Wrap(err, "failed to fetch image")
				}
				c.logger.Warnf("run image %s is not accessible", style.Symbol(i))
			} else {
				runImages = append(runImages, img)
			}
		}
	}

	for _, img := range runImages {
		if opts.Config.Stack.ID != "" {
			stackID, err := img.Label("io.buildpacks.stack.id")
			if err != nil {
				return errors.Wrap(err, "failed to label image")
			}

			if stackID != opts.Config.Stack.ID {
				return fmt.Errorf(
					"stack %s from builder config is incompatible with stack %s from run image %s",
					style.Symbol(opts.Config.Stack.ID),
					style.Symbol(stackID),
					style.Symbol(img.Name()),
				)
			}
		}
	}

	return nil
}

func (c *Client) createBaseBuilder(ctx context.Context, opts CreateBuilderOptions, target *dist.Target, multiArch bool) (*builder.Builder, error) {
	baseImage, err := c.imageFetcher.Fetch(ctx, opts.Config.Build.Image, image.FetchOptions{Daemon: !opts.Publish, PullPolicy: opts.PullPolicy, Target: target})
	if err != nil {
		return nil, errors.Wrap(err, "fetch build image")
	}

	c.logger.Debugf("Creating builder %s from build-image %s", style.Symbol(opts.BuilderName), style.Symbol(baseImage.Name()))

	var builderOpts []builder.BuilderOption
	if opts.Flatten != nil && len(opts.Flatten.FlattenModules()) > 0 {
		builderOpts = append(builderOpts, builder.WithFlattened(opts.Flatten))
	}
	if len(opts.Labels) > 0 {
		builderOpts = append(builderOpts, builder.WithLabels(opts.Labels))
	}

	builderName := opts.BuilderName
	if multiArch && opts.AppendImageNameSuffix {
		builderName, err = name.AppendSuffix(builderName, *target)
		if err != nil {
			return nil, errors.Wrap(err, "invalid image name")
		}
	}

	bldr, err := builder.New(baseImage, builderName, builderOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "invalid build-image")
	}

	architecture, err := baseImage.Architecture()
	if err != nil {
		return nil, errors.Wrap(err, "lookup image Architecture")
	}

	os, err := baseImage.OS()
	if err != nil {
		return nil, errors.Wrap(err, "lookup image OS")
	}

	if os == "windows" && !c.experimental {
		return nil, NewExperimentError("Windows containers support is currently experimental.")
	}

	bldr.SetDescription(opts.Config.Description)

	if opts.Config.Stack.ID != "" && bldr.StackID != opts.Config.Stack.ID {
		return nil, fmt.Errorf(
			"stack %s from builder config is incompatible with stack %s from build image",
			style.Symbol(opts.Config.Stack.ID),
			style.Symbol(bldr.StackID),
		)
	}

	lifecycle, err := c.fetchLifecycle(ctx, opts.Config.Lifecycle, opts.RelativeBaseDir, os, architecture)
	if err != nil {
		return nil, errors.Wrap(err, "fetch lifecycle")
	}

	bldr.SetLifecycle(lifecycle)
	bldr.SetBuildConfigEnv(opts.BuildConfigEnv)

	return bldr, nil
}

func (c *Client) fetchLifecycle(ctx context.Context, config pubbldr.LifecycleConfig, relativeBaseDir, os string, architecture string) (builder.Lifecycle, error) {
	if config.Version != "" && config.URI != "" {
		return nil, errors.Errorf(
			"%s can only declare %s or %s, not both",
			style.Symbol("lifecycle"), style.Symbol("version"), style.Symbol("uri"),
		)
	}

	var uri string
	var err error
	switch {
	case config.Version != "":
		v, err := semver.NewVersion(config.Version)
		if err != nil {
			return nil, errors.Wrapf(err, "%s must be a valid semver", style.Symbol("lifecycle.version"))
		}

		uri = c.uriFromLifecycleVersion(*v, os, architecture)
	case config.URI != "":
		uri, err = paths.FilePathToURI(config.URI, relativeBaseDir)
		if err != nil {
			return nil, err
		}
	default:
		uri = c.uriFromLifecycleVersion(*semver.MustParse(builder.DefaultLifecycleVersion), os, architecture)
	}

	blob, err := c.downloader.Download(ctx, uri)
	if err != nil {
		return nil, errors.Wrap(err, "downloading lifecycle")
	}

	lifecycle, err := builder.NewLifecycle(blob)
	if err != nil {
		return nil, errors.Wrap(err, "invalid lifecycle")
	}

	return lifecycle, nil
}

func (c *Client) addBuildpacksToBuilder(ctx context.Context, opts CreateBuilderOptions, bldr *builder.Builder) error {
	for _, b := range opts.Config.Buildpacks {
		if err := c.addConfig(ctx, buildpack.KindBuildpack, b, opts, bldr); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) addExtensionsToBuilder(ctx context.Context, opts CreateBuilderOptions, bldr *builder.Builder) error {
	for _, e := range opts.Config.Extensions {
		if err := c.addConfig(ctx, buildpack.KindExtension, e, opts, bldr); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) addConfig(ctx context.Context, kind string, config pubbldr.ModuleConfig, opts CreateBuilderOptions, bldr *builder.Builder) error {
	c.logger.Debugf("Looking up %s %s", kind, style.Symbol(config.DisplayString()))

	builderOS, err := bldr.Image().OS()
	if err != nil {
		return errors.Wrapf(err, "getting builder OS")
	}
	builderArch, err := bldr.Image().Architecture()
	if err != nil {
		return errors.Wrapf(err, "getting builder architecture")
	}

	target := &dist.Target{OS: builderOS, Arch: builderArch}
	c.logger.Debugf("Downloading buildpack for platform: %s", target.ValuesAsPlatform())

	mainBP, depBPs, err := c.buildpackDownloader.Download(ctx, config.URI, buildpack.DownloadOptions{
		Daemon:          !opts.Publish,
		ImageName:       config.ImageName,
		ModuleKind:      kind,
		PullPolicy:      opts.PullPolicy,
		RegistryName:    opts.Registry,
		RelativeBaseDir: opts.RelativeBaseDir,
		Target:          target,
	})
	if err != nil {
		return errors.Wrapf(err, "downloading %s", kind)
	}
	err = validateModule(kind, mainBP, config.URI, config.ID, config.Version)
	if err != nil {
		return errors.Wrapf(err, "invalid %s", kind)
	}

	bpDesc := mainBP.Descriptor()
	for _, deprecatedAPI := range bldr.LifecycleDescriptor().APIs.Buildpack.Deprecated {
		if deprecatedAPI.Equal(bpDesc.API()) {
			c.logger.Warnf(
				"%s %s is using deprecated Buildpacks API version %s",
				cases.Title(language.AmericanEnglish).String(kind),
				style.Symbol(bpDesc.Info().FullName()),
				style.Symbol(bpDesc.API().String()),
			)
			break
		}
	}

	// Fixes 1453
	sort.Slice(depBPs, func(i, j int) bool {
		compareID := strings.Compare(depBPs[i].Descriptor().Info().ID, depBPs[j].Descriptor().Info().ID)
		if compareID == 0 {
			return strings.Compare(depBPs[i].Descriptor().Info().Version, depBPs[j].Descriptor().Info().Version) <= 0
		}
		return compareID < 0
	})

	switch kind {
	case buildpack.KindBuildpack:
		bldr.AddBuildpacks(mainBP, depBPs)
	case buildpack.KindExtension:
		// Extensions can't be composite
		bldr.AddExtension(mainBP)
	default:
		return fmt.Errorf("unknown module kind: %s", kind)
	}
	return nil
}

func (c *Client) processBuilderCreateTargets(ctx context.Context, opts CreateBuilderOptions) ([]dist.Target, error) {
	var targets []dist.Target

	if len(opts.Targets) > 0 {
		if opts.Publish {
			targets = opts.Targets
		} else {
			// find a target that matches the daemon
			daemonTarget, err := c.daemonTarget(ctx, opts.Targets)
			if err != nil {
				return targets, err
			}
			targets = append(targets, daemonTarget)
		}
	}
	return targets, nil
}

func validateModule(kind string, module buildpack.BuildModule, source, expectedID, expectedVersion string) error {
	info := module.Descriptor().Info()
	if expectedID != "" && info.ID != expectedID {
		return fmt.Errorf(
			"%s from URI %s has ID %s which does not match ID %s from builder config",
			kind,
			style.Symbol(source),
			style.Symbol(info.ID),
			style.Symbol(expectedID),
		)
	}

	if expectedVersion != "" && info.Version != expectedVersion {
		return fmt.Errorf(
			"%s from URI %s has version %s which does not match version %s from builder config",
			kind,
			style.Symbol(source),
			style.Symbol(info.Version),
			style.Symbol(expectedVersion),
		)
	}

	return nil
}

func (c *Client) uriFromLifecycleVersion(version semver.Version, os string, architecture string) string {
	arch := "x86-64"

	if os == "windows" {
		return fmt.Sprintf("https://github.com/buildpacks/lifecycle/releases/download/v%s/lifecycle-v%s+windows.%s.tgz", version.String(), version.String(), arch)
	}

	if builder.SupportedLinuxArchitecture(architecture) {
		arch = architecture
	} else {
		// FIXME: this should probably be an error case in the future, see https://github.com/buildpacks/pack/issues/2163
		c.logger.Warnf("failed to find a lifecycle binary for requested architecture %s, defaulting to %s", style.Symbol(architecture), style.Symbol(arch))
	}

	return fmt.Sprintf("https://github.com/buildpacks/lifecycle/releases/download/v%s/lifecycle-v%s+linux.%s.tgz", version.String(), version.String(), arch)
}
