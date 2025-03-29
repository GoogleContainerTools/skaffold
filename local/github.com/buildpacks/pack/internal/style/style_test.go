package style_test

import (
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/style"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestStyle(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testStyle", testStyle, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testStyle(t *testing.T, when spec.G, it spec.S) {
	when("#Symbol", func() {
		it("It should return the expected value", func() {
			h.AssertEq(t, style.Symbol("Symbol"), "'Symbol'")
		})

		it("It should return an empty string", func() {
			h.AssertEq(t, style.Symbol(""), "''")
		})

		it("It should return the expected value while color enabled", func() {
			color.Disable(false)
			defer color.Disable(true)
			h.AssertEq(t, style.Symbol("Symbol"), "\x1b[94mSymbol\x1b[0m")
		})

		it("It should return an empty string while color enabled", func() {
			color.Disable(false)
			defer color.Disable(true)
			h.AssertEq(t, style.Symbol(""), "\x1b[94m\x1b[0m")
		})
	})

	when("#SymbolF", func() {
		it("It should return the expected value", func() {
			h.AssertEq(t, style.SymbolF("values %s %d", "hello", 1), "'values hello 1'")
		})

		it("It should return an empty string", func() {
			h.AssertEq(t, style.SymbolF(""), "''")
		})

		it("It should return the expected value while color enabled", func() {
			color.Disable(false)
			defer color.Disable(true)
			h.AssertEq(t, style.SymbolF("values %s %d", "hello", 1), "\x1b[94mvalues hello 1\x1b[0m")
		})

		it("It should return an empty string while color enabled", func() {
			color.Disable(false)
			defer color.Disable(true)
			h.AssertEq(t, style.SymbolF(""), "\x1b[94m\x1b[0m")
		})
	})

	when("#Map", func() {
		it("It should return a string with all key value pairs", func() {
			h.AssertEq(t, style.Map(map[string]string{"FOO": "foo", "BAR": "bar"}, "", " "), "'BAR=bar FOO=foo'")
			h.AssertEq(t, style.Map(map[string]string{"BAR": "bar", "FOO": "foo"}, "  ", "\n"), "'BAR=bar\n  FOO=foo'")
		})

		it("It should return a string with all key value pairs while color enabled", func() {
			color.Disable(false)
			defer color.Disable(true)
			h.AssertEq(t, style.Map(map[string]string{"FOO": "foo", "BAR": "bar"}, "", " "), "\x1b[94mBAR=bar FOO=foo\x1b[0m")
			h.AssertEq(t, style.Map(map[string]string{"BAR": "bar", "FOO": "foo"}, "  ", "\n"), "\x1b[94mBAR=bar\n  FOO=foo\x1b[0m")
		})

		it("It should return an empty string", func() {
			h.AssertEq(t, style.Map(map[string]string{}, "", " "), "''")
		})

		it("It should return an empty string while color enabled", func() {
			color.Disable(false)
			defer color.Disable(true)
			h.AssertEq(t, style.Map(map[string]string{}, "", " "), "\x1b[94m\x1b[0m")
		})
	})
}
