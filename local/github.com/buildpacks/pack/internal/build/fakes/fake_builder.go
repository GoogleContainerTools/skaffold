package fakes

import (
	"github.com/Masterminds/semver"
	"github.com/buildpacks/imgutil"
	ifakes "github.com/buildpacks/imgutil/fakes"
	"github.com/buildpacks/lifecycle/api"

	"github.com/buildpacks/pack/internal/build"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/dist"
)

type FakeBuilder struct {
	ReturnForImage               imgutil.Image
	ReturnForUID                 int
	ReturnForGID                 int
	ReturnForLifecycleDescriptor builder.LifecycleDescriptor
	ReturnForStack               builder.StackMetadata
	ReturnForRunImages           []builder.RunImageMetadata
	ReturnForOrderExtensions     dist.Order
}

func NewFakeBuilder(ops ...func(*FakeBuilder)) (*FakeBuilder, error) {
	fakeBuilder := &FakeBuilder{
		ReturnForImage: ifakes.NewImage("some-builder-name", "", nil),
		ReturnForUID:   99,
		ReturnForGID:   99,
		ReturnForLifecycleDescriptor: builder.LifecycleDescriptor{
			Info: builder.LifecycleInfo{
				Version: &builder.Version{Version: *semver.MustParse("12.34")},
			},
			APIs: builder.LifecycleAPIs{
				Buildpack: builder.APIVersions{
					Supported: builder.APISet{api.MustParse("0.4")},
				},
				Platform: builder.APIVersions{
					Supported: builder.APISet{api.MustParse("0.4")},
				},
			},
		},
		ReturnForStack: builder.StackMetadata{},
	}

	for _, op := range ops {
		op(fakeBuilder)
	}

	return fakeBuilder, nil
}

func WithDeprecatedPlatformAPIs(apis []*api.Version) func(*FakeBuilder) {
	return func(builder *FakeBuilder) {
		builder.ReturnForLifecycleDescriptor.APIs.Platform.Deprecated = apis
	}
}

func WithSupportedPlatformAPIs(apis []*api.Version) func(*FakeBuilder) {
	return func(builder *FakeBuilder) {
		builder.ReturnForLifecycleDescriptor.APIs.Platform.Supported = apis
	}
}

func WithImage(image imgutil.Image) func(*FakeBuilder) {
	return func(builder *FakeBuilder) {
		builder.ReturnForImage = image
	}
}

func WithOrderExtensions(orderExt dist.Order) func(*FakeBuilder) {
	return func(builder *FakeBuilder) {
		builder.ReturnForOrderExtensions = orderExt
	}
}

func WithUID(uid int) func(*FakeBuilder) {
	return func(builder *FakeBuilder) {
		builder.ReturnForUID = uid
	}
}

func WithGID(gid int) func(*FakeBuilder) {
	return func(builder *FakeBuilder) {
		builder.ReturnForGID = gid
	}
}

func (b *FakeBuilder) Name() string {
	return b.ReturnForImage.Name()
}

func (b *FakeBuilder) Image() imgutil.Image {
	return b.ReturnForImage
}

func (b *FakeBuilder) UID() int {
	return b.ReturnForUID
}

func (b *FakeBuilder) GID() int {
	return b.ReturnForGID
}

func (b *FakeBuilder) LifecycleDescriptor() builder.LifecycleDescriptor {
	return b.ReturnForLifecycleDescriptor
}

func (b *FakeBuilder) OrderExtensions() dist.Order {
	return b.ReturnForOrderExtensions
}

func (b *FakeBuilder) Stack() builder.StackMetadata {
	return b.ReturnForStack
}

func (b *FakeBuilder) RunImages() []builder.RunImageMetadata {
	return b.ReturnForRunImages
}

func WithBuilder(builder *FakeBuilder) func(*build.LifecycleOptions) {
	return func(opts *build.LifecycleOptions) {
		opts.Builder = builder
	}
}
