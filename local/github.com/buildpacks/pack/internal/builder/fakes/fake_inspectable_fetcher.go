package fakes

import (
	"context"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/image"
)

type FakeInspectableFetcher struct {
	InspectableToReturn *FakeInspectable
	ErrorToReturn       error

	CallCount int

	ReceivedName       string
	ReceivedDaemon     bool
	ReceivedPullPolicy image.PullPolicy
}

func (f *FakeInspectableFetcher) Fetch(ctx context.Context, name string, options image.FetchOptions) (builder.Inspectable, error) {
	f.CallCount++

	f.ReceivedName = name
	f.ReceivedDaemon = options.Daemon
	f.ReceivedPullPolicy = options.PullPolicy

	return f.InspectableToReturn, f.ErrorToReturn
}
