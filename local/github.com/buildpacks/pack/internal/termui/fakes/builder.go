package fakes

import (
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/dist"
)

type Builder struct {
	baseImageName       string
	buildpacks          []dist.ModuleInfo
	lifecycleDescriptor builder.LifecycleDescriptor
	stack               builder.StackMetadata
}

func NewBuilder(baseImageName string, buildpacks []dist.ModuleInfo, lifecycleDescriptor builder.LifecycleDescriptor, stack builder.StackMetadata) *Builder {
	return &Builder{
		baseImageName:       baseImageName,
		buildpacks:          buildpacks,
		lifecycleDescriptor: lifecycleDescriptor,
		stack:               stack,
	}
}

func (b *Builder) BaseImageName() string {
	return b.baseImageName
}

func (b *Builder) Buildpacks() []dist.ModuleInfo {
	return b.buildpacks
}

func (b *Builder) LifecycleDescriptor() builder.LifecycleDescriptor {
	return b.lifecycleDescriptor
}

func (b *Builder) Stack() builder.StackMetadata {
	return b.stack
}
