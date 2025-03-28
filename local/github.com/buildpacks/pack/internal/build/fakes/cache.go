package fakes

import (
	"context"

	"github.com/buildpacks/pack/pkg/cache"
)

type FakeCache struct {
	ReturnForType  cache.Type
	ReturnForClear error
	ReturnForName  string

	TypeCallCount  int
	ClearCallCount int
	NameCallCount  int
}

func NewFakeCache() *FakeCache {
	return &FakeCache{}
}

func (f *FakeCache) Type() cache.Type {
	f.TypeCallCount++
	return f.ReturnForType
}

func (f *FakeCache) Clear(ctx context.Context) error {
	f.ClearCallCount++
	return f.ReturnForClear
}
func (f *FakeCache) Name() string {
	f.NameCallCount++
	return f.ReturnForName
}
