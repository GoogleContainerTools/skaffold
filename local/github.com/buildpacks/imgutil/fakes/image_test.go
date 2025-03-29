package fakes_test

import (
	"archive/tar"
	"fmt"

	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/fakes"
	h "github.com/buildpacks/imgutil/testhelpers"
)

var localTestRegistry *h.DockerRegistry

func newRepoName() string {
	return "test-image-" + h.RandString(10)
}

func TestFake(t *testing.T) {
	localTestRegistry = h.NewDockerRegistry()
	localTestRegistry.Start(t)
	defer localTestRegistry.Stop(t)

	spec.Run(t, "FakeImage", testFake, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testFake(t *testing.T, when spec.G, it spec.S) {
	it("implements imgutil.Image", func() {
		var _ imgutil.Image = fakes.NewImage("", "", nil)
	})

	when("#SavedNames", func() {
		when("additional names are provided during save", func() {
			var (
				repoName        = newRepoName()
				additionalNames = []string{
					newRepoName(),
					newRepoName(),
				}
			)

			it("returns list of saved names", func() {
				image := fakes.NewImage(repoName, "", nil)

				_ = image.Save(additionalNames...)

				names := image.SavedNames()
				h.AssertContains(t, names, append(additionalNames, repoName)...)
			})

			when("an image name is not valid", func() {
				it("returns a list of image names with errors", func() {
					badImageName := repoName + ":🧨"

					image := fakes.NewImage(repoName, "", nil)

					err := image.Save(append([]string{badImageName}, additionalNames...)...)
					saveErr, ok := err.(imgutil.SaveError)
					h.AssertEq(t, ok, true)
					h.AssertEq(t, len(saveErr.Errors), 1)
					h.AssertEq(t, saveErr.Errors[0].ImageName, badImageName)
					h.AssertError(t, saveErr.Errors[0].Cause, "could not parse reference")

					names := image.SavedNames()
					h.AssertContains(t, names, append(additionalNames, repoName)...)
					h.AssertDoesNotContain(t, names, badImageName)
				})
			})
		})
	})

	when("#FindLayerWithPath", func() {
		var (
			image      *fakes.Image
			layer1Path string
			layer2Path string
		)

		it.Before(func() {
			var err error

			image = fakes.NewImage("some-image", "", nil)

			layer1Path, err = createLayerTar(map[string]string{})
			h.AssertNil(t, err)

			err = image.AddLayer(layer1Path)
			h.AssertNil(t, err)

			layer2Path, err = createLayerTar(map[string]string{
				"/layer2/file1":     "file-1-contents",
				"/layer2/file2":     "file-2-contents",
				"/layer2/some.toml": "[[something]]",
			})
			h.AssertNil(t, err)

			err = image.AddLayer(layer2Path)
			h.AssertNil(t, err)
		})

		it.After(func() {
			os.RemoveAll(layer1Path)
			os.RemoveAll(layer2Path)
		})

		when("path not found in image", func() {
			it("should list out contents", func() {
				_, err := image.FindLayerWithPath("/non-existent/file")

				h.AssertError(t, err, fmt.Sprintf(`could not find '/non-existent/file' in any layer.

Layers
-------
%s
  (empty)

%s
  - [F] /layer2/file1
  - [F] /layer2/file2
  - [F] /layer2/some.toml
`,
					filepath.Base(layer1Path),
					filepath.Base(layer2Path)),
				)
			})
		})
	})

	when("#AnnotateRefName", func() {
		var repoName = newRepoName()

		it("adds org.opencontainers.image.ref.name annotation", func() {
			image := fakes.NewImage(repoName, "", nil)
			image.AnnotateRefName("my-tag")

			_ = image.Save()

			annotations := image.SavedAnnotations()
			refName, _ := image.GetAnnotateRefName()
			h.AssertEq(t, annotations["org.opencontainers.image.ref.name"], refName)
		})
	})
}

func createLayerTar(contents map[string]string) (string, error) {
	file, err := os.CreateTemp("", "layer-*.tar")
	if err != nil {
		return "", nil
	}
	defer file.Close()

	tw := tar.NewWriter(file)

	var paths []string
	for k := range contents {
		paths = append(paths, k)
	}
	sort.Strings(paths)

	for _, path := range paths {
		txt := contents[path]

		if err := tw.WriteHeader(&tar.Header{Name: path, Size: int64(len(txt)), Mode: 0644}); err != nil {
			return "", err
		}
		if _, err := tw.Write([]byte(txt)); err != nil {
			return "", err
		}
	}

	if err := tw.Close(); err != nil {
		return "", err
	}

	return file.Name(), nil
}
