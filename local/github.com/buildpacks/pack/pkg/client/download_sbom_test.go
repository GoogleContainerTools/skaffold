package client

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/imgutil/fakes"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/archive"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestDownloadSBOM(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "DownloadSBOM", testDownloadSBOM, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testDownloadSBOM(t *testing.T, when spec.G, it spec.S) {
	var (
		subject          *Client
		mockImageFetcher *testmocks.MockImageFetcher
		mockDockerClient *testmocks.MockCommonAPIClient
		mockController   *gomock.Controller
		out              bytes.Buffer
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockImageFetcher = testmocks.NewMockImageFetcher(mockController)
		mockDockerClient = testmocks.NewMockCommonAPIClient(mockController)

		var err error
		subject, err = NewClient(WithLogger(logging.NewLogWithWriters(&out, &out)), WithFetcher(mockImageFetcher), WithDockerClient(mockDockerClient))
		h.AssertNil(t, err)
	})

	it.After(func() {
		mockController.Finish()
	})

	when("the image exists", func() {
		var (
			mockImage *testmocks.MockImage
			tmpDir    string
			tmpFile   string
		)

		it.Before(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "pack.download.sbom.test.")
			h.AssertNil(t, err)

			f, err := os.CreateTemp("", "pack.download.sbom.test.")
			h.AssertNil(t, err)
			tmpFile = f.Name()

			err = archive.CreateSingleFileTar(tmpFile, "sbom", "some-sbom-content")
			h.AssertNil(t, err)

			data, err := os.ReadFile(tmpFile)
			h.AssertNil(t, err)

			hsh := sha256.New()
			hsh.Write(data)
			shasum := hex.EncodeToString(hsh.Sum(nil))

			mockImage = testmocks.NewImage("some/image", "", nil)
			mockImage.AddLayerWithDiffID(tmpFile, fmt.Sprintf("sha256:%s", shasum))
			h.AssertNil(t, mockImage.SetLabel(
				"io.buildpacks.lifecycle.metadata",
				fmt.Sprintf(
					`{
  "sbom": {
    "sha": "sha256:%s"
  }
}`, shasum)))

			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/image", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(mockImage, nil)
		})

		it.After(func() {
			os.RemoveAll(tmpDir)
			os.RemoveAll(tmpFile)
		})

		it("returns the stack ID", func() {
			err := subject.DownloadSBOM("some/image", DownloadSBOMOptions{Daemon: true, DestinationDir: tmpDir})
			h.AssertNil(t, err)

			contents, err := os.ReadFile(filepath.Join(tmpDir, "sbom"))
			h.AssertNil(t, err)

			h.AssertEq(t, string(contents), "some-sbom-content")
		})
	})

	when("the image doesn't exist", func() {
		it("returns nil", func() {
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/non-existent-image", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(nil, image.ErrNotFound)

			err := subject.DownloadSBOM("some/non-existent-image", DownloadSBOMOptions{Daemon: true, DestinationDir: ""})
			expectedError := fmt.Sprintf("image '%s' cannot be found", "some/non-existent-image")
			h.AssertError(t, err, expectedError)

			expectedMessage := fmt.Sprintf("Warning: if the image is saved on a registry run with the flag '--remote', for example: 'pack sbom download --remote %s'", "some/non-existent-image")
			h.AssertContains(t, out.String(), expectedMessage)
		})
	})

	when("there is an error fetching the image", func() {
		it("returns the error", func() {
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/image", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).Return(nil, errors.New("some-error"))

			err := subject.DownloadSBOM("some/image", DownloadSBOMOptions{Daemon: true, DestinationDir: ""})
			h.AssertError(t, err, "some-error")
		})
	})

	when("the image is SBOM metadata", func() {
		it("returns empty data", func() {
			mockImageFetcher.EXPECT().
				Fetch(gomock.Any(), "some/image-without-labels", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).
				Return(fakes.NewImage("some/image-without-labels", "", nil), nil)

			err := subject.DownloadSBOM("some/image-without-labels", DownloadSBOMOptions{Daemon: true, DestinationDir: ""})
			h.AssertError(t, err, "could not find SBoM information on 'some/image-without-labels'")
		})
	})

	when("the image has malformed metadata", func() {
		var badImage *fakes.Image

		it.Before(func() {
			badImage = fakes.NewImage("some/image-with-malformed-metadata", "", nil)
			mockImageFetcher.EXPECT().
				Fetch(gomock.Any(), "some/image-with-malformed-metadata", image.FetchOptions{Daemon: true, PullPolicy: image.PullNever}).
				Return(badImage, nil)
		})

		it("returns an error when layers md cannot parse", func() {
			h.AssertNil(t, badImage.SetLabel("io.buildpacks.lifecycle.metadata", "not   ----  json"))

			err := subject.DownloadSBOM("some/image-with-malformed-metadata", DownloadSBOMOptions{Daemon: true, DestinationDir: ""})
			h.AssertError(t, err, "unmarshalling label 'io.buildpacks.lifecycle.metadata'")
		})
	})
}
