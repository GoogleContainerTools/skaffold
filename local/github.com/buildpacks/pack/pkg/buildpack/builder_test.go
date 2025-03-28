package buildpack_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/buildpacks/imgutil/layer"
	"github.com/buildpacks/lifecycle/api"
	"github.com/golang/mock/gomock"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"github.com/heroku/color"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/logging"

	ifakes "github.com/buildpacks/pack/internal/fakes"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPackageBuilder(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "PackageBuilder", testPackageBuilder, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testPackageBuilder(t *testing.T, when spec.G, it spec.S) {
	var (
		mockController   *gomock.Controller
		mockImageFactory func(expectedImageOS string) *testmocks.MockImageFactory
		tmpDir           string
	)

	it.Before(func() {
		mockController = gomock.NewController(t)

		mockImageFactory = func(expectedImageOS string) *testmocks.MockImageFactory {
			imageFactory := testmocks.NewMockImageFactory(mockController)

			if expectedImageOS != "" {
				fakePackageImage := fakes.NewImage("some/package", "", nil)
				imageFactory.EXPECT().NewImage("some/package", true, dist.Target{OS: expectedImageOS}).Return(fakePackageImage, nil).MaxTimes(1)
			}

			return imageFactory
		}

		var err error
		tmpDir, err = os.MkdirTemp("", "package_builder_tests")
		h.AssertNil(t, err)
	})

	it.After(func() {
		h.AssertNilE(t, os.RemoveAll(tmpDir))
		mockController.Finish()
	})

	when("validation", func() {
		linux := dist.Target{OS: "linux"}
		windows := dist.Target{OS: "windows"}

		for _, _test := range []*struct {
			name            string
			expectedImageOS string
			fn              func(*buildpack.PackageBuilder) error
		}{
			{name: "SaveAsImage", expectedImageOS: "linux", fn: func(builder *buildpack.PackageBuilder) error {
				_, err := builder.SaveAsImage("some/package", false, linux, map[string]string{})
				return err
			}},
			{name: "SaveAsImage", expectedImageOS: "windows", fn: func(builder *buildpack.PackageBuilder) error {
				_, err := builder.SaveAsImage("some/package", false, windows, map[string]string{})
				return err
			}},
			{name: "SaveAsFile", expectedImageOS: "linux", fn: func(builder *buildpack.PackageBuilder) error {
				return builder.SaveAsFile(path.Join(tmpDir, "package.cnb"), linux, map[string]string{})
			}},
			{name: "SaveAsFile", expectedImageOS: "windows", fn: func(builder *buildpack.PackageBuilder) error {
				return builder.SaveAsFile(path.Join(tmpDir, "package.cnb"), windows, map[string]string{})
			}},
		} {
			// always use copies to avoid stale refs
			testFn := _test.fn
			expectedImageOS := _test.expectedImageOS
			testName := _test.name
			_test = nil

			when(testName, func() {
				when(expectedImageOS, func() {
					when("validate buildpack", func() {
						when("buildpack not set", func() {
							it("returns error", func() {
								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								err := testFn(builder)
								h.AssertError(t, err, "buildpack or extension must be set")
							})
						})

						when("there is a buildpack not referenced", func() {
							it("should error", func() {
								bp1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithStacks: []dist.Stack{{ID: "some.stack"}},
								}, 0644)
								h.AssertNil(t, err)

								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(bp1)

								bp2, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI:    api.MustParse("0.2"),
									WithInfo:   dist.ModuleInfo{ID: "bp.2.id", Version: "bp.2.version"},
									WithStacks: []dist.Stack{{ID: "some.stack"}},
									WithOrder:  nil,
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(bp2)

								err = testFn(builder)
								h.AssertError(t, err, "buildpack 'bp.2.id@bp.2.version' is not used by buildpack 'bp.1.id@bp.1.version'")
							})
						})

						when("there is a referenced buildpack from main buildpack that is not present", func() {
							it("should error", func() {
								mainBP, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{
											{ModuleInfo: dist.ModuleInfo{ID: "bp.present.id", Version: "bp.present.version"}},
											{ModuleInfo: dist.ModuleInfo{ID: "bp.missing.id", Version: "bp.missing.version"}},
										},
									}},
								}, 0644)
								h.AssertNil(t, err)

								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(mainBP)

								presentBP, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI:    api.MustParse("0.2"),
									WithInfo:   dist.ModuleInfo{ID: "bp.present.id", Version: "bp.present.version"},
									WithStacks: []dist.Stack{{ID: "some.stack"}},
									WithOrder:  nil,
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(presentBP)

								err = testFn(builder)
								h.AssertError(t, err, "buildpack 'bp.1.id@bp.1.version' references buildpack 'bp.missing.id@bp.missing.version' which is not present")
							})
						})

						when("there is a referenced buildpack from dependency buildpack that is not present", func() {
							it("should error", func() {
								mainBP, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{
											{ModuleInfo: dist.ModuleInfo{ID: "bp.present.id", Version: "bp.present.version"}},
										},
									}},
								}, 0644)
								h.AssertNil(t, err)
								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(mainBP)

								presentBP, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI:  api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{ID: "bp.present.id", Version: "bp.present.version"},
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{
											{ModuleInfo: dist.ModuleInfo{ID: "bp.missing.id", Version: "bp.missing.version"}},
										},
									}},
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(presentBP)

								err = testFn(builder)
								h.AssertError(t, err, "buildpack 'bp.present.id@bp.present.version' references buildpack 'bp.missing.id@bp.missing.version' which is not present")
							})
						})

						when("there is a referenced buildpack from dependency buildpack that does not have proper version defined", func() {
							it("should error", func() {
								mainBP, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{
											{ModuleInfo: dist.ModuleInfo{ID: "bp.present.id", Version: "bp.present.version"}},
										},
									}},
								}, 0644)
								h.AssertNil(t, err)
								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(mainBP)

								presentBP, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI:  api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{ID: "bp.present.id", Version: "bp.present.version"},
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{
											{ModuleInfo: dist.ModuleInfo{ID: "bp.missing.id"}},
										},
									}},
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(presentBP)

								err = testFn(builder)
								h.AssertError(t, err, "buildpack 'bp.present.id@bp.present.version' must specify a version when referencing buildpack 'bp.missing.id'")
							})
						})
					})

					when("validate stacks", func() {
						when("buildpack does not define stacks", func() {
							it("should succeed", func() {
								bp, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.10"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithStacks: nil,
									WithOrder:  nil,
								}, 0644)
								h.AssertNil(t, err)
								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(bp)
								err = testFn(builder)
								h.AssertNil(t, err)
							})
						})

						when("buildpack is meta-buildpack", func() {
							it("should succeed", func() {
								bp, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithStacks: nil,
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{
											{ModuleInfo: dist.ModuleInfo{ID: "bp.nested.id", Version: "bp.nested.version"}},
										},
									}},
								}, 0644)
								h.AssertNil(t, err)

								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(bp)

								dependency, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.nested.id",
										Version: "bp.nested.version",
									},
									WithStacks: []dist.Stack{
										{ID: "stack.id.1", Mixins: []string{"Mixin-A"}},
									},
									WithOrder: nil,
								}, 0644)
								h.AssertNil(t, err)

								builder.AddDependency(dependency)

								err = testFn(builder)
								h.AssertNil(t, err)
							})
						})

						when("dependencies don't have a common stack", func() {
							it("should error", func() {
								bp, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{{
											ModuleInfo: dist.ModuleInfo{ID: "bp.2.id", Version: "bp.2.version"},
											Optional:   false,
										}, {
											ModuleInfo: dist.ModuleInfo{ID: "bp.3.id", Version: "bp.3.version"},
											Optional:   false,
										}},
									}},
								}, 0644)
								h.AssertNil(t, err)

								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(bp)

								dependency1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.2.id",
										Version: "bp.2.version",
									},
									WithStacks: []dist.Stack{
										{ID: "stack.id.1", Mixins: []string{"Mixin-A"}},
										{ID: "stack.id.2", Mixins: []string{"Mixin-A"}},
									},
									WithOrder: nil,
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(dependency1)

								dependency2, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.3.id",
										Version: "bp.3.version",
									},
									WithStacks: []dist.Stack{
										{ID: "stack.id.3", Mixins: []string{"Mixin-A"}},
									},
									WithOrder: nil,
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(dependency2)

								err = testFn(builder)
								h.AssertError(t, err, "no compatible stacks among provided buildpacks")
							})
						})

						when("dependency has stacks that aren't supported by buildpack", func() {
							it("should only support common stacks", func() {
								bp, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{{
											ModuleInfo: dist.ModuleInfo{ID: "bp.2.id", Version: "bp.2.version"},
											Optional:   false,
										}, {
											ModuleInfo: dist.ModuleInfo{ID: "bp.3.id", Version: "bp.3.version"},
											Optional:   false,
										}},
									}},
								}, 0644)
								h.AssertNil(t, err)

								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(bp)

								dependency1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.2.id",
										Version: "bp.2.version",
									},
									WithStacks: []dist.Stack{
										{ID: "stack.id.1", Mixins: []string{"Mixin-A"}},
										{ID: "stack.id.2", Mixins: []string{"Mixin-A"}},
									},
									WithOrder: nil,
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(dependency1)

								dependency2, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.3.id",
										Version: "bp.3.version",
									},
									WithStacks: []dist.Stack{
										{ID: "stack.id.1", Mixins: []string{"Mixin-A"}},
									},
									WithOrder: nil,
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(dependency2)

								img, err := builder.SaveAsImage("some/package", false, dist.Target{OS: expectedImageOS}, map[string]string{})
								h.AssertNil(t, err)

								metadata := buildpack.Metadata{}
								_, err = dist.GetLabel(img, "io.buildpacks.buildpackage.metadata", &metadata)
								h.AssertNil(t, err)

								h.AssertEq(t, metadata.Stacks, []dist.Stack{{ID: "stack.id.1", Mixins: []string{"Mixin-A"}}})
							})
						})

						when("dependency has wildcard stacks", func() {
							it("should support all the possible stacks", func() {
								bp, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{{
											ModuleInfo: dist.ModuleInfo{ID: "bp.2.id", Version: "bp.2.version"},
											Optional:   false,
										}, {
											ModuleInfo: dist.ModuleInfo{ID: "bp.3.id", Version: "bp.3.version"},
											Optional:   false,
										}},
									}},
								}, 0644)
								h.AssertNil(t, err)

								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(bp)

								dependency1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.2.id",
										Version: "bp.2.version",
									},
									WithStacks: []dist.Stack{
										{ID: "*", Mixins: []string{"Mixin-A"}},
									},
									WithOrder: nil,
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(dependency1)

								dependency2, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.3.id",
										Version: "bp.3.version",
									},
									WithStacks: []dist.Stack{
										{ID: "stack.id.1", Mixins: []string{"Mixin-A"}},
									},
									WithOrder: nil,
								}, 0644)
								h.AssertNil(t, err)
								builder.AddDependency(dependency2)

								img, err := builder.SaveAsImage("some/package", false, dist.Target{OS: expectedImageOS}, map[string]string{})
								h.AssertNil(t, err)

								metadata := buildpack.Metadata{}
								_, err = dist.GetLabel(img, "io.buildpacks.buildpackage.metadata", &metadata)
								h.AssertNil(t, err)

								h.AssertEq(t, metadata.Stacks, []dist.Stack{{ID: "stack.id.1", Mixins: []string{"Mixin-A"}}})
							})
						})

						when("dependency is meta-buildpack", func() {
							it("should succeed and compute common stacks", func() {
								bp, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.1.id",
										Version: "bp.1.version",
									},
									WithStacks: nil,
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{
											{ModuleInfo: dist.ModuleInfo{ID: "bp.nested.id", Version: "bp.nested.version"}},
										},
									}},
								}, 0644)
								h.AssertNil(t, err)

								builder := buildpack.NewBuilder(mockImageFactory(expectedImageOS))
								builder.SetBuildpack(bp)

								dependencyOrder, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.nested.id",
										Version: "bp.nested.version",
									},
									WithOrder: dist.Order{{
										Group: []dist.ModuleRef{
											{ModuleInfo: dist.ModuleInfo{
												ID:      "bp.nested.nested.id",
												Version: "bp.nested.nested.version",
											}},
										},
									}},
								}, 0644)
								h.AssertNil(t, err)

								builder.AddDependency(dependencyOrder)

								dependencyNestedNested, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
									WithAPI: api.MustParse("0.2"),
									WithInfo: dist.ModuleInfo{
										ID:      "bp.nested.nested.id",
										Version: "bp.nested.nested.version",
									},
									WithStacks: []dist.Stack{
										{ID: "stack.id.1", Mixins: []string{"Mixin-A"}},
									},
									WithOrder: nil,
								}, 0644)
								h.AssertNil(t, err)

								builder.AddDependency(dependencyNestedNested)

								img, err := builder.SaveAsImage("some/package", false, dist.Target{OS: expectedImageOS}, map[string]string{})
								h.AssertNil(t, err)

								metadata := buildpack.Metadata{}
								_, err = dist.GetLabel(img, "io.buildpacks.buildpackage.metadata", &metadata)
								h.AssertNil(t, err)

								h.AssertEq(t, metadata.Stacks, []dist.Stack{{ID: "stack.id.1", Mixins: []string{"Mixin-A"}}})
							})
						})
					})
				})
			})
		}
	})

	when("#SaveAsImage", func() {
		it("sets metadata", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				WithAPI: api.MustParse("0.2"),
				WithInfo: dist.ModuleInfo{
					ID:          "bp.1.id",
					Version:     "bp.1.version",
					Name:        "One",
					Description: "some description",
					Homepage:    "https://example.com/homepage",
					Keywords:    []string{"some-keyword"},
					Licenses: []dist.License{
						{
							Type: "MIT",
							URI:  "https://example.com/license",
						},
					},
				},
				WithStacks: []dist.Stack{
					{ID: "stack.id.1"},
					{ID: "stack.id.2"},
				},
				WithOrder: nil,
			}, 0644)
			h.AssertNil(t, err)

			builder := buildpack.NewBuilder(mockImageFactory("linux"))
			builder.SetBuildpack(buildpack1)

			var customLabels = map[string]string{"test.label.one": "1", "test.label.two": "2"}

			packageImage, err := builder.SaveAsImage("some/package", false, dist.Target{OS: "linux"}, customLabels)
			h.AssertNil(t, err)

			labelData, err := packageImage.Label("io.buildpacks.buildpackage.metadata")
			h.AssertNil(t, err)
			var md buildpack.Metadata
			h.AssertNil(t, json.Unmarshal([]byte(labelData), &md))

			h.AssertEq(t, md.ID, "bp.1.id")
			h.AssertEq(t, md.Version, "bp.1.version")
			h.AssertEq(t, len(md.Stacks), 2)
			h.AssertEq(t, md.Stacks[0].ID, "stack.id.1")
			h.AssertEq(t, md.Stacks[1].ID, "stack.id.2")
			h.AssertEq(t, md.Keywords[0], "some-keyword")
			h.AssertEq(t, md.Homepage, "https://example.com/homepage")
			h.AssertEq(t, md.Name, "One")
			h.AssertEq(t, md.Description, "some description")
			h.AssertEq(t, md.Licenses[0].Type, "MIT")
			h.AssertEq(t, md.Licenses[0].URI, "https://example.com/license")

			osVal, err := packageImage.OS()
			h.AssertNil(t, err)
			h.AssertEq(t, osVal, "linux")

			imageLabels, err := packageImage.Labels()
			h.AssertNil(t, err)
			h.AssertEq(t, imageLabels["test.label.one"], "1")
			h.AssertEq(t, imageLabels["test.label.two"], "2")
		})

		it("sets extension metadata", func() {
			extension1, err := ifakes.NewFakeExtension(dist.ExtensionDescriptor{
				WithAPI: api.MustParse("0.2"),
				WithInfo: dist.ModuleInfo{
					ID:          "ex.1.id",
					Version:     "ex.1.version",
					Name:        "One",
					Description: "some description",
					Homepage:    "https://example.com/homepage",
					Keywords:    []string{"some-keyword"},
					Licenses: []dist.License{
						{
							Type: "MIT",
							URI:  "https://example.com/license",
						},
					},
				},
			}, 0644)
			h.AssertNil(t, err)
			builder := buildpack.NewBuilder(mockImageFactory("linux"))
			builder.SetExtension(extension1)
			packageImage, err := builder.SaveAsImage("some/package", false, dist.Target{OS: "linux"}, map[string]string{})
			h.AssertNil(t, err)
			labelData, err := packageImage.Label("io.buildpacks.buildpackage.metadata")
			h.AssertNil(t, err)
			var md buildpack.Metadata
			h.AssertNil(t, json.Unmarshal([]byte(labelData), &md))

			h.AssertEq(t, md.ID, "ex.1.id")
			h.AssertEq(t, md.Version, "ex.1.version")
			h.AssertEq(t, md.Keywords[0], "some-keyword")
			h.AssertEq(t, md.Homepage, "https://example.com/homepage")
			h.AssertEq(t, md.Name, "One")
			h.AssertEq(t, md.Description, "some description")
			h.AssertEq(t, md.Licenses[0].Type, "MIT")
			h.AssertEq(t, md.Licenses[0].URI, "https://example.com/license")

			osVal, err := packageImage.OS()
			h.AssertNil(t, err)
			h.AssertEq(t, osVal, "linux")
		})

		it("sets buildpack layers label", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				WithAPI:    api.MustParse("0.2"),
				WithInfo:   dist.ModuleInfo{ID: "bp.1.id", Version: "bp.1.version"},
				WithStacks: []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}},
				WithOrder:  nil,
			}, 0644)
			h.AssertNil(t, err)

			builder := buildpack.NewBuilder(mockImageFactory("linux"))
			builder.SetBuildpack(buildpack1)

			packageImage, err := builder.SaveAsImage("some/package", false, dist.Target{OS: "linux"}, map[string]string{})
			h.AssertNil(t, err)

			var bpLayers dist.ModuleLayers
			_, err = dist.GetLabel(packageImage, "io.buildpacks.buildpack.layers", &bpLayers)
			h.AssertNil(t, err)

			bp1Info, ok1 := bpLayers["bp.1.id"]["bp.1.version"]
			h.AssertEq(t, ok1, true)
			h.AssertEq(t, bp1Info.Stacks, []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}})
		})

		it("adds buildpack layers for linux", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				WithAPI:    api.MustParse("0.2"),
				WithInfo:   dist.ModuleInfo{ID: "bp.1.id", Version: "bp.1.version"},
				WithStacks: []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}},
				WithOrder:  nil,
			}, 0644)
			h.AssertNil(t, err)

			builder := buildpack.NewBuilder(mockImageFactory("linux"))
			builder.SetBuildpack(buildpack1)

			packageImage, err := builder.SaveAsImage("some/package", false, dist.Target{OS: "linux"}, map[string]string{})
			h.AssertNil(t, err)

			buildpackExists := func(name, version string) {
				t.Helper()
				dirPath := fmt.Sprintf("/cnb/buildpacks/%s/%s", name, version)
				fakePackageImage := packageImage.(*fakes.Image)
				layerTar, err := fakePackageImage.FindLayerWithPath(dirPath)
				h.AssertNil(t, err)

				h.AssertOnTarEntry(t, layerTar, dirPath,
					h.IsDirectory(),
				)

				h.AssertOnTarEntry(t, layerTar, dirPath+"/bin/build",
					h.ContentEquals("build-contents"),
					h.HasOwnerAndGroup(0, 0),
					h.HasFileMode(0644),
				)

				h.AssertOnTarEntry(t, layerTar, dirPath+"/bin/detect",
					h.ContentEquals("detect-contents"),
					h.HasOwnerAndGroup(0, 0),
					h.HasFileMode(0644),
				)
			}

			buildpackExists("bp.1.id", "bp.1.version")

			fakePackageImage := packageImage.(*fakes.Image)
			osVal, err := fakePackageImage.OS()
			h.AssertNil(t, err)
			h.AssertEq(t, osVal, "linux")
		})

		it("adds baselayer + buildpack layers for windows", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				WithAPI:    api.MustParse("0.2"),
				WithInfo:   dist.ModuleInfo{ID: "bp.1.id", Version: "bp.1.version"},
				WithStacks: []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}},
				WithOrder:  nil,
			}, 0644)
			h.AssertNil(t, err)

			builder := buildpack.NewBuilder(mockImageFactory("windows"))
			builder.SetBuildpack(buildpack1)

			_, err = builder.SaveAsImage("some/package", false, dist.Target{OS: "windows"}, map[string]string{})
			h.AssertNil(t, err)
		})

		it("should report an error when custom label cannot be set", func() {
			mockImageFactory = func(expectedImageOS string) *testmocks.MockImageFactory {
				var imageWithLabelError = &imageWithLabelError{Image: fakes.NewImage("some/package", "", nil)}
				imageFactory := testmocks.NewMockImageFactory(mockController)
				imageFactory.EXPECT().NewImage("some/package", true, dist.Target{OS: expectedImageOS}).Return(imageWithLabelError, nil).MaxTimes(1)
				return imageFactory
			}

			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				WithAPI: api.MustParse("0.2"),
				WithInfo: dist.ModuleInfo{
					ID:          "bp.1.id",
					Version:     "bp.1.version",
					Name:        "One",
					Description: "some description",
					Homepage:    "https://example.com/homepage",
					Keywords:    []string{"some-keyword"},
					Licenses: []dist.License{
						{
							Type: "MIT",
							URI:  "https://example.com/license",
						},
					},
				},
				WithStacks: []dist.Stack{
					{ID: "stack.id.1"},
					{ID: "stack.id.2"},
				},
				WithOrder: nil,
			}, 0644)
			h.AssertNil(t, err)

			builder := buildpack.NewBuilder(mockImageFactory("linux"))
			builder.SetBuildpack(buildpack1)

			var customLabels = map[string]string{"test.label.fail": "true"}

			_, err = builder.SaveAsImage("some/package", false, dist.Target{OS: "linux"}, customLabels)
			h.AssertError(t, err, "adding label test.label.fail=true")
		})

		when("flatten is set", func() {
			var (
				buildpack1   buildpack.BuildModule
				bp1          buildpack.BuildModule
				compositeBP2 buildpack.BuildModule
				bp21         buildpack.BuildModule
				bp22         buildpack.BuildModule
				compositeBP3 buildpack.BuildModule
				bp31         buildpack.BuildModule
				logger       logging.Logger
				outBuf       bytes.Buffer
				err          error
			)
			it.Before(func() {
				bp1, err = ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
					WithAPI: api.MustParse("0.2"),
					WithInfo: dist.ModuleInfo{
						ID:      "buildpack-1-id",
						Version: "buildpack-1-version",
					},
				}, 0644)
				h.AssertNil(t, err)

				bp21, err = ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
					WithAPI: api.MustParse("0.2"),
					WithInfo: dist.ModuleInfo{
						ID:      "buildpack-21-id",
						Version: "buildpack-21-version",
					},
				}, 0644)
				h.AssertNil(t, err)

				bp22, err = ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
					WithAPI: api.MustParse("0.2"),
					WithInfo: dist.ModuleInfo{
						ID:      "buildpack-22-id",
						Version: "buildpack-22-version",
					},
				}, 0644)
				h.AssertNil(t, err)

				bp31, err = ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
					WithAPI: api.MustParse("0.2"),
					WithInfo: dist.ModuleInfo{
						ID:      "buildpack-31-id",
						Version: "buildpack-31-version",
					},
				}, 0644)
				h.AssertNil(t, err)

				compositeBP3, err = ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
					WithAPI: api.MustParse("0.2"),
					WithInfo: dist.ModuleInfo{
						ID:      "composite-buildpack-3-id",
						Version: "composite-buildpack-3-version",
					},
					WithOrder: []dist.OrderEntry{{
						Group: []dist.ModuleRef{
							{
								ModuleInfo: bp31.Descriptor().Info(),
							},
						},
					}},
				}, 0644)
				h.AssertNil(t, err)

				compositeBP2, err = ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
					WithAPI: api.MustParse("0.2"),
					WithInfo: dist.ModuleInfo{
						ID:      "composite-buildpack-2-id",
						Version: "composite-buildpack-2-version",
					},
					WithOrder: []dist.OrderEntry{{
						Group: []dist.ModuleRef{
							{
								ModuleInfo: bp21.Descriptor().Info(),
							},
							{
								ModuleInfo: bp22.Descriptor().Info(),
							},
							{
								ModuleInfo: compositeBP3.Descriptor().Info(),
							},
						},
					}},
				}, 0644)
				h.AssertNil(t, err)

				buildpack1, err = ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
					WithAPI:    api.MustParse("0.2"),
					WithInfo:   dist.ModuleInfo{ID: "bp.1.id", Version: "bp.1.version"},
					WithStacks: []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}},
					WithOrder: []dist.OrderEntry{{
						Group: []dist.ModuleRef{
							{
								ModuleInfo: bp1.Descriptor().Info(),
							},
							{
								ModuleInfo: compositeBP2.Descriptor().Info(),
							},
						},
					}},
				}, 0644)
				h.AssertNil(t, err)

				logger = logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
			})

			when("flatten all", func() {
				var builder *buildpack.PackageBuilder

				when("no exclusions", func() {
					it.Before(func() {
						builder = buildpack.NewBuilder(mockImageFactory("linux"),
							buildpack.FlattenAll(),
							buildpack.WithLogger(logger),
							buildpack.WithLayerWriterFactory(archive.DefaultTarWriterFactory()))
					})

					it("flatten all buildpacks", func() {
						builder.SetBuildpack(buildpack1)
						builder.AddDependencies(bp1, nil)
						builder.AddDependencies(compositeBP2, []buildpack.BuildModule{bp21, bp22, compositeBP3, bp31})

						packageImage, err := builder.SaveAsImage("some/package", false, dist.Target{OS: "linux"}, map[string]string{})
						h.AssertNil(t, err)

						fakePackageImage := packageImage.(*fakes.Image)
						h.AssertEq(t, fakePackageImage.NumberOfAddedLayers(), 1)
					})
				})

				when("exclude buildpacks", func() {
					it.Before(func() {
						excluded := []string{bp31.Descriptor().Info().FullName()}

						builder = buildpack.NewBuilder(mockImageFactory("linux"),
							buildpack.DoNotFlatten(excluded),
							buildpack.WithLogger(logger),
							buildpack.WithLayerWriterFactory(archive.DefaultTarWriterFactory()))
					})

					it("creates 2 layers", func() {
						builder.SetBuildpack(buildpack1)
						builder.AddDependencies(bp1, nil)
						builder.AddDependencies(compositeBP2, []buildpack.BuildModule{bp21, bp22, compositeBP3, bp31})

						packageImage, err := builder.SaveAsImage("some/package", false, dist.Target{OS: "linux"}, map[string]string{})
						h.AssertNil(t, err)

						fakePackageImage := packageImage.(*fakes.Image)
						h.AssertEq(t, fakePackageImage.NumberOfAddedLayers(), 2)
					})
				})
			})
		})
	})

	when("#SaveAsFile", func() {
		it("sets metadata", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				WithAPI:    api.MustParse("0.2"),
				WithInfo:   dist.ModuleInfo{ID: "bp.1.id", Version: "bp.1.version"},
				WithStacks: []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}},
				WithOrder:  nil,
			}, 0644)
			h.AssertNil(t, err)

			builder := buildpack.NewBuilder(mockImageFactory(""))
			builder.SetBuildpack(buildpack1)

			var customLabels = map[string]string{"test.label.one": "1", "test.label.two": "2"}

			outputFile := filepath.Join(tmpDir, fmt.Sprintf("package-%s.cnb", h.RandString(10)))
			h.AssertNil(t, builder.SaveAsFile(outputFile, dist.Target{OS: "linux"}, customLabels))

			withContents := func(fn func(data []byte)) h.TarEntryAssertion {
				return func(t *testing.T, header *tar.Header, data []byte) {
					fn(data)
				}
			}

			h.AssertOnTarEntry(t, outputFile, "/index.json",
				h.HasOwnerAndGroup(0, 0),
				h.HasFileMode(0755),
				withContents(func(data []byte) {
					index := v1.Index{}
					err := json.Unmarshal(data, &index)
					h.AssertNil(t, err)
					h.AssertEq(t, len(index.Manifests), 1)

					// manifest: application/vnd.docker.distribution.manifest.v2+json
					h.AssertOnTarEntry(t, outputFile,
						"/blobs/sha256/"+index.Manifests[0].Digest.Hex(),
						h.HasOwnerAndGroup(0, 0),
						h.IsJSON(),

						withContents(func(data []byte) {
							manifest := v1.Manifest{}
							err := json.Unmarshal(data, &manifest)
							h.AssertNil(t, err)

							// config: application/vnd.docker.container.image.v1+json
							h.AssertOnTarEntry(t, outputFile,
								"/blobs/sha256/"+manifest.Config.Digest.Hex(),
								h.HasOwnerAndGroup(0, 0),
								h.IsJSON(),
								// buildpackage metadata
								h.ContentContains(`"io.buildpacks.buildpackage.metadata":"{\"id\":\"bp.1.id\",\"version\":\"bp.1.version\",\"stacks\":[{\"id\":\"stack.id.1\"},{\"id\":\"stack.id.2\"}]}"`),
								// buildpack layers metadata
								h.ContentContains(`"io.buildpacks.buildpack.layers":"{\"bp.1.id\":{\"bp.1.version\":{\"api\":\"0.2\",\"stacks\":[{\"id\":\"stack.id.1\"},{\"id\":\"stack.id.2\"}],\"layerDiffID\":\"sha256:44447e95b06b73496d1891de5afb01936e9999b97ea03dad6337d9f5610807a7\"}}`),
								// image os
								h.ContentContains(`"os":"linux"`),
								// custom labels
								h.ContentContains(`"test.label.one":"1"`),
								h.ContentContains(`"test.label.two":"2"`),
							)
						}))
				}))
		})

		it("adds buildpack layers", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				WithAPI:    api.MustParse("0.2"),
				WithInfo:   dist.ModuleInfo{ID: "bp.1.id", Version: "bp.1.version"},
				WithStacks: []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}},
				WithOrder:  nil,
			}, 0644)
			h.AssertNil(t, err)

			builder := buildpack.NewBuilder(mockImageFactory(""))
			builder.SetBuildpack(buildpack1)

			outputFile := filepath.Join(tmpDir, fmt.Sprintf("package-%s.cnb", h.RandString(10)))
			h.AssertNil(t, builder.SaveAsFile(outputFile, dist.Target{OS: "linux"}, map[string]string{}))

			h.AssertOnTarEntry(t, outputFile, "/blobs",
				h.IsDirectory(),
				h.HasOwnerAndGroup(0, 0),
				h.HasFileMode(0755))
			h.AssertOnTarEntry(t, outputFile, "/blobs/sha256",
				h.IsDirectory(),
				h.HasOwnerAndGroup(0, 0),
				h.HasFileMode(0755))

			bpReader, err := buildpack1.Open()
			h.AssertNil(t, err)
			defer bpReader.Close()

			// layer: application/vnd.docker.image.rootfs.diff.tar.gzip
			buildpackLayerSHA, err := computeLayerSHA(bpReader)
			h.AssertNil(t, err)
			h.AssertOnTarEntry(t, outputFile,
				"/blobs/sha256/"+buildpackLayerSHA,
				h.HasOwnerAndGroup(0, 0),
				h.HasFileMode(0755),
				h.IsGzipped(),
				h.AssertOnNestedTar("/cnb/buildpacks/bp.1.id",
					h.IsDirectory(),
					h.HasOwnerAndGroup(0, 0),
					h.HasFileMode(0644)),
				h.AssertOnNestedTar("/cnb/buildpacks/bp.1.id/bp.1.version/bin/build",
					h.ContentEquals("build-contents"),
					h.HasOwnerAndGroup(0, 0),
					h.HasFileMode(0644)),
				h.AssertOnNestedTar("/cnb/buildpacks/bp.1.id/bp.1.version/bin/detect",
					h.ContentEquals("detect-contents"),
					h.HasOwnerAndGroup(0, 0),
					h.HasFileMode(0644)))
		})

		it("adds baselayer + buildpack layers for windows", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				WithAPI:    api.MustParse("0.2"),
				WithInfo:   dist.ModuleInfo{ID: "bp.1.id", Version: "bp.1.version"},
				WithStacks: []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}},
				WithOrder:  nil,
			}, 0644)
			h.AssertNil(t, err)

			builder := buildpack.NewBuilder(mockImageFactory(""))
			builder.SetBuildpack(buildpack1)

			outputFile := filepath.Join(tmpDir, fmt.Sprintf("package-%s.cnb", h.RandString(10)))
			h.AssertNil(t, builder.SaveAsFile(outputFile, dist.Target{OS: "windows"}, map[string]string{}))

			// Windows baselayer content is constant
			expectedBaseLayerReader, err := layer.WindowsBaseLayer()
			h.AssertNil(t, err)

			// layer: application/vnd.docker.image.rootfs.diff.tar.gzip
			expectedBaseLayerSHA, err := computeLayerSHA(io.NopCloser(expectedBaseLayerReader))
			h.AssertNil(t, err)
			h.AssertOnTarEntry(t, outputFile,
				"/blobs/sha256/"+expectedBaseLayerSHA,
				h.HasOwnerAndGroup(0, 0),
				h.HasFileMode(0755),
				h.IsGzipped(),
			)

			bpReader, err := buildpack1.Open()
			h.AssertNil(t, err)
			defer bpReader.Close()

			buildpackLayerSHA, err := computeLayerSHA(bpReader)
			h.AssertNil(t, err)
			h.AssertOnTarEntry(t, outputFile,
				"/blobs/sha256/"+buildpackLayerSHA,
				h.HasOwnerAndGroup(0, 0),
				h.HasFileMode(0755),
				h.IsGzipped(),
			)
		})
	})
}

func computeLayerSHA(reader io.ReadCloser) (string, error) {
	bpLayer := stream.NewLayer(reader, stream.WithCompressionLevel(gzip.DefaultCompression))
	compressed, err := bpLayer.Compressed()
	if err != nil {
		return "", err
	}
	defer compressed.Close()

	if _, err := io.Copy(io.Discard, compressed); err != nil {
		return "", err
	}

	digest, err := bpLayer.Digest()
	if err != nil {
		return "", err
	}

	return digest.Hex, nil
}

type imageWithLabelError struct {
	*fakes.Image
}

func (i *imageWithLabelError) SetLabel(string, string) error {
	return errors.New("Label could not be set")
}
