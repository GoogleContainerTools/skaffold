package buildpack_test

import (
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/buildpack"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestParseName(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "ParseName", testParseName, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testParseName(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = h.NewAssertionManager(t)
	)

	when("#ParseIDLocator", func() {
		type testParams struct {
			desc            string
			locator         string
			expectedID      string
			expectedVersion string
		}

		for _, params := range []testParams{
			{
				desc:            "naked id+version",
				locator:         "ns/name@0.0.1",
				expectedID:      "ns/name",
				expectedVersion: "0.0.1",
			},
			{
				desc:            "naked only id",
				locator:         "ns/name",
				expectedID:      "ns/name",
				expectedVersion: "",
			},
			{
				desc:            "from=builder id+version",
				locator:         "from=builder:ns/name@1.2.3",
				expectedID:      "ns/name",
				expectedVersion: "1.2.3",
			},
			{
				desc:            "urn:cnb:builder id+version",
				locator:         "urn:cnb:builder:ns/name@1.2.3",
				expectedID:      "ns/name",
				expectedVersion: "1.2.3",
			},
			{
				desc:            "urn:cnb:registry id+version",
				locator:         "urn:cnb:registry:ns/name@1.2.3",
				expectedID:      "ns/name",
				expectedVersion: "1.2.3",
			},
		} {
			params := params
			when(params.desc+" "+params.locator, func() {
				it("should parse as id="+params.expectedID+" and version="+params.expectedVersion, func() {
					id, version := buildpack.ParseIDLocator(params.locator)
					assert.Equal(id, params.expectedID)
					assert.Equal(version, params.expectedVersion)
				})
			})
		}
	})

	when("#ParsePackageLocator", func() {
		type testParams struct {
			desc              string
			locator           string
			expectedImageName string
		}

		for _, params := range []testParams{
			{
				desc:              "docker scheme (missing host)",
				locator:           "docker:///ns/name:latest",
				expectedImageName: "ns/name:latest",
			},
			{
				desc:              "docker scheme (missing host shorthand)",
				locator:           "docker:/ns/name:latest",
				expectedImageName: "ns/name:latest",
			},
			{
				desc:              "docker scheme",
				locator:           "docker://docker.io/ns/name:latest",
				expectedImageName: "docker.io/ns/name:latest",
			},
			{
				desc:              "schemaless w/ host",
				locator:           "docker.io/ns/name:latest",
				expectedImageName: "docker.io/ns/name:latest",
			},
			{
				desc:              "schemaless w/o host",
				locator:           "ns/name:latest",
				expectedImageName: "ns/name:latest",
			},
		} {
			params := params
			when(params.desc+" "+params.locator, func() {
				it("should parse as "+params.expectedImageName, func() {
					imageName := buildpack.ParsePackageLocator(params.locator)
					assert.Equal(imageName, params.expectedImageName)
				})
			})
		}
	})

	when("#ParseRegistryID", func() {
		type testParams struct {
			desc,
			locator,
			expectedNS,
			expectedName,
			expectedVersion,
			expectedErr string
		}

		for _, params := range []testParams{
			{
				desc:            "naked id+version",
				locator:         "ns/name@0.1.2",
				expectedNS:      "ns",
				expectedName:    "name",
				expectedVersion: "0.1.2",
			},
			{
				desc:            "naked id",
				locator:         "ns/name",
				expectedNS:      "ns",
				expectedName:    "name",
				expectedVersion: "",
			},
			{
				desc:            "urn:cnb:registry ref",
				locator:         "urn:cnb:registry:ns/name@1.2.3",
				expectedNS:      "ns",
				expectedName:    "name",
				expectedVersion: "1.2.3",
			},
			{
				desc:        "invalid id",
				locator:     "invalid/id/name@1.2.3",
				expectedErr: "invalid registry ID: invalid/id/name@1.2.3",
			},
		} {
			params := params
			when(params.desc, func() {
				if params.expectedErr != "" {
					it("errors", func() {
						_, _, _, err := buildpack.ParseRegistryID(params.locator)
						assert.ErrorWithMessage(err, params.expectedErr)
					})
				} else {
					it("parses", func() {
						ns, name, version, err := buildpack.ParseRegistryID(params.locator)
						assert.Nil(err)
						assert.Equal(ns, params.expectedNS)
						assert.Equal(name, params.expectedName)
						assert.Equal(version, params.expectedVersion)
					})
				}
			})
		}
	})
}
