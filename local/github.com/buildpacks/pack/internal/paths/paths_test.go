package paths_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/paths"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestPaths(t *testing.T) {
	spec.Run(t, "Paths", testPaths, spec.Report(report.Terminal{}))
}

func testPaths(t *testing.T, when spec.G, it spec.S) {
	when("#IsURI", func() {
		for _, params := range []struct {
			desc    string
			uri     string
			isValid bool
		}{
			{
				desc:    "missing scheme",
				uri:     ":/invalid",
				isValid: false,
			},
			{
				desc:    "missing scheme",
				uri:     "://invalid",
				isValid: false,
			},
			{
				uri:     "file://host/file.txt",
				isValid: true,
			},
			{
				desc:    "no host (shorthand)",
				uri:     "file:/valid",
				isValid: true,
			},
			{
				desc:    "no host",
				uri:     "file:///valid",
				isValid: true,
			},
		} {
			params := params

			when(params.desc+":"+params.uri, func() {
				it(fmt.Sprintf("returns %v", params.isValid), func() {
					h.AssertEq(t, paths.IsURI(params.uri), params.isValid)
				})
			})
		}
	})

	when("#FilterReservedNames", func() {
		when("volume contains a reserved name", func() {
			it("modifies the volume name", func() {
				volumeName := "auxauxaux"
				subject := paths.FilterReservedNames(volumeName)
				expected := "a_u_xa_u_xa_u_x"
				if subject != expected {
					t.Fatalf("The volume should not contain reserved names")
				}
			})
		})

		when("volume does not contain reserved names", func() {
			it("does not modify the volume name", func() {
				volumeName := "lbtlbtlbt"
				subject := paths.FilterReservedNames(volumeName)
				if subject != volumeName {
					t.Fatalf("The volume should not be modified")
				}
			})
		})
	})

	when("#FilePathToURI", func() {
		when("is windows", func() {
			it.Before(func() {
				h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")
			})

			when("path is absolute", func() {
				it("returns uri", func() {
					uri, err := paths.FilePathToURI(`C:\some\file.txt`, "")
					h.AssertNil(t, err)
					h.AssertEq(t, uri, `file:///C:/some/file.txt`)
				})
			})

			when("path is relative", func() {
				var (
					err    error
					ogDir  string
					tmpDir string
				)
				it.Before(func() {
					ogDir, err = os.Getwd()
					h.AssertNil(t, err)

					tmpDir = os.TempDir()

					err = os.Chdir(tmpDir)
					h.AssertNil(t, err)
				})

				it.After(func() {
					err := os.Chdir(ogDir)
					h.AssertNil(t, err)
				})

				it("returns uri", func() {
					cwd, err := os.Getwd()
					h.AssertNil(t, err)

					uri, err := paths.FilePathToURI(`some\file.tgz`, "")
					h.AssertNil(t, err)

					h.AssertEq(t, uri, fmt.Sprintf(`file:///%s/some/file.tgz`, filepath.ToSlash(cwd)))
				})
			})
		})

		when("is *nix", func() {
			it.Before(func() {
				h.SkipIf(t, runtime.GOOS == "windows", "Skipped on windows")
			})

			when("path is absolute", func() {
				it("returns uri", func() {
					uri, err := paths.FilePathToURI("/tmp/file.tgz", "")
					h.AssertNil(t, err)
					h.AssertEq(t, uri, "file:///tmp/file.tgz")
				})
			})

			when("path is relative", func() {
				it("returns uri", func() {
					cwd, err := os.Getwd()
					h.AssertNil(t, err)

					uri, err := paths.FilePathToURI("some/file.tgz", "")
					h.AssertNil(t, err)

					h.AssertEq(t, uri, fmt.Sprintf("file://%s/some/file.tgz", cwd))
				})

				it("returns uri based on relativeTo", func() {
					uri, err := paths.FilePathToURI("some/file.tgz", "/my/base/dir")
					h.AssertNil(t, err)

					h.AssertEq(t, uri, "file:///my/base/dir/some/file.tgz")
				})
			})
		})
	})

	when("#URIToFilePath", func() {
		when("is windows", func() {
			when("uri is drive", func() {
				it("returns path", func() {
					h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")

					path, err := paths.URIToFilePath(`file:///c:/laptop/file.tgz`)
					h.AssertNil(t, err)

					h.AssertEq(t, path, `c:\laptop\file.tgz`)
				})
			})

			when("uri is network share", func() {
				it("returns path", func() {
					h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")

					path, err := paths.URIToFilePath(`file://laptop/file.tgz`)
					h.AssertNil(t, err)

					h.AssertEq(t, path, `\\laptop\file.tgz`)
				})
			})
		})

		when("is *nix", func() {
			when("uri is valid", func() {
				it("returns path", func() {
					h.SkipIf(t, runtime.GOOS == "windows", "Skipped on windows")

					path, err := paths.URIToFilePath(`file:///tmp/file.tgz`)
					h.AssertNil(t, err)

					h.AssertEq(t, path, `/tmp/file.tgz`)
				})
			})
		})
	})

	when("#WindowsDir", func() {
		it("returns the path directory", func() {
			path := paths.WindowsDir(`C:\layers\file.txt`)
			h.AssertEq(t, path, `C:\layers`)
		})

		it("returns empty for empty", func() {
			path := paths.WindowsBasename("")
			h.AssertEq(t, path, "")
		})
	})

	when("#WindowsBasename", func() {
		it("returns the path basename", func() {
			path := paths.WindowsBasename(`C:\layers\file.txt`)
			h.AssertEq(t, path, `file.txt`)
		})

		it("returns empty for empty", func() {
			path := paths.WindowsBasename("")
			h.AssertEq(t, path, "")
		})
	})

	when("#WindowsToSlash", func() {
		it("returns the path; backward slashes converted to forward with volume stripped ", func() {
			path := paths.WindowsToSlash(`C:\layers\file.txt`)
			h.AssertEq(t, path, `/layers/file.txt`)
		})

		it("returns / for volume", func() {
			path := paths.WindowsToSlash(`c:\`)
			h.AssertEq(t, path, `/`)
		})

		it("returns empty for empty", func() {
			path := paths.WindowsToSlash("")
			h.AssertEq(t, path, "")
		})
	})

	when("#WindowsPathSID", func() {
		when("UID and GID are both 0", func() {
			it(`returns the built-in BUILTIN\Administrators SID`, func() {
				sid := paths.WindowsPathSID(0, 0)
				h.AssertEq(t, sid, "S-1-5-32-544")
			})
		})

		when("UID and GID are both non-zero", func() {
			it(`returns the built-in BUILTIN\Users SID`, func() {
				sid := paths.WindowsPathSID(99, 99)
				h.AssertEq(t, sid, "S-1-5-32-545")
			})
		})
	})

	when("#CanonicalTarPath", func() {
		for _, params := range []struct {
			desc     string
			path     string
			expected string
		}{
			{
				desc:     "noop",
				path:     "my/clean/path",
				expected: "my/clean/path",
			},
			{
				desc:     "leading slash",
				path:     "/my/path",
				expected: "my/path",
			},
			{
				desc:     "dot",
				path:     "my/./path",
				expected: "my/path",
			},
			{
				desc:     "dotdot",
				path:     "my/../my/path",
				expected: "my/path",
			},
		} {
			params := params

			when(params.desc+":"+params.path, func() {
				it(fmt.Sprintf("returns %v", params.expected), func() {
					h.AssertEq(t, paths.CanonicalTarPath(params.path), params.expected)
				})
			})
		}
	})
}
