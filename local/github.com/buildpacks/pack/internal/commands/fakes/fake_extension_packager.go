package fakes

import (
	"context"

	"github.com/buildpacks/pack/pkg/client"
)

func (c *FakeBuildpackPackager) PackageExtension(ctx context.Context, opts client.PackageBuildpackOptions) error {
	c.CreateCalledWithOptions = opts

	return nil
}
