package client

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	h "github.com/buildpacks/pack/testhelpers"
)

func TestInputImageReference(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "InputImageReference", testInputImageReference, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testInputImageReference(t *testing.T, when spec.G, it spec.S) {
	var defaultImageReference, layoutImageReference InputImageReference

	it.Before(func() {
		defaultImageReference = ParseInputImageReference("busybox")
		layoutImageReference = ParseInputImageReference("oci:my-app")
	})

	when("#ParseInputImageReference", func() {
		when("oci layout image reference is not provided", func() {
			it("default implementation is returned", func() {
				h.AssertEq(t, defaultImageReference.Layout(), false)
				h.AssertEq(t, defaultImageReference.Name(), "busybox")

				fullName, err := defaultImageReference.FullName()
				h.AssertNil(t, err)
				h.AssertEq(t, fullName, "busybox")
			})
		})

		when("oci layout image reference is provided", func() {
			it("layout implementation is returned", func() {
				h.AssertTrue(t, layoutImageReference.Layout())
				h.AssertEq(t, layoutImageReference.Name(), "my-app")
			})
		})
	})

	when("#FullName", func() {
		when("oci layout image reference is provided", func() {
			when("not absolute path provided", func() {
				it("it will be joined with the current working directory", func() {
					fullPath, err := layoutImageReference.FullName()
					h.AssertNil(t, err)

					currentWorkingDir, err := os.Getwd()
					h.AssertNil(t, err)

					expectedPath := filepath.Join(currentWorkingDir, layoutImageReference.Name())
					h.AssertEq(t, fullPath, expectedPath)
				})
			})

			when("absolute path provided", func() {
				var (
					fullPath, expectedFullPath, tmpDir string
					err                                error
				)

				it.Before(func() {
					tmpDir, err = os.MkdirTemp("", "pack.input.image.reference.test")
					h.AssertNil(t, err)
					expectedFullPath = filepath.Join(tmpDir, "my-app")
					layoutImageReference = ParseInputImageReference(fmt.Sprintf("oci:%s", expectedFullPath))
				})

				it.After(func() {
					err = os.RemoveAll(tmpDir)
					h.AssertNil(t, err)
				})

				it("it must returned the path provided", func() {
					fullPath, err = layoutImageReference.FullName()
					h.AssertNil(t, err)
					h.AssertEq(t, fullPath, expectedFullPath)
				})
			})
		})
	})
}
