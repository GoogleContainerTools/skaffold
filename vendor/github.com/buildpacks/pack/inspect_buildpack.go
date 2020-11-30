package pack

import (
	"context"
	"fmt"
	"sort"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/buildpacks/pack/internal/style"

	"github.com/buildpacks/pack/config"
	"github.com/buildpacks/pack/internal/buildpack"
	"github.com/buildpacks/pack/internal/buildpackage"
	"github.com/buildpacks/pack/internal/dist"
)

type BuildpackInfo struct {
	BuildpackMetadata buildpackage.Metadata
	Buildpacks        []dist.BuildpackInfo
	Order             dist.Order
	BuildpackLayers   dist.BuildpackLayers
	Location          buildpack.LocatorType
}

type InspectBuildpackOptions struct {
	BuildpackName string
	Daemon        bool
	Registry      string
}

type ImgWrapper struct {
	v1.ImageConfig
}

func (iw ImgWrapper) Label(name string) (string, error) {
	return iw.Labels[name], nil
}

// Must return an BuildpackNotFoundError
func (c *Client) InspectBuildpack(opts InspectBuildpackOptions) (*BuildpackInfo, error) {
	locatorType, err := buildpack.GetLocatorType(opts.BuildpackName, []dist.BuildpackInfo{})
	if err != nil {
		return nil, err
	}
	var layersMd dist.BuildpackLayers
	var buildpackMd buildpackage.Metadata

	switch locatorType {
	case buildpack.RegistryLocator:
		buildpackMd, layersMd, err = metadataFromRegistry(c, opts.BuildpackName, opts.Registry)
	case buildpack.PackageLocator:
		buildpackMd, layersMd, err = metadataFromImage(c, opts.BuildpackName, opts.Daemon)
	case buildpack.URILocator:
		buildpackMd, layersMd, err = metadataFromArchive(c.downloader, opts.BuildpackName)
	default:
		return nil, fmt.Errorf("unable to handle locator %q: for buildpack %q", locatorType, opts.BuildpackName)
	}
	if err != nil {
		return nil, err
	}

	return &BuildpackInfo{
		BuildpackMetadata: buildpackMd,
		BuildpackLayers:   layersMd,
		Order:             extractOrder(buildpackMd),
		Buildpacks:        extractBuildpacks(layersMd),
		Location:          locatorType,
	}, nil
}

func metadataFromRegistry(client *Client, name, registry string) (buildpackMd buildpackage.Metadata, layersMd dist.BuildpackLayers, err error) {
	registryCache, err := client.getRegistry(client.logger, registry)
	if err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, fmt.Errorf("invalid registry %s: %q", registry, err)
	}

	registryBp, err := registryCache.LocateBuildpack(name)
	if err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, fmt.Errorf("unable to find %s in registry: %q", style.Symbol(name), err)
	}
	buildpackMd, layersMd, err = metadataFromImage(client, registryBp.Address, false)
	if err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, fmt.Errorf("error pulling registry specified image: %s", err)
	}
	return buildpackMd, layersMd, nil
}

func metadataFromArchive(downloader Downloader, path string) (buildpackMd buildpackage.Metadata, layersMd dist.BuildpackLayers, err error) {
	// open archive, read it as an image and get all metadata.
	// This looks like a prime candidate to be added to imgutil.

	imgBlob, err := downloader.Download(context.Background(), path)
	if err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, fmt.Errorf("unable to download archive: %q", err)
		//return buildpackMd, layersMd, err
	}

	config, err := buildpackage.ConfigFromOCILayoutBlob(imgBlob)
	if err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, fmt.Errorf("unable to fetch config from buildpack blob: %q", err)
	}
	wrapper := ImgWrapper{config}

	if _, err := dist.GetLabel(wrapper, dist.BuildpackLayersLabel, &layersMd); err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, err
	}

	if _, err := dist.GetLabel(wrapper, buildpackage.MetadataLabel, &buildpackMd); err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, err
	}
	return buildpackMd, layersMd, nil
}

func metadataFromImage(client *Client, name string, daemon bool) (buildpackMd buildpackage.Metadata, layersMd dist.BuildpackLayers, err error) {
	img, err := client.imageFetcher.Fetch(context.Background(), name, daemon, config.PullNever)
	if err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, err
	}
	if _, err := dist.GetLabel(img, dist.BuildpackLayersLabel, &layersMd); err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, fmt.Errorf("unable to get image label %s: %q", dist.BuildpackLayersLabel, err)
	}

	if _, err := dist.GetLabel(img, buildpackage.MetadataLabel, &buildpackMd); err != nil {
		return buildpackage.Metadata{}, dist.BuildpackLayers{}, fmt.Errorf("unable to get image label %s: %q", buildpackage.MetadataLabel, err)
	}
	return buildpackMd, layersMd, nil
}

func extractOrder(buildpackMd buildpackage.Metadata) dist.Order {
	return dist.Order{
		{
			Group: []dist.BuildpackRef{
				{
					BuildpackInfo: buildpackMd.BuildpackInfo,
				},
			},
		},
	}
}

func extractBuildpacks(layersMd dist.BuildpackLayers) []dist.BuildpackInfo {
	result := []dist.BuildpackInfo{}
	buildpackSet := map[dist.BuildpackInfo]bool{}

	for buildpackID, buildpackMap := range layersMd {
		for version, layerInfo := range buildpackMap {
			bp := dist.BuildpackInfo{
				ID:       buildpackID,
				Version:  version,
				Homepage: layerInfo.Homepage,
			}
			buildpackSet[bp] = true
		}
	}

	for currentBuildpack := range buildpackSet {
		result = append(result, currentBuildpack)
	}

	sort.Slice(result, func(i int, j int) bool {
		switch {
		case result[i].ID < result[j].ID:
			return true
		case result[i].ID == result[j].ID:
			return result[i].Version < result[j].Version
		default:
			return false
		}
	})
	return result
}
