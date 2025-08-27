package client

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/buildpacks/pack/internal/name"

	"github.com/pkg/errors"

	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/internal/layer"
	"github.com/buildpacks/pack/internal/paths"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

const (
	// Packaging indicator that format of inputs/outputs will be an OCI image on the registry.
	FormatImage = "image"

	// Packaging indicator that format of output will be a file on the host filesystem.
	FormatFile = "file"

	// CNBExtension is the file extension for a cloud native buildpack tar archive
	CNBExtension = ".cnb"
)

// PackageBuildpackOptions is a configuration object used to define
// the behavior of PackageBuildpack.
type PackageBuildpackOptions struct {
	// The base director to resolve relative assest from
	RelativeBaseDir string

	// The name of the output buildpack artifact.
	Name string

	// Type of output format, The options are the either the const FormatImage, or FormatFile.
	Format string

	// Defines the Buildpacks configuration.
	Config pubbldpkg.Config

	// Push resulting builder image up to a registry
	// specified in the Name variable.
	Publish bool

	// Append [os]-[arch] suffix to the image tag when publishing a multi-arch to a registry
	// Requires Publish to be true
	AppendImageNameSuffix bool

	// Strategy for updating images before packaging.
	PullPolicy image.PullPolicy

	// Name of the buildpack registry. Used to
	// add buildpacks to a package.
	Registry string

	// Flatten layers
	Flatten bool

	// List of buildpack images to exclude from being flattened.
	FlattenExclude []string

	// Map of labels to add to the Buildpack
	Labels map[string]string

	// Target platforms to build packages for
	Targets []dist.Target
}

// PackageBuildpack packages buildpack(s) into either an image or file.
func (c *Client) PackageBuildpack(ctx context.Context, opts PackageBuildpackOptions) error {
	if opts.Format == "" {
		opts.Format = FormatImage
	}

	targets, err := c.processPackageBuildpackTargets(ctx, opts)
	if err != nil {
		return err
	}
	multiArch := len(targets) > 1 && (opts.Publish || opts.Format == FormatFile)

	var digests []string
	targets = dist.ExpandTargetsDistributions(targets...)
	for _, target := range targets {
		digest, err := c.packageBuildpackTarget(ctx, opts, target, multiArch)
		if err != nil {
			return err
		}
		digests = append(digests, digest)
	}

	if opts.Publish && len(digests) > 1 {
		// Image Index must be created only when we pushed to registry
		return c.CreateManifest(ctx, CreateManifestOptions{
			IndexRepoName: opts.Name,
			RepoNames:     digests,
			Publish:       true,
		})
	}

	return nil
}

