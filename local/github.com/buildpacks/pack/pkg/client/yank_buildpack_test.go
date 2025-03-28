package client

import (
	"bytes"
	"testing"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	ifakes "github.com/buildpacks/pack/internal/fakes"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestYankBuildpack(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "yank_buildpack", testYankBuildpack, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testYankBuildpack(t *testing.T, when spec.G, it spec.S) {
	when("#YankBuildpack", func() {
		var (
			fakeImageFetcher *ifakes.FakeImageFetcher
			fakeAppImage     *fakes.Image
			subject          *Client
			out              bytes.Buffer
		)

		it.Before(func() {
			fakeImageFetcher = ifakes.NewFakeImageFetcher()
			fakeAppImage = fakes.NewImage("buildpack/image", "", &fakeIdentifier{name: "buildpack-image"})

			h.AssertNil(t, fakeAppImage.SetLabel("io.buildpacks.buildpackage.metadata",
				`{"id":"heroku/java-function","version":"1.1.1","stacks":[{"id":"heroku-18"},{"id":"io.buildpacks.stacks.jammy"},{"id":"org.cloudfoundry.stacks.cflinuxfs3"}]}`))
			fakeImageFetcher.RemoteImages["buildpack/image"] = fakeAppImage

			fakeLogger := logging.NewLogWithWriters(&out, &out)
			subject = &Client{
				logger:       fakeLogger,
				imageFetcher: fakeImageFetcher,
			}
		})

		it.After(func() {
			_ = fakeAppImage.Cleanup()
		})

		it("should return error for missing namespace id", func() {
			err := subject.YankBuildpack(YankBuildpackOptions{
				ID: "hello",
			})
			h.AssertError(t, err, "invalid id 'hello' does not contain a namespace")
		})

		it("should return error for invalid id", func() {
			err := subject.YankBuildpack(YankBuildpackOptions{
				ID: "bad/id/name",
			})
			h.AssertError(t, err, "invalid id 'bad/id/name' contains unexpected characters")
		})

		it("should return error when URL is missing", func() {
			err := subject.YankBuildpack(YankBuildpackOptions{
				ID:      "heroku/java",
				Version: "0.2.1",
				Type:    "github",
				URL:     "",
			})
			h.AssertError(t, err, "missing github URL")
		})

		it("should return error when URL is invalid", func() {
			err := subject.YankBuildpack(YankBuildpackOptions{
				ID:      "heroku/java",
				Version: "0.2.1",
				Type:    "github",
				URL:     "bad url",
			})
			h.AssertNotNil(t, err)
			h.AssertContains(t, err.Error(), "invalid URI for request")
		})
	})
}
