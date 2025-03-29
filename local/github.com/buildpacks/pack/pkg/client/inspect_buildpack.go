package client

import (
	"context"
	"fmt"
	"sort"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

type BuildpackInfo struct {
	BuildpackMetadata buildpack.Metadata
	Buildpacks        []dist.ModuleInfo
	Order             dist.Order
	BuildpackLayers   dist.ModuleLayers
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

func (c *Client) InspectBuildpack(opts InspectBuildpackOptions) (*BuildpackInfo, error) {
	locatorType, err := buildpack.GetLocatorType(opts.BuildpackName, "", []dist.ModuleInfo{})
	if err != nil {
		return nil, err
	}
	var layersMd dist.ModuleLayers
	var buildpackMd buildpack.Metadata

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

func metadataFromRegistry(client *Client, name, registry string) (buildpackMd buildpack.Metadata, layersMd dist.ModuleLayers, err error) {
	registryCache, err := getRegistry(client.logger, registry)
	if err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, fmt.Errorf("invalid registry %s: %q", registry, err)
	}

	registryBp, err := registryCache.LocateBuildpack(name)
	if err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, fmt.Errorf("unable to find %s in registry: %q", style.Symbol(name), err)
	}
	buildpackMd, layersMd, err = metadataFromImage(client, registryBp.Address, false)
	if err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, fmt.Errorf("error pulling registry specified image: %s", err)
	}
	return buildpackMd, layersMd, nil
}

func metadataFromArchive(downloader BlobDownloader, path string) (buildpackMd buildpack.Metadata, layersMd dist.ModuleLayers, err error) {
	imgBlob, err := downloader.Download(context.Background(), path)
	if err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, fmt.Errorf("unable to download archive: %q", err)
	}

	config, err := buildpack.ConfigFromOCILayoutBlob(imgBlob)
	if err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, fmt.Errorf("unable to fetch config from buildpack blob: %q", err)
	}
	wrapper := ImgWrapper{config}

	if _, err := dist.GetLabel(wrapper, dist.BuildpackLayersLabel, &layersMd); err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, err
	}

	if _, err := dist.GetLabel(wrapper, buildpack.MetadataLabel, &buildpackMd); err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, err
	}
	return buildpackMd, layersMd, nil
}

func metadataFromImage(client *Client, name string, daemon bool) (buildpackMd buildpack.Metadata, layersMd dist.ModuleLayers, err error) {
	imageName := buildpack.ParsePackageLocator(name)
	img, err := client.imageFetcher.Fetch(context.Background(), imageName, image.FetchOptions{Daemon: daemon, PullPolicy: image.PullNever})
	if err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, err
	}
	if _, err := dist.GetLabel(img, dist.BuildpackLayersLabel, &layersMd); err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, fmt.Errorf("unable to get image label %s: %q", dist.BuildpackLayersLabel, err)
	}

	if _, err := dist.GetLabel(img, buildpack.MetadataLabel, &buildpackMd); err != nil {
		return buildpack.Metadata{}, dist.ModuleLayers{}, fmt.Errorf("unable to get image label %s: %q", buildpack.MetadataLabel, err)
	}
	return buildpackMd, layersMd, nil
}

func extractOrder(buildpackMd buildpack.Metadata) dist.Order {
	return dist.Order{
		{
			Group: []dist.ModuleRef{
				{
					ModuleInfo: buildpackMd.ModuleInfo,
				},
			},
		},
	}
}

func extractBuildpacks(layersMd dist.ModuleLayers) []dist.ModuleInfo {
	result := []dist.ModuleInfo{}
	buildpackSet := map[*dist.ModuleInfo]bool{}

	for buildpackID, buildpackMap := range layersMd {
		for version, layerInfo := range buildpackMap {
			bp := dist.ModuleInfo{
				ID:       buildpackID,
				Name:     layerInfo.Name,
				Version:  version,
				Homepage: layerInfo.Homepage,
			}
			buildpackSet[&bp] = true
		}
	}

	for currentBuildpack := range buildpackSet {
		result = append(result, *currentBuildpack)
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
