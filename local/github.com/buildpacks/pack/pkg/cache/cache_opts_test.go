package cache

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	h "github.com/buildpacks/pack/testhelpers"
)

type CacheOptTestCase struct {
	name       string
	input      string
	output     string
	shouldFail bool
}

func TestMetadata(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Metadata", testCacheOpts, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testCacheOpts(t *testing.T, when spec.G, it spec.S) {
	when("image cache format options are passed", func() {
		it("with complete options", func() {
			testcases := []CacheOptTestCase{
				{
					name:   "Build cache as Image",
					input:  "type=build;format=image;name=io.test.io/myorg/my-cache:build",
					output: "type=build;format=image;name=io.test.io/myorg/my-cache:build;type=launch;format=volume;",
				},
				{
					name:   "Launch cache as Image",
					input:  "type=launch;format=image;name=io.test.io/myorg/my-cache:build",
					output: "type=build;format=volume;type=launch;format=image;name=io.test.io/myorg/my-cache:build;",
				},
			}

			for _, testcase := range testcases {
				var cacheFlags CacheOpts
				t.Logf("Testing cache type: %s", testcase.name)
				err := cacheFlags.Set(testcase.input)
				h.AssertNil(t, err)
				h.AssertEq(t, testcase.output, cacheFlags.String())
			}
		})

		it("with missing options", func() {
			successTestCases := []CacheOptTestCase{
				{
					name:   "Build cache as Image missing: type",
					input:  "format=image;name=io.test.io/myorg/my-cache:build",
					output: "type=build;format=image;name=io.test.io/myorg/my-cache:build;type=launch;format=volume;",
				},
				{
					name:   "Build cache as Image missing: format",
					input:  "type=build;name=io.test.io/myorg/my-cache:build",
					output: "type=build;format=volume;name=io.test.io/myorg/my-cache:build;type=launch;format=volume;",
				},
				{
					name:       "Build cache as Image missing: name",
					input:      "type=build;format=image",
					output:     "cache 'name' is required",
					shouldFail: true,
				},
				{
					name:   "Build cache as Image missing: type, format",
					input:  "name=io.test.io/myorg/my-cache:build",
					output: "type=build;format=volume;name=io.test.io/myorg/my-cache:build;type=launch;format=volume;",
				},
				{
					name:   "Build cache as Image missing: format, name",
					input:  "type=build",
					output: "type=build;format=volume;type=launch;format=volume;",
				},
				{
					name:       "Build cache as Image missing: type, name",
					input:      "format=image",
					output:     "cache 'name' is required",
					shouldFail: true,
				},
				{
					name:       "Launch cache as Image missing: name",
					input:      "type=launch;format=image",
					output:     "cache 'name' is required",
					shouldFail: true,
				},
			}

			for _, testcase := range successTestCases {
				var cacheFlags CacheOpts
				t.Logf("Testing cache type: %s", testcase.name)
				err := cacheFlags.Set(testcase.input)

				if testcase.shouldFail {
					h.AssertError(t, err, testcase.output)
				} else {
					h.AssertNil(t, err)
					output := cacheFlags.String()
					h.AssertEq(t, testcase.output, output)
				}
			}
		})

		it("with invalid options", func() {
			testcases := []CacheOptTestCase{
				{
					name:       "Invalid cache type",
					input:      "type=invalid_cache;format=image;name=io.test.io/myorg/my-cache:build",
					output:     "invalid cache type 'invalid_cache'",
					shouldFail: true,
				},
				{
					name:       "Invalid cache format",
					input:      "type=launch;format=invalid_format;name=io.test.io/myorg/my-cache:build",
					output:     "invalid cache format 'invalid_format'",
					shouldFail: true,
				},
				{
					name:       "Not a key=value pair",
					input:      "launch;format=image;name=io.test.io/myorg/my-cache:build",
					output:     "invalid field 'launch' must be a key=value pair",
					shouldFail: true,
				},
				{
					name:       "Extra semicolon",
					input:      "type=launch;format=image;name=io.test.io/myorg/my-cache:build;",
					output:     "invalid field '' must be a key=value pair",
					shouldFail: true,
				},
			}

			for _, testcase := range testcases {
				var cacheFlags CacheOpts
				t.Logf("Testing cache type: %s", testcase.name)
				err := cacheFlags.Set(testcase.input)
				h.AssertError(t, err, testcase.output)
			}
		})
	})

	when("volume cache format options are passed", func() {
		it("with complete options", func() {
			testcases := []CacheOptTestCase{
				{
					name:   "Build cache as Volume",
					input:  "type=build;format=volume;name=test-build-volume-cache",
					output: "type=build;format=volume;name=test-build-volume-cache;type=launch;format=volume;",
				},
				{
					name:   "Launch cache as Volume",
					input:  "type=launch;format=volume;name=test-launch-volume-cache",
					output: "type=build;format=volume;type=launch;format=volume;name=test-launch-volume-cache;",
				},
			}

			for _, testcase := range testcases {
				var cacheFlags CacheOpts
				t.Logf("Testing cache type: %s", testcase.name)
				err := cacheFlags.Set(testcase.input)
				h.AssertNil(t, err)
				h.AssertEq(t, testcase.output, cacheFlags.String())
			}
		})

		it("with missing options", func() {
			successTestCases := []CacheOptTestCase{
				{
					name:   "Launch cache as Volume missing: format",
					input:  "type=launch;name=test-launch-volume",
					output: "type=build;format=volume;type=launch;format=volume;name=test-launch-volume;",
				},
				{
					name:   "Launch cache as Volume missing: name",
					input:  "type=launch;format=volume",
					output: "type=build;format=volume;type=launch;format=volume;",
				},
				{
					name:   "Launch cache as Volume missing: format, name",
					input:  "type=launch",
					output: "type=build;format=volume;type=launch;format=volume;",
				},
				{
					name:   "Launch cache as Volume missing: type, name",
					input:  "format=volume",
					output: "type=build;format=volume;type=launch;format=volume;",
				},
			}

			for _, testcase := range successTestCases {
				var cacheFlags CacheOpts
				t.Logf("Testing cache type: %s", testcase.name)
				err := cacheFlags.Set(testcase.input)

				if testcase.shouldFail {
					h.AssertError(t, err, testcase.output)
				} else {
					h.AssertNil(t, err)
					output := cacheFlags.String()
					h.AssertEq(t, testcase.output, output)
				}
			}
		})
	})

	when("bind cache format options are passed", func() {
		it("with complete options", func() {
			var testcases []CacheOptTestCase
			homeDir, err := os.UserHomeDir()
			h.AssertNil(t, err)
			cwd, err := os.Getwd()
			h.AssertNil(t, err)

			if runtime.GOOS != "windows" {
				testcases = []CacheOptTestCase{
					{
						name:   "Build cache as bind",
						input:  fmt.Sprintf("type=build;format=bind;source=%s/test-bind-build-cache", homeDir),
						output: fmt.Sprintf("type=build;format=bind;source=%s/test-bind-build-cache/build-cache;type=launch;format=volume;", homeDir),
					},
					{
						name:   "Build cache as bind with relative path",
						input:  "type=build;format=bind;source=./test-bind-build-cache-relative",
						output: fmt.Sprintf("type=build;format=bind;source=%s/test-bind-build-cache-relative/build-cache;type=launch;format=volume;", cwd),
					},
					{
						name:   "Launch cache as bind",
						input:  fmt.Sprintf("type=launch;format=bind;source=%s/test-bind-volume-cache", homeDir),
						output: fmt.Sprintf("type=build;format=volume;type=launch;format=bind;source=%s/test-bind-volume-cache/launch-cache;", homeDir),
					},
					{
						name:   "Case sensitivity test with uppercase path",
						input:  fmt.Sprintf("type=build;format=bind;source=%s/TestBindBuildCache", homeDir),
						output: fmt.Sprintf("type=build;format=bind;source=%s/TestBindBuildCache/build-cache;type=launch;format=volume;", homeDir),
					},
					{
						name:   "Case sensitivity test with mixed case path",
						input:  fmt.Sprintf("type=build;format=bind;source=%s/TeStBiNdBuildCaChe", homeDir),
						output: fmt.Sprintf("type=build;format=bind;source=%s/TeStBiNdBuildCaChe/build-cache;type=launch;format=volume;", homeDir),
					},
				}
			} else {
				testcases = []CacheOptTestCase{
					{
						name:   "Build cache as bind",
						input:  fmt.Sprintf("type=build;format=bind;source=%s\\test-bind-build-cache", homeDir),
						output: fmt.Sprintf("type=build;format=bind;source=%s\\test-bind-build-cache\\build-cache;type=launch;format=volume;", homeDir),
					},
					{
						name:   "Build cache as bind with relative path",
						input:  "type=build;format=bind;source=.\\test-bind-build-cache-relative",
						output: fmt.Sprintf("type=build;format=bind;source=%s\\test-bind-build-cache-relative\\build-cache;type=launch;format=volume;", cwd),
					},
					{
						name:   "Launch cache as bind",
						input:  fmt.Sprintf("type=launch;format=bind;source=%s\\test-bind-volume-cache", homeDir),
						output: fmt.Sprintf("type=build;format=volume;type=launch;format=bind;source=%s\\test-bind-volume-cache\\launch-cache;", homeDir),
					},
					// Case sensitivity test cases for Windows
					{
						name:   "Case sensitivity test with uppercase path",
						input:  fmt.Sprintf("type=build;format=bind;source=%s\\TestBindBuildCache", homeDir),
						output: fmt.Sprintf("type=build;format=bind;source=%s\\TestBindBuildCache\\build-cache;type=launch;format=volume;", homeDir),
					},
					{
						name:   "Case sensitivity test with mixed case path",
						input:  fmt.Sprintf("type=build;format=bind;source=%s\\TeStBiNdBuildCaChe", homeDir),
						output: fmt.Sprintf("type=build;format=bind;source=%s\\TeStBiNdBuildCaChe\\build-cache;type=launch;format=volume;", homeDir),
					},
				}
			}

			for _, testcase := range testcases {
				var cacheFlags CacheOpts
				t.Logf("Testing cache type: %s", testcase.name)
				err := cacheFlags.Set(testcase.input)
				h.AssertNil(t, err)
				h.AssertEq(t, strings.ToLower(testcase.output), strings.ToLower(cacheFlags.String()))
			}
		})

		it("with missing options", func() {
			successTestCases := []CacheOptTestCase{
				{
					name:       "Launch cache as bind missing: source",
					input:      "type=launch;format=bind",
					output:     "cache 'source' is required",
					shouldFail: true,
				},
				{
					name:       "Launch cache as Volume missing: type, source",
					input:      "format=bind",
					output:     "cache 'source' is required",
					shouldFail: true,
				},
			}

			for _, testcase := range successTestCases {
				var cacheFlags CacheOpts
				t.Logf("Testing cache type: %s", testcase.name)
				err := cacheFlags.Set(testcase.input)

				if testcase.shouldFail {
					h.AssertError(t, err, testcase.output)
				} else {
					h.AssertNil(t, err)
					output := cacheFlags.String()
					h.AssertEq(t, testcase.output, output)
				}
			}
		})
	})
}
