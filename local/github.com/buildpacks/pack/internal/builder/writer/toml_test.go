package writer_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/pelletier/go-toml"

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

func TestTOML(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Builder Writer", testTOML, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testTOML(t *testing.T, when spec.G, it spec.S) {
	const (
		expectedRemoteRunImages = `  [[remote_info.run_images]]
    name = "first/local"
    user_configured = true

  [[remote_info.run_images]]
    name = "second/local"
    user_configured = true

  [[remote_info.run_images]]
    name = "some/run-image"

  [[remote_info.run_images]]
    name = "first/default"

  [[remote_info.run_images]]
    name = "second/default"`

		expectedLocalRunImages = `  [[local_info.run_images]]
    name = "first/local"
    user_configured = true

  [[local_info.run_images]]
    name = "second/local"
    user_configured = true

  [[local_info.run_images]]
    name = "some/run-image"

  [[local_info.run_images]]
    name = "first/local-default"

  [[local_info.run_images]]
    name = "second/local-default"`

		expectedLocalBuildpacks = `  [[local_info.buildpacks]]
    id = "test.top.nested"
    version = "test.top.nested.version"

  [[local_info.buildpacks]]
    id = "test.nested"
    homepage = "http://geocities.com/top-bp"

  [[local_info.buildpacks]]
    id = "test.bp.one"
    version = "test.bp.one.version"
    homepage = "http://geocities.com/cool-bp"

  [[local_info.buildpacks]]
    id = "test.bp.two"
    version = "test.bp.two.version"

  [[local_info.buildpacks]]
    id = "test.bp.three"
    version = "test.bp.three.version"`

		expectedRemoteBuildpacks = `  [[remote_info.buildpacks]]
    id = "test.top.nested"
    version = "test.top.nested.version"

  [[remote_info.buildpacks]]
    id = "test.nested"
    homepage = "http://geocities.com/top-bp"

  [[remote_info.buildpacks]]
    id = "test.bp.one"
    version = "test.bp.one.version"
    homepage = "http://geocities.com/cool-bp"

  [[remote_info.buildpacks]]
    id = "test.bp.two"
    version = "test.bp.two.version"

  [[remote_info.buildpacks]]
    id = "test.bp.three"
    version = "test.bp.three.version"`

		expectedLocalDetectionOrder = `  [[local_info.detection_order]]

    [[local_info.detection_order.buildpacks]]
      id = "test.top.nested"
      version = "test.top.nested.version"

      [[local_info.detection_order.buildpacks.buildpacks]]
        id = "test.nested"
        homepage = "http://geocities.com/top-bp"

        [[local_info.detection_order.buildpacks.buildpacks.buildpacks]]
          id = "test.bp.one"
          version = "test.bp.one.version"
          homepage = "http://geocities.com/cool-bp"
          optional = true

      [[local_info.detection_order.buildpacks.buildpacks]]
        id = "test.bp.three"
        version = "test.bp.three.version"
        optional = true

      [[local_info.detection_order.buildpacks.buildpacks]]
        id = "test.nested.two"
        version = "test.nested.two.version"

        [[local_info.detection_order.buildpacks.buildpacks.buildpacks]]
          id = "test.bp.one"
          version = "test.bp.one.version"
          homepage = "http://geocities.com/cool-bp"
          optional = true
          cyclic = true

    [[local_info.detection_order.buildpacks]]
      id = "test.bp.two"
      version = "test.bp.two.version"
      optional = true

  [[local_info.detection_order]]
    id = "test.bp.three"
    version = "test.bp.three.version"`

		expectedRemoteDetectionOrder = `  [[remote_info.detection_order]]

    [[remote_info.detection_order.buildpacks]]
      id = "test.top.nested"
      version = "test.top.nested.version"

      [[remote_info.detection_order.buildpacks.buildpacks]]
        id = "test.nested"
        homepage = "http://geocities.com/top-bp"

        [[remote_info.detection_order.buildpacks.buildpacks.buildpacks]]
          id = "test.bp.one"
          version = "test.bp.one.version"
          homepage = "http://geocities.com/cool-bp"
          optional = true

      [[remote_info.detection_order.buildpacks.buildpacks]]
        id = "test.bp.three"
        version = "test.bp.three.version"
        optional = true

      [[remote_info.detection_order.buildpacks.buildpacks]]
        id = "test.nested.two"
        version = "test.nested.two.version"

        [[remote_info.detection_order.buildpacks.buildpacks.buildpacks]]
          id = "test.bp.one"
          version = "test.bp.one.version"
          homepage = "http://geocities.com/cool-bp"
          optional = true
          cyclic = true

    [[remote_info.detection_order.buildpacks]]
      id = "test.bp.two"
      version = "test.bp.two.version"
      optional = true

  [[remote_info.detection_order]]
    id = "test.bp.three"
    version = "test.bp.three.version"`

		stackWithMixins = `  [stack]
    id = "test.stack.id"
    mixins = ["mixin1", "mixin2", "build:mixin3", "build:mixin4"]`
	)

	var (
		assert = h.NewAssertionManager(t)
		outBuf bytes.Buffer

		remoteInfo *client.BuilderInfo
		localInfo  *client.BuilderInfo

		expectedRemoteInfo = fmt.Sprintf(`[remote_info]
  description = "Some remote description"

  [remote_info.created_by]
    Name = "Pack CLI"
    Version = "1.2.3"

  [remote_info.stack]
    id = "test.stack.id"

  [remote_info.lifecycle]
    version = "6.7.8"

    [remote_info.lifecycle.buildpack_apis]
      deprecated = []
      supported = ["1.2", "2.3"]

    [remote_info.lifecycle.platform_apis]
      deprecated = ["0.1", "1.2"]
      supported = ["4.5"]

%s

%s

%s`, expectedRemoteRunImages, expectedRemoteBuildpacks, expectedRemoteDetectionOrder)

		expectedLocalInfo = fmt.Sprintf(`[local_info]
  description = "Some local description"

  [local_info.created_by]
    Name = "Pack CLI"
    Version = "4.5.6"

  [local_info.stack]
    id = "test.stack.id"

  [local_info.lifecycle]
    version = "4.5.6"

    [local_info.lifecycle.buildpack_apis]
      deprecated = ["4.5", "6.7"]
      supported = ["8.9", "10.11"]

    [local_info.lifecycle.platform_apis]
      deprecated = []
      supported = ["7.8"]

%s

%s

%s`, expectedLocalRunImages, expectedLocalBuildpacks, expectedLocalDetectionOrder)

		expectedPrettifiedTOML = fmt.Sprintf(`builder_name = "test-builder"
trusted = false
default = false

%s

%s`, expectedRemoteInfo, expectedLocalInfo)
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
		})

		it("prints both local remote builders as valid TOML", func() {
			tomlWriter := writer.NewTOML()

			logger := logging.NewLogWithWriters(&outBuf, &outBuf)
			err := tomlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
			assert.Nil(err)

			assert.Succeeds(validTOMLOutput(outBuf))
			assert.Nil(err)

			assert.ContainsTOML(outBuf.String(), expectedPrettifiedTOML)
		})

		when("builder doesn't exist locally or remotely", func() {
			it("returns an error", func() {
				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := tomlWriter.Print(logger, localRunImages, nil, nil, nil, nil, sharedBuilderInfo)
				assert.ErrorWithMessage(err, "unable to find builder 'test-builder' locally or remotely")
			})
		})

		when("builder doesn't exist locally", func() {
			it("shows null for local builder, and normal output for remote", func() {
				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := tomlWriter.Print(logger, localRunImages, nil, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Succeeds(validTOMLOutput(outBuf))
				assert.Nil(err)

				assert.NotContains(outBuf.String(), "local_info")
				assert.ContainsTOML(outBuf.String(), expectedRemoteInfo)
			})
		})

		when("builder doesn't exist remotely", func() {
			it("shows null for remote builder, and normal output for local", func() {
				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := tomlWriter.Print(logger, localRunImages, localInfo, nil, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Succeeds(validTOMLOutput(outBuf))
				assert.Nil(err)

				assert.NotContains(outBuf.String(), "remote_info")
				assert.ContainsTOML(outBuf.String(), expectedLocalInfo)
			})
		})

		when("localErr is an error", func() {
			it("returns the error, and doesn't write any toml output", func() {
				expectedErr := errors.New("failed to retrieve local info")

				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := tomlWriter.Print(logger, localRunImages, localInfo, remoteInfo, expectedErr, nil, sharedBuilderInfo)
				assert.ErrorWithMessage(err, "preparing output for 'test-builder': failed to retrieve local info")

				assert.Equal(outBuf.String(), "")
			})
		})

		when("remoteErr is an error", func() {
			it("returns the error, and doesn't write any toml output", func() {
				expectedErr := errors.New("failed to retrieve remote info")

				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := tomlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, expectedErr, sharedBuilderInfo)
				assert.ErrorWithMessage(err, "preparing output for 'test-builder': failed to retrieve remote info")

				assert.Equal(outBuf.String(), "")
			})
		})

		when("logger is verbose", func() {
			it("displays mixins associated with the stack", func() {
				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
				err := tomlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Succeeds(validTOMLOutput(outBuf))
				assert.ContainsTOML(outBuf.String(), stackWithMixins)
			})
		})

		when("no run images are specified", func() {
			it("omits run images from output", func() {
				localInfo.RunImages = []pubbldr.RunImageConfig{}
				remoteInfo.RunImages = []pubbldr.RunImageConfig{}
				emptyLocalRunImages := []config.RunImage{}

				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
				err := tomlWriter.Print(logger, emptyLocalRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Succeeds(validTOMLOutput(outBuf))

				assert.NotContains(outBuf.String(), "run_images")
			})
		})

		when("no buildpacks are specified", func() {
			it("omits buildpacks from output", func() {
				localInfo.Buildpacks = []dist.ModuleInfo{}
				remoteInfo.Buildpacks = []dist.ModuleInfo{}

				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
				err := tomlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Succeeds(validTOMLOutput(outBuf))

				assert.NotContains(outBuf.String(), "local_info.buildpacks")
				assert.NotContains(outBuf.String(), "remote_info.buildpacks")
			})
		})

		when("no detection order is specified", func() {
			it("omits dection order in output", func() {
				localInfo.Order = pubbldr.DetectionOrder{}
				remoteInfo.Order = pubbldr.DetectionOrder{}

				tomlWriter := writer.NewTOML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
				err := tomlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				assert.Succeeds(validTOMLOutput(outBuf))
				assert.NotContains(outBuf.String(), "detection_order")
			})
		})
	})
}

func validTOMLOutput(source bytes.Buffer) error {
	err := toml.NewDecoder(&source).Decode(&struct{}{})
	if err != nil {
		return fmt.Errorf("failed to unmarshal to toml: %w", err)
	}
	return nil
}
