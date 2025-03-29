package fakes

import (
	"context"

	"github.com/buildpacks/pack/internal/build"
)

type FakeLifecycle struct {
	Opts build.LifecycleOptions
}

func (f *FakeLifecycle) Execute(ctx context.Context, opts build.LifecycleOptions) error {
	f.Opts = opts
	return nil
}
