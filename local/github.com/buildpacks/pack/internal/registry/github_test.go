package registry_test

import (
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/registry"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestGithub(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Github", testGitHub, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testGitHub(t *testing.T, when spec.G, it spec.S) {
	when("#CreateBrowserCommand", func() {
		it("should error when browser URL is invalid", func() {
			_, err := registry.CreateBrowserCmd("", "linux")
			h.AssertError(t, err, "invalid URL")
		})

		it("should error when os is unsupported", func() {
			_, err := registry.CreateBrowserCmd("https://url.com", "valinor")
			h.AssertError(t, err, "unsupported platform 'valinor'")
		})

		it("should work for linux", func() {
			cmd, err := registry.CreateBrowserCmd("https://buildpacks.io", "linux")
			h.AssertNil(t, err)
			h.AssertEq(t, len(cmd.Args), 2)
			h.AssertEq(t, cmd.Args[0], "xdg-open")
			h.AssertEq(t, cmd.Args[1], "https://buildpacks.io")
		})

		it("should work for darwin", func() {
			cmd, err := registry.CreateBrowserCmd("https://buildpacks.io", "darwin")
			h.AssertNil(t, err)
			h.AssertEq(t, len(cmd.Args), 2)
			h.AssertEq(t, cmd.Args[0], "open")
			h.AssertEq(t, cmd.Args[1], "https://buildpacks.io")
		})

		it("should work for windows", func() {
			cmd, err := registry.CreateBrowserCmd("https://buildpacks.io", "windows")
			h.AssertNil(t, err)
			h.AssertEq(t, len(cmd.Args), 3)
			h.AssertEq(t, cmd.Args[0], "rundll32")
			h.AssertEq(t, cmd.Args[1], "url.dll,FileProtocolHandler")
			h.AssertEq(t, cmd.Args[2], "https://buildpacks.io")
		})
	})

	when("#GetIssueURL", func() {
		it("should return an issueURL", func() {
			url, err := registry.GetIssueURL("https://github.com/buildpacks")

			h.AssertNil(t, err)
			h.AssertEq(t, url.String(), "https://github.com/buildpacks/issues/new")
		})

		it("should fail when url is empty", func() {
			_, err := registry.GetIssueURL("")

			h.AssertError(t, err, "missing github URL")
		})
	})
}
