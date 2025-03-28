package buildpack_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/fakes"
	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/blob"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestOCILayoutPackage(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Extract", testOCILayoutPackage, spec.Parallel(), spec.Report(report.Terminal{}))
}

type testCase struct {
	mediatype string
	file      string
}

func testOCILayoutPackage(t *testing.T, when spec.G, it spec.S) {
	when("#BuildpacksFromOCILayoutBlob", func() {
		for _, test := range []testCase{
			{
				mediatype: "application/vnd.docker.distribution.manifest.v2+json",
				file:      "hello-universe.cnb",
			},
			{
				mediatype: v1.MediaTypeImageManifest,
				file:      "hello-universe-oci.cnb",
			},
		} {
			it(fmt.Sprintf("extracts buildpacks, media type: %s", test.mediatype), func() {
				mainBP, depBPs, err := buildpack.BuildpacksFromOCILayoutBlob(blob.NewBlob(filepath.Join("testdata", test.file)))
				h.AssertNil(t, err)

				h.AssertEq(t, mainBP.Descriptor().Info().ID, "io.buildpacks.samples.hello-universe")
				h.AssertEq(t, mainBP.Descriptor().Info().Version, "0.0.1")
				h.AssertEq(t, len(depBPs), 2)
			})

			it(fmt.Sprintf("provides readable blobs, media type: %s", test.mediatype), func() {
				mainBP, depBPs, err := buildpack.BuildpacksFromOCILayoutBlob(blob.NewBlob(filepath.Join("testdata", test.file)))
				h.AssertNil(t, err)

				for _, bp := range append([]buildpack.BuildModule{mainBP}, depBPs...) {
					reader, err := bp.Open()
					h.AssertNil(t, err)

					_, contents, err := archive.ReadTarEntry(
						reader,
						fmt.Sprintf("/cnb/buildpacks/%s/%s/buildpack.toml",
							bp.Descriptor().Info().ID,
							bp.Descriptor().Info().Version,
						),
					)
					h.AssertNil(t, err)
					h.AssertContains(t, string(contents), bp.Descriptor().Info().ID)
					h.AssertContains(t, string(contents), bp.Descriptor().Info().Version)
				}
			})
		}
	})

	when("#ExtensionsFromOCILayoutBlob", func() {
		it("extracts buildpacks", func() {
			ext, err := buildpack.ExtensionsFromOCILayoutBlob(blob.NewBlob(filepath.Join("testdata", "tree-extension.cnb")))
			h.AssertNil(t, err)

			h.AssertEq(t, ext.Descriptor().Info().ID, "samples-tree")
			h.AssertEq(t, ext.Descriptor().Info().Version, "0.0.1")
		})

		it("provides readable blobs", func() {
			ext, err := buildpack.ExtensionsFromOCILayoutBlob(blob.NewBlob(filepath.Join("testdata", "tree-extension.cnb")))
			h.AssertNil(t, err)
			reader, err := ext.Open()
			h.AssertNil(t, err)

			_, contents, err := archive.ReadTarEntry(
				reader,
				fmt.Sprintf("/cnb/extensions/%s/%s/extension.toml",
					ext.Descriptor().Info().ID,
					ext.Descriptor().Info().Version,
				),
			)
			h.AssertNil(t, err)
			h.AssertContains(t, string(contents), ext.Descriptor().Info().ID)
			h.AssertContains(t, string(contents), ext.Descriptor().Info().Version)
		})
	})

	when("#IsOCILayoutBlob", func() {
		when("is an OCI layout blob", func() {
			it("returns true", func() {
				isOCILayoutBlob, err := buildpack.IsOCILayoutBlob(blob.NewBlob(filepath.Join("testdata", "hello-universe.cnb")))
				h.AssertNil(t, err)
				h.AssertEq(t, isOCILayoutBlob, true)
			})
		})

		when("is NOT an OCI layout blob", func() {
			it("returns false", func() {
				buildpackBlob, err := fakes.NewFakeBuildpackBlob(&dist.BuildpackDescriptor{
					WithAPI: api.MustParse("0.3"),
					WithInfo: dist.ModuleInfo{
						ID:      "bp.id",
						Version: "bp.version",
					},
					WithStacks: []dist.Stack{{}},
					WithOrder:  nil,
				}, 0755)
				h.AssertNil(t, err)

				isOCILayoutBlob, err := buildpack.IsOCILayoutBlob(buildpackBlob)
				h.AssertNil(t, err)
				h.AssertEq(t, isOCILayoutBlob, false)
			})
		})
	})
}
