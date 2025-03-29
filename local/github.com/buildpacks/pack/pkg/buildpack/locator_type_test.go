package buildpack_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestGetLocatorType(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testGetLocatorType", testGetLocatorType, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testGetLocatorType(t *testing.T, when spec.G, it spec.S) {
	type testCase struct {
		locator      string
		builderBPs   []dist.ModuleInfo
		expectedType buildpack.LocatorType
		expectedErr  string
	}

	var localPath = func(path string) string {
		return filepath.Join("testdata", path)
	}

	for _, tc := range []testCase{
		{
			locator:      "from=builder",
			expectedType: buildpack.FromBuilderLocator,
		},
		{
			locator:      "from=builder:some-bp",
			builderBPs:   []dist.ModuleInfo{{ID: "some-bp", Version: "some-version"}},
			expectedType: buildpack.IDLocator,
		},
		{
			locator:     "from=builder:some-bp",
			expectedErr: "'from=builder:some-bp' is not a valid identifier",
		},
		{
			locator:     "from=builder:some-bp@some-other-version",
			builderBPs:  []dist.ModuleInfo{{ID: "some-bp", Version: "some-version"}},
			expectedErr: "'from=builder:some-bp@some-other-version' is not a valid identifier",
		},
		{
			locator:      "urn:cnb:builder:some-bp",
			builderBPs:   []dist.ModuleInfo{{ID: "some-bp", Version: "some-version"}},
			expectedType: buildpack.IDLocator,
		},
		{
			locator:     "urn:cnb:builder:some-bp",
			expectedErr: "'urn:cnb:builder:some-bp' is not a valid identifier",
		},
		{
			locator:     "urn:cnb:builder:some-bp@some-other-version",
			builderBPs:  []dist.ModuleInfo{{ID: "some-bp", Version: "some-version"}},
			expectedErr: "'urn:cnb:builder:some-bp@some-other-version' is not a valid identifier",
		},
		{
			locator:      "some-bp",
			builderBPs:   []dist.ModuleInfo{{ID: "some-bp", Version: "any-version"}},
			expectedType: buildpack.IDLocator,
		},
		{
			locator:      localPath("buildpack"),
			builderBPs:   []dist.ModuleInfo{{ID: "bp.one", Version: "1.2.3"}},
			expectedType: buildpack.URILocator,
		},
		{
			locator:      "https://example.com/buildpack.tgz",
			expectedType: buildpack.URILocator,
		},
		{
			locator:      "localhost:1234/example/package-cnb",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "cnbs/some-bp:latest",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "docker://cnbs/some-bp",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "docker://cnbs/some-bp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "docker://cnbs/some-bp:some-tag",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "docker://cnbs/some-bp:some-tag@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "docker://registry.com/cnbs/some-bp",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "docker://registry.com/cnbs/some-bp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "docker://registry.com/cnbs/some-bp:some-tag",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "docker://registry.com/cnbs/some-bp:some-tag@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "cnbs/some-bp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "cnbs/some-bp:some-tag@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "registry.com/cnbs/some-bp",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "registry.com/cnbs/some-bp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "registry.com/cnbs/some-bp:some-tag",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "registry.com/cnbs/some-bp:some-tag@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expectedType: buildpack.PackageLocator,
		},
		{
			locator:      "urn:cnb:registry:example/foo@1.0.0",
			expectedType: buildpack.RegistryLocator,
		},
		{
			locator:      "example/foo@1.0.0",
			expectedType: buildpack.RegistryLocator,
		},
		{
			locator:      "example/registry-cnb",
			expectedType: buildpack.RegistryLocator,
		},
		{
			locator:      "cnbs/sample-package@hello-universe",
			expectedType: buildpack.InvalidLocator,
		},
		{
			locator:      "dev.local/http-go-fn:latest",
			expectedType: buildpack.PackageLocator,
		},
	} {
		tc := tc

		desc := fmt.Sprintf("locator is %s", tc.locator)
		if len(tc.builderBPs) > 0 {
			var names []string
			for _, bp := range tc.builderBPs {
				names = append(names, bp.FullName())
			}
			desc += fmt.Sprintf(" and builder has buildpacks %s", names)
		}

		when(desc, func() {
			it(fmt.Sprintf("should return %s", tc.expectedType), func() {
				actualType, actualErr := buildpack.GetLocatorType(tc.locator, "", tc.builderBPs)

				if tc.expectedErr == "" {
					h.AssertNil(t, actualErr)
				} else {
					h.AssertError(t, actualErr, tc.expectedErr)
				}

				h.AssertEq(t, actualType, tc.expectedType)
			})
		})
	}
}
