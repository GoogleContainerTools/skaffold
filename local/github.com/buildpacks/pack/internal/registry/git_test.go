package registry_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/registry"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestGit(t *testing.T) {
	spec.Run(t, "Git", testGit, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testGit(t *testing.T, when spec.G, it spec.S) {
	var (
		registryCache   registry.Cache
		tmpDir          string
		err             error
		registryFixture string
		outBuf          bytes.Buffer
		logger          logging.Logger
		username        = "supra08"
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)

		tmpDir, err = os.MkdirTemp("", "registry")
		h.AssertNil(t, err)

		registryFixture = h.CreateRegistryFixture(t, tmpDir, filepath.Join("..", "..", "testdata", "registry"))
		registryCache, err = registry.NewRegistryCache(logger, tmpDir, registryFixture)
		h.AssertNil(t, err)
	})

	it.After(func() {
		if runtime.GOOS != "windows" {
			h.AssertNil(t, os.RemoveAll(tmpDir))
		}
		os.RemoveAll(tmpDir)
	})

	when("#GitCommit", func() {
		when("ADD buildpack", func() {
			it("commits addition", func() {
				err := registry.GitCommit(registry.Buildpack{
					Namespace: "example",
					Name:      "python",
					Version:   "1.0.0",
					Yanked:    false,
					Address:   "example.com",
				}, username, registryCache)
				h.AssertNil(t, err)

				repo, err := git.PlainOpen(registryCache.Root)
				h.AssertNil(t, err)

				head, err := repo.Head()
				h.AssertNil(t, err)

				cIter, err := repo.Log(&git.LogOptions{From: head.Hash()})
				h.AssertNil(t, err)

				commit, err := cIter.Next()
				h.AssertNil(t, err)

				h.AssertEq(t, commit.Message, "ADD example/python@1.0.0")
			})
		})

		when("YANK buildpack", func() {
			it("commits yank", func() {
				err := registry.GitCommit(registry.Buildpack{
					Namespace: "example",
					Name:      "python",
					Version:   "1.0.0",
					Yanked:    true,
					Address:   "example.com",
				}, username, registryCache)
				h.AssertNil(t, err)

				repo, err := git.PlainOpen(registryCache.Root)
				h.AssertNil(t, err)

				head, err := repo.Head()
				h.AssertNil(t, err)

				cIter, err := repo.Log(&git.LogOptions{From: head.Hash()})
				h.AssertNil(t, err)

				commit, err := cIter.Next()
				h.AssertNil(t, err)

				h.AssertEq(t, commit.Message, "YANK example/python@1.0.0")
			})
		})
	})
}
