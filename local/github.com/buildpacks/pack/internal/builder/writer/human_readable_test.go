package writer_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	pubbldr "github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/builder/writer"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestHumanReadable(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Builder Writer", testHumanReadable, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testHumanReadable(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = h.NewAssertionManager(t)
		outBuf bytes.Buffer

		remoteInfo *client.BuilderInfo
		localInfo  *client.BuilderInfo

		expectedRemoteOutput = `
REMOTE:

Description: Some remote description

Created By:
  Name: Pack CLI
  Version: 1.2.3

Trusted: No

Stack:
  ID: test.stack.id

Lifecycle:
  Version: 6.7.8
  Buildpack APIs:
    Deprecated: (none)
    Supported: 1.2, 2.3
  Platform APIs:
    Deprecated: 0.1, 1.2
    Supported: 4.5

Run Images:
  first/local     (user-configured)
  second/local    (user-configured)
  some/run-image
  first/default
  second/default

Buildpacks:
  ID                     NAME        VERSION                        HOMEPAGE
  test.top.nested        -           test.top.nested.version        -
  test.nested            -                                          http://geocities.com/top-bp
  test.bp.one            -           test.bp.one.version            http://geocities.com/cool-bp
  test.bp.two            -           test.bp.two.version            -
  test.bp.three          -           test.bp.three.version          -

Detection Order:
 ├ Group #1:
 │  ├ test.top.nested@test.top.nested.version
 │  │  └ Group #1:
 │  │     ├ test.nested
 │  │     │  └ Group #1:
 │  │     │     └ test.bp.one@test.bp.one.version      (optional)
 │  │     ├ test.bp.three@test.bp.three.version        (optional)
 │  │     └ test.nested.two@test.nested.two.version
 │  │        └ Group #2:
 │  │           └ test.bp.one@test.bp.one.version    (optional)[cyclic]
 │  └ test.bp.two@test.bp.two.version                (optional)
 └ test.bp.three@test.bp.three.version

Extensions:
  ID                   NAME        VERSION                      HOMEPAGE
  test.bp.one          -           test.bp.one.version          http://geocities.com/cool-bp
  test.bp.two          -           test.bp.two.version          -
  test.bp.three        -           test.bp.three.version        -

Detection Order (Extensions):
 ├ test.top.nested@test.top.nested.version
 ├ test.bp.one@test.bp.one.version            (optional)
 ├ test.bp.two@test.bp.two.version            (optional)
 └ test.bp.three@test.bp.three.version
`
		expectedRemoteOutputWithoutExtensions = `
REMOTE:

Description: Some remote description

Created By:
  Name: Pack CLI
  Version: 1.2.3

Trusted: No

Stack:
  ID: test.stack.id

Lifecycle:
  Version: 6.7.8
  Buildpack APIs:
    Deprecated: (none)
    Supported: 1.2, 2.3
  Platform APIs:
    Deprecated: 0.1, 1.2
    Supported: 4.5

Run Images:
  first/local     (user-configured)
  second/local    (user-configured)
  some/run-image
  first/default
  second/default

Buildpacks:
  ID                     NAME        VERSION                        HOMEPAGE
  test.top.nested        -           test.top.nested.version        -
  test.nested            -                                          http://geocities.com/top-bp
  test.bp.one            -           test.bp.one.version            http://geocities.com/cool-bp
  test.bp.two            -           test.bp.two.version            -
  test.bp.three          -           test.bp.three.version          -

Detection Order:
 ├ Group #1:
 │  ├ test.top.nested@test.top.nested.version
 │  │  └ Group #1:
 │  │     ├ test.nested
 │  │     │  └ Group #1:
 │  │     │     └ test.bp.one@test.bp.one.version      (optional)
 │  │     ├ test.bp.three@test.bp.three.version        (optional)
 │  │     └ test.nested.two@test.nested.two.version
 │  │        └ Group #2:
 │  │           └ test.bp.one@test.bp.one.version    (optional)[cyclic]
 │  └ test.bp.two@test.bp.two.version                (optional)
 └ test.bp.three@test.bp.three.version
`

		expectedLocalOutput = `
LOCAL:

Description: Some local description

Created By:
  Name: Pack CLI
  Version: 4.5.6

Trusted: No

Stack:
  ID: test.stack.id

Lifecycle:
  Version: 4.5.6
  Buildpack APIs:
    Deprecated: 4.5, 6.7
    Supported: 8.9, 10.11
  Platform APIs:
    Deprecated: (none)
    Supported: 7.8

Run Images:
  first/local     (user-configured)
  second/local    (user-configured)
  some/run-image
  first/local-default
  second/local-default

Buildpacks:
  ID                     NAME        VERSION                        HOMEPAGE
  test.top.nested        -           test.top.nested.version        -
  test.nested            -                                          http://geocities.com/top-bp
  test.bp.one            -           test.bp.one.version            http://geocities.com/cool-bp
  test.bp.two            -           test.bp.two.version            -
  test.bp.three          -           test.bp.three.version          -

Detection Order:
 ├ Group #1:
 │  ├ test.top.nested@test.top.nested.version
 │  │  └ Group #1:
 │  │     ├ test.nested
 │  │     │  └ Group #1:
 │  │     │     └ test.bp.one@test.bp.one.version      (optional)
 │  │     ├ test.bp.three@test.bp.three.version        (optional)
 │  │     └ test.nested.two@test.nested.two.version
 │  │        └ Group #2:
 │  │           └ test.bp.one@test.bp.one.version    (optional)[cyclic]
 │  └ test.bp.two@test.bp.two.version                (optional)
 └ test.bp.three@test.bp.three.version

Extensions:
  ID                   NAME        VERSION                      HOMEPAGE
  test.bp.one          -           test.bp.one.version          http://geocities.com/cool-bp
  test.bp.two          -           test.bp.two.version          -
  test.bp.three        -           test.bp.three.version        -

Detection Order (Extensions):
 ├ test.top.nested@test.top.nested.version
 ├ test.bp.one@test.bp.one.version            (optional)
 ├ test.bp.two@test.bp.two.version            (optional)
 └ test.bp.three@test.bp.three.version
`

		expectedLocalOutputWithoutExtensions = `
LOCAL:

Description: Some local description

Created By:
  Name: Pack CLI
  Version: 4.5.6

Trusted: No

Stack:
  ID: test.stack.id

Lifecycle:
  Version: 4.5.6
  Buildpack APIs:
    Deprecated: 4.5, 6.7
    Supported: 8.9, 10.11
  Platform APIs:
    Deprecated: (none)
    Supported: 7.8

Run Images:
  first/local     (user-configured)
  second/local    (user-configured)
  some/run-image
  first/local-default
  second/local-default

Buildpacks:
  ID                     NAME        VERSION                        HOMEPAGE
  test.top.nested        -           test.top.nested.version        -
  test.nested            -                                          http://geocities.com/top-bp
  test.bp.one            -           test.bp.one.version            http://geocities.com/cool-bp
  test.bp.two            -           test.bp.two.version            -
  test.bp.three          -           test.bp.three.version          -

Detection Order:
 ├ Group #1:
 │  ├ test.top.nested@test.top.nested.version
 │  │  └ Group #1:
 │  │     ├ test.nested
 │  │     │  └ Group #1:
 │  │     │     └ test.bp.one@test.bp.one.version      (optional)
 │  │     ├ test.bp.three@test.bp.three.version        (optional)
 │  │     └ test.nested.two@test.nested.two.version
 │  │        └ Group #2:
 │  │           └ test.bp.one@test.bp.one.version    (optional)[cyclic]
 │  └ test.bp.two@test.bp.two.version                (optional)
 └ test.bp.three@test.bp.three.version
`

		expectedVerboseStack = `
Stack:
  ID: test.stack.id
  Mixins:
    mixin1
    mixin2
    build:mixin3
    build:mixin4
`
		expectedNilLifecycleVersion = `
Lifecycle:
  Version: (none)
`
		expectedEmptyRunImages = `
Run Images:
  (none)
`
		expectedEmptyBuildpacks = `
Buildpacks:
  (none)
`
		expectedEmptyOrder = `
Detection Order:
  (none)
`
		expectedEmptyOrderExt = `
Detection Order (Extensions):
  (none)
`
		expectedMissingLocalInfo = `
LOCAL:
(not present)
`
		expectedMissingRemoteInfo = `
REMOTE:
(not present)
`
	)

	when("Print", func() {
		it.Before(func() {
			remoteInfo = &client.BuilderInfo{
				Description:     "Some remote description",
				Stack:           "test.stack.id",
				Mixins:          []string{"mixin1", "mixin2", "build:mixin3", "build:mixin4"},
				RunImages:       []pubbldr.RunImageConfig{{Image: "some/run-image", Mirrors: []string{"first/default", "second/default"}}},
				Buildpacks:      buildpacks,
				Order:           order,
				Extensions:      extensions,
				OrderExtensions: orderExtensions,
				BuildpackLayers: dist.ModuleLayers{},
				Lifecycle: builder.LifecycleDescriptor{
					Info: builder.LifecycleInfo{
						Version: &builder.Version{
							Version: *semver.MustParse("6.7.8"),
						},
					},
					APIs: builder.LifecycleAPIs{
						Buildpack: builder.APIVersions{
							Deprecated: nil,
							Supported:  builder.APISet{api.MustParse("1.2"), api.MustParse("2.3")},
						},
						Platform: builder.APIVersions{
							Deprecated: builder.APISet{api.MustParse("0.1"), api.MustParse("1.2")},
							Supported:  builder.APISet{api.MustParse("4.5")},
						},
					},
				},
				CreatedBy: builder.CreatorMetadata{
					Name:    "Pack CLI",
					Version: "1.2.3",
				},
			}

			localInfo = &client.BuilderInfo{
				Description:     "Some local description",
				Stack:           "test.stack.id",
				Mixins:          []string{"mixin1", "mixin2", "build:mixin3", "build:mixin4"},
				RunImages:       []pubbldr.RunImageConfig{{Image: "some/run-image", Mirrors: []string{"first/local-default", "second/local-default"}}},
				Buildpacks:      buildpacks,
				Order:           order,
				Extensions:      extensions,
				OrderExtensions: orderExtensions,
				BuildpackLayers: dist.ModuleLayers{},
				Lifecycle: builder.LifecycleDescriptor{
					Info: builder.LifecycleInfo{
						Version: &builder.Version{
							Version: *semver.MustParse("4.5.6"),
						},
					},
					APIs: builder.LifecycleAPIs{
						Buildpack: builder.APIVersions{
							Deprecated: builder.APISet{api.MustParse("4.5"), api.MustParse("6.7")},
							Supported:  builder.APISet{api.MustParse("8.9"), api.MustParse("10.11")},
						},
						Platform: builder.APIVersions{
							Deprecated: nil,
							Supported:  builder.APISet{api.MustParse("7.8")},
						},
					},
				},
				CreatedBy: builder.CreatorMetadata{
					Name:    "Pack CLI",
					Version: "4.5.6",
				},
			}

			outBuf = bytes.Buffer{}
		})

		it("prints both local and remote builders in a human readable format", func() {
			humanReadableWriter := writer.NewHumanReadable()

			logger := logging.NewLogWithWriters(&outBuf, &outBuf)
			err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
			assert.Nil(err)

			assert.Contains(outBuf.String(), "Inspecting builder: 'test-builder'")
			assert.Contains(outBuf.String(), expectedRemoteOutput)
			assert.Contains(outBuf.String(), expectedLocalOutput)
		})

		when("builder is default", func() {
			it("prints inspecting default builder", func() {
				defaultSharedBuildInfo := sharedBuilderInfo
				defaultSharedBuildInfo.IsDefault = true

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, defaultSharedBuildInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), "Inspecting default builder: 'test-builder'")
			})
		})

		when("builder doesn't exist locally or remotely", func() {
			it("returns an error", func() {
				localInfo = nil
				remoteInfo = nil

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.ErrorWithMessage(err, "unable to find builder 'test-builder' locally or remotely")
			})
		})

		when("builder doesn't exist locally", func() {
			it("shows not present for local builder, and normal output for remote", func() {
				localInfo = nil

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedMissingLocalInfo)
				assert.Contains(outBuf.String(), expectedRemoteOutput)
			})
		})

		when("builder doesn't exist remotely", func() {
			it("shows not present for remote builder, and normal output for local", func() {
				remoteInfo = nil

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedMissingRemoteInfo)
				assert.Contains(outBuf.String(), expectedLocalOutput)
			})
		})

		when("localErr is an error", func() {
			it("error is logged, local info is not displayed, but remote info is", func() {
				errorMessage := "failed to retrieve local info"

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, errors.New(errorMessage), nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), errorMessage)
				assert.NotContains(outBuf.String(), expectedLocalOutput)
				assert.Contains(outBuf.String(), expectedRemoteOutput)
			})
		})

		when("remoteErr is an error", func() {
			it("error is logged, remote info is not displayed, but local info is", func() {
				errorMessage := "failed to retrieve remote info"

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, errors.New(errorMessage), sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), errorMessage)
				assert.NotContains(outBuf.String(), expectedRemoteOutput)
				assert.Contains(outBuf.String(), expectedLocalOutput)
			})
		})

		when("description is blank", func() {
			it("doesn't print the description block", func() {
				localInfo.Description = ""
				remoteInfo.Description = ""

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.NotContains(outBuf.String(), "Description:")
			})
		})

		when("created by name is blank", func() {
			it("doesn't print created by block", func() {
				localInfo.CreatedBy.Name = ""
				remoteInfo.CreatedBy.Name = ""

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.NotContains(outBuf.String(), "Created By:")
			})
		})

		when("logger is verbose", func() {
			it("displays mixins associated with the stack", func() {
				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedVerboseStack)
			})
		})

		when("lifecycle version is not set", func() {
			it("displays lifecycle version as (none) and warns that version if not set", func() {
				localInfo.Lifecycle.Info.Version = nil
				remoteInfo.Lifecycle.Info.Version = nil

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedNilLifecycleVersion)
				assert.Contains(outBuf.String(), "test-builder does not specify a Lifecycle version")
			})
		})

		when("there are no supported buildpack APIs specified", func() {
			it("prints a warning", func() {
				localInfo.Lifecycle.APIs.Buildpack.Supported = builder.APISet{}
				remoteInfo.Lifecycle.APIs.Buildpack.Supported = builder.APISet{}

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), "test-builder does not specify supported Lifecycle Buildpack APIs")
			})
		})

		when("there are no supported platform APIs specified", func() {
			it("prints a warning", func() {
				localInfo.Lifecycle.APIs.Platform.Supported = builder.APISet{}
				remoteInfo.Lifecycle.APIs.Platform.Supported = builder.APISet{}

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), "test-builder does not specify supported Lifecycle Platform APIs")
			})
		})

		when("no run images are specified", func() {
			it("displays run images as (none) and warns about unset run image", func() {
				localInfo.RunImages = []pubbldr.RunImageConfig{}
				remoteInfo.RunImages = []pubbldr.RunImageConfig{}
				emptyLocalRunImages := []config.RunImage{}

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, emptyLocalRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedEmptyRunImages)
				assert.Contains(outBuf.String(), "test-builder does not specify a run image")
				assert.Contains(outBuf.String(), "Users must build with an explicitly specified run image")
			})
		})

		when("no buildpacks are specified", func() {
			it("displays buildpacks as (none) and prints warnings", func() {
				localInfo.Buildpacks = []dist.ModuleInfo{}
				remoteInfo.Buildpacks = []dist.ModuleInfo{}

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedEmptyBuildpacks)
				assert.Contains(outBuf.String(), "test-builder has no buildpacks")
				assert.Contains(outBuf.String(), "Users must supply buildpacks from the host machine")
			})
		})

		when("no extensions are specified", func() {
			it("displays no extensions as (none)", func() {
				localInfo.Extensions = []dist.ModuleInfo{}
				remoteInfo.Extensions = []dist.ModuleInfo{}

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), "Inspecting builder: 'test-builder'")
				assert.Contains(outBuf.String(), expectedRemoteOutputWithoutExtensions)
				assert.Contains(outBuf.String(), expectedLocalOutputWithoutExtensions)
			})
		})

		when("multiple top level groups", func() {
			it("displays order correctly", func() {

			})
		})

		when("no detection order is specified", func() {
			it("displays detection order as (none) and prints warnings", func() {
				localInfo.Order = pubbldr.DetectionOrder{}
				remoteInfo.Order = pubbldr.DetectionOrder{}

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedEmptyOrder)
				assert.Contains(outBuf.String(), "test-builder has no buildpacks")
				assert.Contains(outBuf.String(), "Users must build with explicitly specified buildpacks")
			})
		})

		when("no detection order for extension is specified", func() {
			it("displays detection order for extensions as (none)", func() {
				localInfo.OrderExtensions = pubbldr.DetectionOrder{}
				remoteInfo.OrderExtensions = pubbldr.DetectionOrder{}

				humanReadableWriter := writer.NewHumanReadable()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := humanReadableWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Contains(outBuf.String(), expectedEmptyOrderExt)
			})
		})
	})
}
