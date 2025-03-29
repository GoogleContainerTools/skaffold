package image_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/image"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPullPolicy(t *testing.T) {
	spec.Run(t, "PullPolicy", testPullPolicy, spec.Report(report.Terminal{}))
}

func testPullPolicy(t *testing.T, when spec.G, it spec.S) {
	when("#ParsePullPolicy", func() {
		it("returns PullNever for never", func() {
			policy, err := image.ParsePullPolicy("never")
			h.AssertNil(t, err)
			h.AssertEq(t, policy, image.PullNever)
		})

		it("returns PullAlways for always", func() {
			policy, err := image.ParsePullPolicy("always")
			h.AssertNil(t, err)
			h.AssertEq(t, policy, image.PullAlways)
		})

		it("returns PullIfNotPresent for if-not-present", func() {
			policy, err := image.ParsePullPolicy("if-not-present")
			h.AssertNil(t, err)
			h.AssertEq(t, policy, image.PullIfNotPresent)
		})

		it("defaults to PullAlways, if empty string", func() {
			policy, err := image.ParsePullPolicy("")
			h.AssertNil(t, err)
			h.AssertEq(t, policy, image.PullAlways)
		})

		it("returns error for unknown string", func() {
			_, err := image.ParsePullPolicy("fake-policy-here")
			h.AssertError(t, err, "invalid pull policy")
		})
	})

	when("#String", func() {
		it("returns the right String value", func() {
			h.AssertEq(t, image.PullAlways.String(), "always")
			h.AssertEq(t, image.PullNever.String(), "never")
			h.AssertEq(t, image.PullIfNotPresent.String(), "if-not-present")
		})
	})
}
