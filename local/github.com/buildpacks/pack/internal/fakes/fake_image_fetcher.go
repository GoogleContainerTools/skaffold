package fakes

import (
	"context"

	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/image"
)

type FetchArgs struct {
	Daemon       bool
	PullPolicy   image.PullPolicy
	LayoutOption image.LayoutOption
	Target       *dist.Target
}

type FakeImageFetcher struct {
	LocalImages  map[string]imgutil.Image
	RemoteImages map[string]imgutil.Image
	FetchCalls   map[string]*FetchArgs
}

func NewFakeImageFetcher() *FakeImageFetcher {
	return &FakeImageFetcher{
		LocalImages:  map[string]imgutil.Image{},
		RemoteImages: map[string]imgutil.Image{},
		FetchCalls:   map[string]*FetchArgs{},
	}
}

func (f *FakeImageFetcher) Fetch(ctx context.Context, name string, options image.FetchOptions) (imgutil.Image, error) {
	f.FetchCalls[name] = &FetchArgs{Daemon: options.Daemon, PullPolicy: options.PullPolicy, Target: options.Target, LayoutOption: options.LayoutOption}

	ri, remoteFound := f.RemoteImages[name]

	if options.Daemon {
		li, localFound := f.LocalImages[name]

		if shouldPull(localFound, remoteFound, options.PullPolicy) {
			f.LocalImages[name] = ri
			li = ri
		}
		if !localFound {
			return nil, errors.Wrapf(image.ErrNotFound, "image '%s' does not exist on the daemon", name)
		}
		return li, nil
	}

	if !remoteFound {
		return nil, errors.Wrapf(image.ErrNotFound, "image '%s' does not exist in registry", name)
	}

	return ri, nil
}

func (f *FakeImageFetcher) CheckReadAccess(_ string, _ image.FetchOptions) bool {
	return true
}

func shouldPull(localFound, remoteFound bool, policy image.PullPolicy) bool {
	if remoteFound && !localFound && policy == image.PullIfNotPresent {
		return true
	}

	return remoteFound && policy == image.PullAlways
}
