package comparehelpers_test

import (
	"encoding/json"
	"testing"

	"github.com/buildpacks/pack/testhelpers"
	"github.com/buildpacks/pack/testhelpers/comparehelpers"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestDeepContains(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Builder Writer", testDeepContains, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testDeepContains(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = testhelpers.NewAssertionManager(t)
	)
	when("DeepContains", func() {
		var (
			containerJSON string
			container     interface{}
		)
		when("Searching for array containment", func() {
			it.Before(func() {
				containerJSON = `[
	{
	  "Name": "Platypus",
	  "Order": "Monotremata",
      "Info":  [
		{
			"Population": 5000,
			"Habitat": ["splish-spash", "waters"]
		},
		{
			"Geography" : "Moon"
		},
		{
			"Discography": "My records are all platynum"
		}
	  ]
	},
	{
	  "Name": "Quoll",
	  "Order": "Dasyuromorphia",
	  "Info": []
	}
]`

				assert.Succeeds(json.Unmarshal([]byte(containerJSON), &container))
			})
			when("subarray is contained", func() {
				it("return true", func() {
					containedJSON := `[{ "Geography":"Moon" }, {"Discography": "My records are all platynum"}]`

					var contained interface{}
					assert.Succeeds(json.Unmarshal([]byte(containedJSON), &contained))

					out := comparehelpers.DeepContains(container, contained)
					assert.Equal(out, true)
				})
			})
			when("subarray is not contained", func() {
				it("returns false", func() {
					containedJSON := `[{ "Geography":"Moon" }, {"Discography": "Splish-splash Cash III"}]`

					var contained interface{}
					assert.Succeeds(json.Unmarshal([]byte(containedJSON), &contained))

					out := comparehelpers.DeepContains(container, contained)
					assert.Equal(out, false)
				})
			})
		})
		when("Searching for map containment", func() {
			var (
				containerJSON string
				container     interface{}
			)
			it.Before(func() {
				containerJSON = `[
	{
	  "Name": "Platypus",
	  "Order": "Monotremata",
      "Info":  [
		{
			"Population": 5000,
			"Size": "smol",
			"Habitat": ["shallow", "waters"]
		},
		{
			"Geography" : "Moon"
		},
		{
			"Discography": "My records are all platynum"
		}
	  ]
	},
	{
	  "Name": "Quoll",
	  "Order": "Dasyuromorphia",
	  "Info": []
	}
]`
				assert.Succeeds(json.Unmarshal([]byte(containerJSON), &container))
			})
			when("map is contained", func() {
				it("returns true", func() {
					containedJSON := `{"Population": 5000, "Size": "smol"}`
					var contained interface{}
					assert.Succeeds(json.Unmarshal([]byte(containedJSON), &contained))

					out := comparehelpers.DeepContains(container, contained)
					assert.Equal(out, true)
				})
			})
			when("map is not contained", func() {
				it("returns false", func() {
					containedJSON := `{"Order": "Nemotode"}`
					var contained interface{}
					assert.Succeeds(json.Unmarshal([]byte(containedJSON), &contained))

					out := comparehelpers.DeepContains(container, contained)
					assert.Equal(out, false)
				})
			})
		})
	})
	when("json is not contained", func() {
		it("return false", func() {
			containerJSON := `[
	{"Name": "Platypus", "Order": "Monotremata"},
	{"Name": "Quoll",    "Order": "Dasyuromorphia"}
]`
			var container interface{}
			assert.Succeeds(json.Unmarshal([]byte(containerJSON), &container))

			containedJSON := `[{"Name": "Notapus", "Order": "Monotremata"}]`

			var contained interface{}
			assert.Succeeds(json.Unmarshal([]byte(containedJSON), &contained))

			out := comparehelpers.DeepContains(container, contained)
			assert.Equal(out, false)
		})
	})
}
