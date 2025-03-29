package target_test

import (
	"bytes"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/target"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestParseTargets(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ParseTargets", testParseTargets, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testParseTargets(t *testing.T, when spec.G, it spec.S) {
	outBuf := bytes.Buffer{}
	it.Before(func() {
		outBuf = bytes.Buffer{}
		h.AssertEq(t, outBuf.String(), "")
		var err error
		h.AssertNil(t, err)
	})

	when("target#ParseTarget", func() {
		it("should show a warn when [os][/arch][/variant] is nil", func() {
			target.ParseTarget(":distro@version", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotEq(t, outBuf.String(), "")
		})
		it("should parse target as expected", func() {
			output, err := target.ParseTarget("linux/arm/v6", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertEq(t, outBuf.String(), "")
			h.AssertNil(t, err)
			h.AssertEq(t, output, dist.Target{
				OS:          "linux",
				Arch:        "arm",
				ArchVariant: "v6",
			})
		})
		it("should return an error", func() {
			_, err := target.ParseTarget("", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotNil(t, err)
		})
		it("should log a warning when only [os] has typo or is unknown", func() {
			target.ParseTarget("os/arm/v6", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotEq(t, outBuf.String(), "")
		})
		it("should log a warning when only [arch] has typo or is unknown", func() {
			target.ParseTarget("darwin/arm/v6", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotEq(t, outBuf.String(), "")
		})
		it("should log a warning when only [variant] has typo or is unknown", func() {
			target.ParseTarget("linux/arm/unknown", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotEq(t, outBuf.String(), "")
		})
	})

	when("target#ParseTargets", func() {
		it("should throw an error when atleast one target throws error", func() {
			_, err := target.ParseTargets([]string{"linux/arm/v6", ":distro@version"}, logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotNil(t, err)
		})
		it("should parse targets as expected", func() {
			output, err := target.ParseTargets([]string{"linux/arm/v6", "linux/amd64:ubuntu@22.04;debian@8.10;debian@10.06"}, logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNil(t, err)
			h.AssertEq(t, output, []dist.Target{
				{
					OS:          "linux",
					Arch:        "arm",
					ArchVariant: "v6",
				},
				{
					OS:   "linux",
					Arch: "amd64",
					Distributions: []dist.Distribution{
						{
							Name:    "ubuntu",
							Version: "22.04",
						},
						{
							Name:    "debian",
							Version: "8.10",
						},
						{
							Name:    "debian",
							Version: "10.06",
						},
					},
				},
			})
		})
	})

	when("target#ParseDistro", func() {
		it("should parse distro as expected", func() {
			output, err := target.ParseDistro("ubuntu@22.04", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertEq(t, output, dist.Distribution{
				Name:    "ubuntu",
				Version: "22.04",
			})
			h.AssertNil(t, err)
		})
		it("should return an error when name is missing", func() {
			_, err := target.ParseDistro("@22.04@20.08", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotNil(t, err)
		})
		it("should return an error when there are two versions", func() {
			_, err := target.ParseDistro("some-distro@22.04@20.08", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "invalid distro")
		})
		it("should warn when distro version is not specified", func() {
			target.ParseDistro("ubuntu", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotEq(t, outBuf.String(), "")
		})
	})

	when("target#ParseDistros", func() {
		it("should parse distros as expected", func() {
			output, err := target.ParseDistros("ubuntu@22.04;ubuntu@20.08;debian@8.10;debian@10.06", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertEq(t, output, []dist.Distribution{
				{
					Name:    "ubuntu",
					Version: "22.04",
				},
				{
					Name:    "ubuntu",
					Version: "20.08",
				},
				{
					Name:    "debian",
					Version: "8.10",
				},
				{
					Name:    "debian",
					Version: "10.06",
				},
			})
			h.AssertNil(t, err)
		})
		it("result should be nil", func() {
			output, err := target.ParseDistros("", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertEq(t, output, []dist.Distribution(nil))
			h.AssertNil(t, err)
		})
		it("should return an error", func() {
			_, err := target.ParseDistros(";", logging.NewLogWithWriters(&outBuf, &outBuf))
			h.AssertNotNil(t, err)
		})
	})
}
