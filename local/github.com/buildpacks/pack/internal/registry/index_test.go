package registry_test

import (
	"path/filepath"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/registry"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestIndex(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Index", testIndex, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testIndex(t *testing.T, when spec.G, it spec.S) {
	when("#IndexPath", func() {
		when("valid", func() {
			for _, scenario := range []struct {
				desc,
				root,
				ns,
				name,
				expectation string
			}{
				{
					desc:        "1 char name",
					root:        "/tmp",
					ns:          "acme",
					name:        "a",
					expectation: filepath.Join("/tmp", "1", "acme_a"),
				},
				{
					desc:        "2 char name",
					root:        "/tmp",
					ns:          "acme",
					name:        "ab",
					expectation: filepath.Join("/tmp", "2", "acme_ab"),
				},
				{
					desc:        "3 char name",
					root:        "/tmp",
					ns:          "acme",
					name:        "abc",
					expectation: filepath.Join("/tmp", "3", "ab", "acme_abc"),
				},
				{
					desc:        "4 char name",
					root:        "/tmp",
					ns:          "acme",
					name:        "abcd",
					expectation: filepath.Join("/tmp", "ab", "cd", "acme_abcd"),
				},
				{
					desc:        "> 4 char name",
					root:        "/tmp",
					ns:          "acme",
					name:        "acmelang",
					expectation: filepath.Join("/tmp", "ac", "me", "acme_acmelang"),
				},
			} {
				scenario := scenario
				it(scenario.desc, func() {
					index, err := registry.IndexPath(scenario.root, scenario.ns, scenario.name)
					h.AssertNil(t, err)
					h.AssertEq(t, index, scenario.expectation)
				})
			}
		})

		when("invalid", func() {
			for _, scenario := range []struct {
				desc,
				ns,
				name,
				error string
			}{
				{
					desc:  "ns is empty",
					ns:    "",
					name:  "a",
					error: "'namespace' cannot be empty",
				},
				{
					desc:  "name is empty",
					ns:    "a",
					name:  "",
					error: "'name' cannot be empty",
				},
				{
					desc:  "namespace has capital letters",
					ns:    "Acme",
					name:  "buildpack",
					error: "'namespace' contains illegal characters (must match '[a-z0-9\\-.]+')",
				},
				{
					desc:  "name has capital letters",
					ns:    "acme",
					name:  "Buildpack",
					error: "'name' contains illegal characters (must match '[a-z0-9\\-.]+')",
				},
				{
					desc:  "namespace is too long",
					ns:    h.RandString(254),
					name:  "buildpack",
					error: "'namespace' too long (max 253 chars)",
				},
				{
					desc:  "name is too long",
					ns:    "acme",
					name:  h.RandString(254),
					error: "'name' too long (max 253 chars)",
				},
			} {
				scenario := scenario
				when(scenario.desc, func() {
					it("errors", func() {
						_, err := registry.IndexPath("/tmp", scenario.ns, scenario.name)
						h.AssertError(t, err, scenario.error)
					})
				})
			}
		})
	})
}
