package client

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/layer"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
)

// PackageExtension packages extension(s) into either an image or file.
func (c *Client) PackageExtension(ctx context.Context, opts PackageBuildpackOptions) error {
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
		digest, err := c.packageExtensionTarget(ctx, opts, target, multiArch)
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

func (c *Client) packageExtensionTarget(ctx context.Context, opts PackageBuildpackOptions, target dist.Target, multiArch bool) (string, error) {
	var digest string
	if target.OS == "windows" && !c.experimental {
		return "", NewExperimentError("Windows extensionpackage support is currently experimental.")
	}

	err := c.validateOSPlatform(ctx, target.OS, opts.Publish, opts.Format)
	if err != nil {
		return digest, err
	}

	writerFactory, err := layer.NewWriterFactory(target.OS)
	if err != nil {
		return digest, errors.Wrap(err, "creating layer writer factory")
	}

	packageBuilder := buildpack.NewBuilder(c.imageFactory)

	exURI := opts.Config.Extension.URI
	if exURI == "" {
		return digest, errors.New("extension URI must be provided")
	}

	if ok, platformRootFolder := buildpack.PlatformRootFolder(exURI, target); ok {
		exURI = platformRootFolder
	}

	mainBlob, err := c.downloadBuildpackFromURI(ctx, exURI, opts.RelativeBaseDir)
	if err != nil {
		return digest, err
	}

	ex, err := buildpack.FromExtensionRootBlob(mainBlob, writerFactory, c.logger)
	if err != nil {
		return digest, errors.Wrapf(err, "creating extension from %s", style.Symbol(exURI))
	}

	packageBuilder.SetExtension(ex)

	switch opts.Format {
	case FormatFile:
		name := opts.Name
		if multiArch {
			fileExtension := filepath.Ext(name)
			origFileName := name[:len(name)-len(filepath.Ext(name))]
			if target.Arch != "" {
				name = fmt.Sprintf("%s-%s-%s%s", origFileName, target.OS, target.Arch, fileExtension)
			} else {
				name = fmt.Sprintf("%s-%s%s", origFileName, target.OS, fileExtension)
			}
		}
		err = packageBuilder.SaveAsFile(name, target, opts.Labels)
		if err != nil {
			return digest, err
		}
	case FormatImage:
		img, err := packageBuilder.SaveAsImage(opts.Name, opts.Publish, target, opts.Labels)
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
