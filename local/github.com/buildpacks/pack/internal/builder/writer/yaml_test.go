package writer_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	yaml "gopkg.in/yaml.v3"

	pubbldr "github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/builder/writer"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestYAML(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Builder Writer", testYAML, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testYAML(t *testing.T, when spec.G, it spec.S) {
	const (
		expectedRemoteRunImages = `    run_images:
        - name: first/local
          user_configured: true
        - name: second/local
          user_configured: true
        - name: some/run-image
        - name: first/default
        - name: second/default`

		expectedLocalRunImages = `    run_images:
        - name: first/local
          user_configured: true
        - name: second/local
          user_configured: true
        - name: some/run-image
        - name: first/local-default
        - name: second/local-default`

		expectedBuildpacks = `    buildpacks:
        - id: test.top.nested
          version: test.top.nested.version
        - id: test.nested
          homepage: http://geocities.com/top-bp
        - id: test.bp.one
          version: test.bp.one.version
          homepage: http://geocities.com/cool-bp
        - id: test.bp.two
          version: test.bp.two.version
        - id: test.bp.three
          version: test.bp.three.version`

		expectedExtensions = `    extensions:
        - id: test.bp.one
          version: test.bp.one.version
          homepage: http://geocities.com/cool-bp
        - id: test.bp.two
          version: test.bp.two.version
        - id: test.bp.three
          version: test.bp.three.version`

		expectedDetectionOrder = `    detection_order:
        - buildpacks:
            - id: test.top.nested
              version: test.top.nested.version
              buildpacks:
                - id: test.nested
                  homepage: http://geocities.com/top-bp
                  buildpacks:
                    - id: test.bp.one
                      version: test.bp.one.version
                      homepage: http://geocities.com/cool-bp
                      optional: true
                - id: test.bp.three
                  version: test.bp.three.version
                  optional: true
                - id: test.nested.two
                  version: test.nested.two.version
                  buildpacks:
                    - id: test.bp.one
                      version: test.bp.one.version
                      homepage: http://geocities.com/cool-bp
                      optional: true
                      cyclic: true
            - id: test.bp.two
              version: test.bp.two.version
              optional: true
        - id: test.bp.three
          version: test.bp.three.version`

		expectedOrderExtensions = `    order_extensions:
        - id: test.top.nested
          version: test.top.nested.version
        - id: test.bp.one
          version: test.bp.one.version
          homepage: http://geocities.com/cool-bp
          optional: true
        - id: test.bp.two
          version: test.bp.two.version
          optional: true
        - id: test.bp.three
          version: test.bp.three.version`
		expectedStackWithMixins = `    stack:
        id: test.stack.id
        mixins:
            - mixin1
            - mixin2
            - build:mixin3
            - build:mixin4`
	)

	var (
		assert = h.NewAssertionManager(t)
		outBuf bytes.Buffer

		remoteInfo *client.BuilderInfo
		localInfo  *client.BuilderInfo

		expectedRemoteInfo = fmt.Sprintf(`remote_info:
    description: Some remote description
    created_by:
        name: Pack CLI
        version: 1.2.3
    stack:
        id: test.stack.id
    lifecycle:
        version: 6.7.8
        buildpack_apis:
            deprecated: []
            supported:
                - "1.2"
                - "2.3"
        platform_apis:
            deprecated:
                - "0.1"
                - "1.2"
            supported:
                - "4.5"
%s
%s
%s
%s
%s`, expectedRemoteRunImages, expectedBuildpacks, expectedDetectionOrder, expectedExtensions, expectedOrderExtensions)

		expectedLocalInfo = fmt.Sprintf(`local_info:
    description: Some local description
    created_by:
        name: Pack CLI
        version: 4.5.6
    stack:
        id: test.stack.id
    lifecycle:
        version: 4.5.6
        buildpack_apis:
            deprecated:
                - "4.5"
                - "6.7"
            supported:
                - "8.9"
                - "10.11"
        platform_apis:
            deprecated: []
            supported:
                - "7.8"
%s
%s
%s
%s
%s`, expectedLocalRunImages, expectedBuildpacks, expectedDetectionOrder, expectedExtensions, expectedOrderExtensions)

		expectedPrettifiedYAML = fmt.Sprintf(`    builder_name: test-builder
    trusted: false
    default: false
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
		})

		it("prints both local remote builders as valid YAML", func() {
			yamlWriter := writer.NewYAML()

			logger := logging.NewLogWithWriters(&outBuf, &outBuf)
			err := yamlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
			assert.Nil(err)

			prettyYAML, err := validYAML(outBuf)
			assert.Nil(err)

			assert.Contains(prettyYAML, expectedPrettifiedYAML)
		})

		when("builder doesn't exist locally or remotely", func() {
			it("returns an error", func() {
				yamlWriter := writer.NewYAML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := yamlWriter.Print(logger, localRunImages, nil, nil, nil, nil, sharedBuilderInfo)
				assert.ErrorWithMessage(err, "unable to find builder 'test-builder' locally or remotely")
			})
		})

		when("builder doesn't exist locally", func() {
			it("shows null for local builder, and normal output for remote", func() {
				yamlWriter := writer.NewYAML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := yamlWriter.Print(logger, localRunImages, nil, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				prettyYAML, err := validYAML(outBuf)
				assert.Nil(err)

				assert.ContainsYAML(prettyYAML, `local_info: null`)
				assert.ContainsYAML(prettyYAML, expectedRemoteInfo)
			})
		})

		when("builder doesn't exist remotely", func() {
			it("shows null for remote builder, and normal output for local", func() {
				yamlWriter := writer.NewYAML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := yamlWriter.Print(logger, localRunImages, localInfo, nil, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				prettyYAML, err := validYAML(outBuf)
				assert.Nil(err)

				assert.ContainsYAML(prettyYAML, `remote_info: null`)
				assert.ContainsYAML(prettyYAML, expectedLocalInfo)
			})
		})

		when("localErr is an error", func() {
			it("returns the error, and doesn't write any yaml output", func() {
				expectedErr := errors.New("failed to retrieve local info")

				yamlWriter := writer.NewYAML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := yamlWriter.Print(logger, localRunImages, localInfo, remoteInfo, expectedErr, nil, sharedBuilderInfo)
				assert.ErrorWithMessage(err, "preparing output for 'test-builder': failed to retrieve local info")

				assert.Equal(outBuf.String(), "")
			})
		})

		when("remoteErr is an error", func() {
			it("returns the error, and doesn't write any yaml output", func() {
				expectedErr := errors.New("failed to retrieve remote info")

				yamlWriter := writer.NewYAML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf)
				err := yamlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, expectedErr, sharedBuilderInfo)
				assert.ErrorWithMessage(err, "preparing output for 'test-builder': failed to retrieve remote info")

				assert.Equal(outBuf.String(), "")
			})
		})

		when("logger is verbose", func() {
			it("displays mixins associated with the stack", func() {
				yamlWriter := writer.NewYAML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
				err := yamlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				prettifiedYAML, err := validYAML(outBuf)
				assert.Nil(err)

				assert.ContainsYAML(prettifiedYAML, expectedStackWithMixins)
			})
		})

		when("no run images are specified", func() {
			it("displays run images as empty list", func() {
				localInfo.RunImages = []pubbldr.RunImageConfig{}
				remoteInfo.RunImages = []pubbldr.RunImageConfig{}
				emptyLocalRunImages := []config.RunImage{}

				yamlWriter := writer.NewYAML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
				err := yamlWriter.Print(logger, emptyLocalRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				prettifiedYAML, err := validYAML(outBuf)
				assert.Nil(err)

				assert.ContainsYAML(prettifiedYAML, `run_images: []`)
			})
		})

		when("no buildpacks are specified", func() {
			it("displays buildpacks as empty list", func() {
				localInfo.Buildpacks = []dist.ModuleInfo{}
				remoteInfo.Buildpacks = []dist.ModuleInfo{}

				yamlWriter := writer.NewYAML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
				err := yamlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				prettifiedYAML, err := validYAML(outBuf)
				assert.Nil(err)

				assert.ContainsYAML(prettifiedYAML, `buildpacks: []`)
			})
		})

		when("no detection order is specified", func() {
			it("displays detection order as empty list", func() {
				localInfo.Order = pubbldr.DetectionOrder{}
				remoteInfo.Order = pubbldr.DetectionOrder{}

				yamlWriter := writer.NewYAML()

				logger := logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
				err := yamlWriter.Print(logger, localRunImages, localInfo, remoteInfo, nil, nil, sharedBuilderInfo)
				assert.Nil(err)

				prettifiedYAML, err := validYAML(outBuf)
				assert.Nil(err)

				assert.ContainsYAML(prettifiedYAML, `detection_order: []`)
			})
		})
	})
}

func validYAML(source bytes.Buffer) (string, error) {
	err := yaml.Unmarshal(source.Bytes(), &struct{}{})
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal to yaml: %w", err)
	}

	return source.String(), nil
}
