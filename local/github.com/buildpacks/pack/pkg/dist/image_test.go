package dist_test

import (
	"testing"

	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/builder/fakes"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestImage(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testImage", testImage, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testImage(t *testing.T, when spec.G, it spec.S) {
	when("A label needs to be get", func() {
		it("sets a label successfully", func() {
			var outputLabel bool
			mockInspectable := fakes.FakeInspectable{ReturnForLabel: "true", ErrorForLabel: nil}

			isPresent, err := dist.GetLabel(&mockInspectable, "random-label", &outputLabel)

			h.AssertNil(t, err)
			h.AssertEq(t, isPresent, true)
			h.AssertEq(t, outputLabel, true)
		})

		it("returns an error", func() {
			var outputLabel bool
			mockInspectable := fakes.FakeInspectable{ReturnForLabel: "", ErrorForLabel: errors.New("random-error")}

			isPresent, err := dist.GetLabel(&mockInspectable, "random-label", &outputLabel)

			h.AssertNotNil(t, err)
			h.AssertEq(t, isPresent, false)
			h.AssertEq(t, outputLabel, false)
		})
	})

	when("Try to get an empty label", func() {
		it("returns isPresent but it doesn't set the label", func() {
			var outputLabel bool
			mockInspectable := fakes.FakeInspectable{ReturnForLabel: "", ErrorForLabel: nil}

			isPresent, err := dist.GetLabel(&mockInspectable, "random-label", &outputLabel)

			h.AssertNil(t, err)
			h.AssertEq(t, isPresent, false)
			h.AssertEq(t, outputLabel, false)
		})
	})
}
