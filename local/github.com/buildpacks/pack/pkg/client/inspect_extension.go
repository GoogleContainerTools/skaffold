package client

import (
	"context"
	"fmt"

	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

type ExtensionInfo struct {
	Extension dist.ModuleInfo
	Location  buildpack.LocatorType
}

type InspectExtensionOptions struct {
	ExtensionName string
	Daemon        bool
}

func (c *Client) InspectExtension(opts InspectExtensionOptions) (*ExtensionInfo, error) {
	locatorType, err := buildpack.GetLocatorType(opts.ExtensionName, "", []dist.ModuleInfo{})
	if err != nil {
		return nil, err
	}
	var layerMd dist.ModuleLayers

	layerMd, err = metadataOfExtensionFromImage(c, opts.ExtensionName, opts.Daemon)

	if err != nil {
		return nil, err
	}

	if len(layerMd) != 1 {
		return nil, fmt.Errorf("expected 1 extension, got %d", len(layerMd))
	}

	return &ExtensionInfo{
		Extension: extractExtension(layerMd),
		Location:  locatorType,
	}, nil
}

func metadataOfExtensionFromImage(client *Client, name string, daemon bool) (layerMd dist.ModuleLayers, err error) {
	imageName := buildpack.ParsePackageLocator(name)
	img, err := client.imageFetcher.Fetch(context.Background(), imageName, image.FetchOptions{Daemon: daemon, PullPolicy: image.PullNever})
	if err != nil {
		return dist.ModuleLayers{}, err
	}

	if _, err := dist.GetLabel(img, dist.ExtensionLayersLabel, &layerMd); err != nil {
		return dist.ModuleLayers{}, fmt.Errorf("unable to get image label %s: %q", dist.ExtensionLayersLabel, err)
	}

	return layerMd, nil
}

func extractExtension(layerMd dist.ModuleLayers) dist.ModuleInfo {
	result := dist.ModuleInfo{}
	for extensionID, extensionMap := range layerMd {
		for version, layerInfo := range extensionMap {
			ex := dist.ModuleInfo{
				ID:       extensionID,
				Name:     layerInfo.Name,
				Version:  version,
				Homepage: layerInfo.Homepage,
			}
			result = ex
		}
	}
	return result
}
