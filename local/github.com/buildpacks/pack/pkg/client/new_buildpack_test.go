package client_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/heroku/color"
	"github.com/pelletier/go-toml"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestNewBuildpack(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "NewBuildpack", testNewBuildpack, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testNewBuildpack(t *testing.T, when spec.G, it spec.S) {
	var (
		subject *client.Client
		tmpDir  string
	)

	it.Before(func() {
		var err error

		tmpDir, err = os.MkdirTemp("", "new-buildpack-test")
		h.AssertNil(t, err)

		subject, err = client.NewClient()
		h.AssertNil(t, err)
	})

	it.After(func() {
		h.AssertNil(t, os.RemoveAll(tmpDir))
	})

	when("#NewBuildpack", func() {
		it("should create bash scripts", func() {
			err := subject.NewBuildpack(context.TODO(), client.NewBuildpackOptions{
				API:     "0.4",
				Path:    tmpDir,
				ID:      "example/my-cnb",
				Version: "0.0.0",
				Stacks: []dist.Stack{
					{
						ID:     "some-stack",
						Mixins: []string{"some-mixin"},
					},
				},
			})
			h.AssertNil(t, err)

			info, err := os.Stat(filepath.Join(tmpDir, "bin/build"))
			h.AssertFalse(t, os.IsNotExist(err))
			if runtime.GOOS != "windows" {
				h.AssertTrue(t, info.Mode()&0100 != 0)
			}

			info, err = os.Stat(filepath.Join(tmpDir, "bin/detect"))
			h.AssertFalse(t, os.IsNotExist(err))
			if runtime.GOOS != "windows" {
				h.AssertTrue(t, info.Mode()&0100 != 0)
			}

			assertBuildpackToml(t, tmpDir, "example/my-cnb")
		})

		when("files exist", func() {
			it.Before(func() {
				var err error

				err = os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755)
				h.AssertNil(t, err)
				err = os.WriteFile(filepath.Join(tmpDir, "buildpack.toml"), []byte("expected value"), 0655)
				h.AssertNil(t, err)
				err = os.WriteFile(filepath.Join(tmpDir, "bin", "build"), []byte("expected value"), 0755)
				h.AssertNil(t, err)
				err = os.WriteFile(filepath.Join(tmpDir, "bin", "detect"), []byte("expected value"), 0755)
				h.AssertNil(t, err)
			})

			it("should not clobber files that exist", func() {
				err := subject.NewBuildpack(context.TODO(), client.NewBuildpackOptions{
					API:     "0.4",
					Path:    tmpDir,
					ID:      "example/my-cnb",
					Version: "0.0.0",
					Stacks: []dist.Stack{
						{
							ID:     "some-stack",
							Mixins: []string{"some-mixin"},
						},
					},
				})
				h.AssertNil(t, err)

				content, err := os.ReadFile(filepath.Join(tmpDir, "buildpack.toml"))
				h.AssertNil(t, err)
				h.AssertEq(t, content, []byte("expected value"))

				content, err = os.ReadFile(filepath.Join(tmpDir, "bin", "build"))
				h.AssertNil(t, err)
				h.AssertEq(t, content, []byte("expected value"))

				content, err = os.ReadFile(filepath.Join(tmpDir, "bin", "detect"))
				h.AssertNil(t, err)
				h.AssertEq(t, content, []byte("expected value"))
			})
		})
	})
}

func assertBuildpackToml(t *testing.T, path string, id string) {
	buildpackTOML := filepath.Join(path, "buildpack.toml")
	_, err := os.Stat(buildpackTOML)
	h.AssertFalse(t, os.IsNotExist(err))

	f, err := os.Open(buildpackTOML)
	h.AssertNil(t, err)
	var buildpackDescriptor dist.BuildpackDescriptor
	err = toml.NewDecoder(f).Decode(&buildpackDescriptor)
	h.AssertNil(t, err)
	defer f.Close()

	h.AssertEq(t, buildpackDescriptor.Info().ID, "example/my-cnb")
}
