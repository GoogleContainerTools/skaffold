package writer_test

import (
	"fmt"
	"testing"

	"github.com/buildpacks/pack/internal/inspectimage/writer"

	h "github.com/buildpacks/pack/testhelpers"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestFactory(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Builder Writer Factory", testFactory, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testFactory(t *testing.T, when spec.G, it spec.S) {
	var assert = h.NewAssertionManager(t)

	when("Writer", func() {
		when("Not BOM", func() {
			when("output format is human-readable", func() {
				it("returns a HumanReadable writer", func() {
					factory := writer.NewFactory()

					returnedWriter, err := factory.Writer("human-readable", false)
					assert.Nil(err)
					_, ok := returnedWriter.(*writer.HumanReadable)
					assert.TrueWithMessage(
						ok,
						fmt.Sprintf("expected %T to be assignable to type `*writer.HumanReadable`", returnedWriter),
					)
				})
			})

			when("output format is json", func() {
				it("return a JSON writer", func() {
					factory := writer.NewFactory()

					returnedWriter, err := factory.Writer("json", false)
					assert.Nil(err)

					_, ok := returnedWriter.(*writer.JSON)
					assert.TrueWithMessage(
						ok,
						fmt.Sprintf("expected %T to be assignable to type `*writer.JSON`", returnedWriter),
					)
				})
			})

			when("output format is yaml", func() {
				it("return a YAML writer", func() {
					factory := writer.NewFactory()

					returnedWriter, err := factory.Writer("yaml", false)
					assert.Nil(err)

					_, ok := returnedWriter.(*writer.YAML)
					assert.TrueWithMessage(
						ok,
						fmt.Sprintf("expected %T to be assignable to type `*writer.YAML`", returnedWriter),
					)
				})
			})

			when("output format is toml", func() {
				it("return a TOML writer", func() {
					factory := writer.NewFactory()

					returnedWriter, err := factory.Writer("toml", false)
					assert.Nil(err)

					_, ok := returnedWriter.(*writer.TOML)
					assert.TrueWithMessage(
						ok,
						fmt.Sprintf("expected %T to be assignable to type `*writer.TOML`", returnedWriter),
					)
				})
			})

			when("output format is not supported", func() {
				it("returns an error", func() {
					factory := writer.NewFactory()

					_, err := factory.Writer("mind-beam", false)
					assert.ErrorWithMessage(err, "output format 'mind-beam' is not supported")
				})
			})
		})

		when("BOM", func() {
			when("output format is json", func() {
				it("return a JSONBOM writer", func() {
					factory := writer.NewFactory()

					returnedWriter, err := factory.Writer("json", true)
					assert.Nil(err)

					_, ok := returnedWriter.(*writer.JSONBOM)
					assert.TrueWithMessage(
						ok,
						fmt.Sprintf("expected %T to be assignable to type `*writer.JSON`", returnedWriter),
					)
				})
			})

			when("output format is yaml", func() {
				it("return a YAMLBOM writer", func() {
					factory := writer.NewFactory()

					returnedWriter, err := factory.Writer("yaml", true)
					assert.Nil(err)

					_, ok := returnedWriter.(*writer.YAMLBOM)
					assert.TrueWithMessage(
						ok,
						fmt.Sprintf("expected %T to be assignable to type `*writer.JSON`", returnedWriter),
					)
				})
			})

			when("error cases", func() {
				when("output format is toml", func() {
					it("return an error", func() {
						factory := writer.NewFactory()

						_, err := factory.Writer("toml", true)
						assert.ErrorWithMessage(err, "output format 'toml' is not supported")
					})
				})

				when("output format is not supported", func() {
					it("returns an error", func() {
						factory := writer.NewFactory()

						_, err := factory.Writer("mind-BOM", true)
						assert.ErrorWithMessage(err, "output format 'mind-BOM' is not supported")
					})
				})
			})
		})
	})
}
