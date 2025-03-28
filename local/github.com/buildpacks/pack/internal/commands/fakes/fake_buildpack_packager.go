package fakes

import (
	"context"

	"github.com/buildpacks/pack/pkg/client"
)

type FakeBuildpackPackager struct {
	CreateCalledWithOptions client.PackageBuildpackOptions
}

func (c *FakeBuildpackPackager) PackageBuildpack(ctx context.Context, opts client.PackageBuildpackOptions) error {
	c.CreateCalledWithOptions = opts

	return nil
}