func (c *Client) packageBuildpackTarget(ctx context.Context, opts PackageBuildpackOptions, target dist.Target, multiArch bool) (string, error) {
	var digest string
	if target.OS == "windows" && !c.experimental {
		return "", NewExperimentError("Windows buildpackage support is currently experimental.")
	}

	err := c.validateOSPlatform(ctx, target.OS, opts.Publish, opts.Format)
	if err != nil {
		return digest, err
	}

	writerFactory, err := layer.NewWriterFactory(target.OS)
	if err != nil {
		return digest, errors.Wrap(err, "creating layer writer factory")
	}

	var packageBuilderOpts []buildpack.PackageBuilderOption
	if opts.Flatten {
		packageBuilderOpts = append(packageBuilderOpts, buildpack.DoNotFlatten(opts.FlattenExclude),
			buildpack.WithLayerWriterFactory(writerFactory), buildpack.WithLogger(c.logger))
	}
	packageBuilder := buildpack.NewBuilder(c.imageFactory, packageBuilderOpts...)

	bpURI := opts.Config.Buildpack.URI
	if bpURI == "" {
		return digest, errors.New("buildpack URI must be provided")
	}

	if ok, platformRootFolder := buildpack.PlatformRootFolder(bpURI, target); ok {
		bpURI = platformRootFolder
	}

	mainBlob, err := c.downloadBuildpackFromURI(ctx, bpURI, opts.RelativeBaseDir)
	if err != nil {
		return digest, err
	}

	bp, err := buildpack.FromBuildpackRootBlob(mainBlob, writerFactory, c.logger)
	if err != nil {
		return digest, errors.Wrapf(err, "creating buildpack from %s", style.Symbol(bpURI))
	}

	packageBuilder.SetBuildpack(bp)

	platform := target.ValuesAsPlatform()

	for _, dep := range opts.Config.Dependencies {
		if multiArch {
			locatorType, err := buildpack.GetLocatorType(dep.URI, opts.RelativeBaseDir, []dist.ModuleInfo{})
			if err != nil {
				return digest, err
			}
			if locatorType == buildpack.URILocator {
				// When building a composite multi-platform buildpack all the dependencies must be pushed to a registry
				return digest, errors.New(fmt.Sprintf("uri %s is not allowed when creating a composite multi-platform buildpack; push your dependencies to a registry and use 'docker://<image>' instead", style.Symbol(dep.URI)))
			}
		}

		c.logger.Debugf("Downloading buildpack dependency for platform %s", platform)
		mainBP, deps, err := c.buildpackDownloader.Download(ctx, dep.URI, buildpack.DownloadOptions{
			RegistryName:    opts.Registry,
			RelativeBaseDir: opts.RelativeBaseDir,
			ImageName:       dep.ImageName,
			Daemon:          !opts.Publish,
			PullPolicy:      opts.PullPolicy,
			Target:          &target,
		})
		if err != nil {
			return digest, errors.Wrapf(err, "packaging dependencies (uri=%s,image=%s)", style.Symbol(dep.URI), style.Symbol(dep.ImageName))
		}

		packageBuilder.AddDependencies(mainBP, deps)
	}

	switch opts.Format {
	case FormatFile:
		name := opts.Name
		if multiArch {
			extension := filepath.Ext(name)
			origFileName := name[:len(name)-len(filepath.Ext(name))]
			if target.Arch != "" {
				name = fmt.Sprintf("%s-%s-%s%s", origFileName, target.OS, target.Arch, extension)
			} else {
				name = fmt.Sprintf("%s-%s%s", origFileName, target.OS, extension)
			}
		}
		err = packageBuilder.SaveAsFile(name, target, opts.Labels)
		if err != nil {
			return digest, err
		}
	case FormatImage:
		packageName := opts.Name
		if multiArch && opts.AppendImageNameSuffix {
			packageName, err = name.AppendSuffix(packageName, target)
			if err != nil {
				return "", errors.Wrap(err, "invalid image name")
			}
		}
		img, err := packageBuilder.SaveAsImage(packageName, opts.Publish, target, opts.Labels)
		if err != nil {
			return digest, errors.Wrapf(err, "saving image")
		}
		if multiArch {
			// We need to keep the identifier to create the image index
			id, err := img.Identifier()
			if err != nil {
				return digest, errors.Wrapf(err, "determining image manifest digest")
			}
			digest = id.String()
		}
	default:
		return digest, errors.Errorf("unknown format: %s", style.Symbol(opts.Format))
	}
	return digest, nil
}

func (c *Client) downloadBuildpackFromURI(ctx context.Context, uri, relativeBaseDir string) (blob.Blob, error) {
	absPath, err := paths.FilePathToURI(uri, relativeBaseDir)
	if err != nil {
		return nil, errors.Wrapf(err, "making absolute: %s", style.Symbol(uri))
	}
	uri = absPath

	c.logger.Debugf("Downloading buildpack from URI: %s", style.Symbol(uri))
	blob, err := c.downloader.Download(ctx, uri)
	if err != nil {
		return nil, errors.Wrapf(err, "downloading buildpack from %s", style.Symbol(uri))
	}

	return blob, nil
}

func (c *Client) processPackageBuildpackTargets(ctx context.Context, opts PackageBuildpackOptions) ([]dist.Target, error) {
	var targets []dist.Target
	if len(opts.Targets) > 0 {
		// when exporting to the daemon, we need to select just one target
		if !opts.Publish && opts.Format == FormatImage {
			daemonTarget, err := c.daemonTarget(ctx, opts.Targets)
			if err != nil {
				return targets, err
			}
			targets = append(targets, daemonTarget)
		} else {
			targets = opts.Targets
		}
	} else {
		targets = append(targets, dist.Target{OS: opts.Config.Platform.OS})
	}
	return targets, nil
}

func (c *Client) validateOSPlatform(ctx context.Context, os string, publish bool, format string) error {
	if publish || format == FormatFile {
		return nil
	}

	info, err := c.docker.Info(ctx)
	if err != nil {
		return err
	}

	if info.OSType != os {
		return errors.Errorf("invalid %s specified: DOCKER_OS is %s", style.Symbol("platform.os"), style.Symbol(info.OSType))
	}

	return nil
}

// daemonTarget returns a target that matches with the given daemon os/arch
func (c *Client) daemonTarget(ctx context.Context, targets []dist.Target) (dist.Target, error) {
	info, err := c.docker.ServerVersion(ctx)
	if err != nil {
		return dist.Target{}, err
	}

	for _, t := range targets {
		if t.Arch != "" && t.OS == info.Os && t.Arch == info.Arch {
			return t, nil
		} else if t.Arch == "" && t.OS == info.Os {
			return t, nil
		}
	}
	return dist.Target{}, errors.Errorf("could not find a target that matches daemon os=%s and architecture=%s", info.Os, info.Arch)
}
