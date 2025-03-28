package termui

import (
	"os"
	"testing"

	"github.com/rivo/tview"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/termui/fakes"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestDiveScreen(t *testing.T) {
	spec.Run(t, "DiveScreen", testDive, spec.Report(report.Terminal{}))
}

func testDive(t *testing.T, when spec.G, it spec.S) {
	var (
		fakeApp           app
		buildpacks        []dist.ModuleInfo
		selectedBuildpack dist.ModuleInfo
		nodes             map[string]*tview.TreeNode
	)

	it.Before(func() {
		fakeApp = fakes.NewApp()
		buildpacks = []dist.ModuleInfo{
			{ID: "some/buildpack-1", Version: "0.0.1"},
			{ID: "some/buildpack-2", Version: "0.0.2"}}
		selectedBuildpack = buildpacks[0]

		// fetch nodes
		termui := &Termui{
			nodes: map[string]*tview.TreeNode{}}
		f, err := os.Open("./testdata/fake-layers.tar")
		h.AssertNil(t, err)
		h.AssertNil(t, termui.ReadLayers(f))
		nodes = termui.nodes
	})

	it("loads buildpack and layer data", func() {
		screen := NewDive(fakeApp, buildpacks, selectedBuildpack, nodes, func() {})
		h.AssertContains(t, screen.menuTable.GetCell(4, 0).Text, "some/buildpack-1@0.0.1")
		h.AssertContains(t, screen.menuTable.GetCell(5, 0).Text, "some/buildpack-2@0.0.2")

		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 0).Text, "-rw-r--r--")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 1).Text, "501:20")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 2).Text, "14 B  ")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 3).Text, "└── some-file-1.txt")

		// select the other buildpack
		screen.menuTable.Select(5, 0)
		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 0).Text, "drwxr-xr-x")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 1).Text, "501:20")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 2).Text, "-  ")
		h.AssertContainsMatch(t, screen.fileExplorerTable.GetCell(1, 3).Text, "└── .*some-dir")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(2, 0).Text, "-rw-r--r--")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(2, 1).Text, "501:20")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(2, 2).Text, "14 B  ")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(2, 3).Text, "    └── some-file-2.txt")

		// select SBOM
		lastRow := screen.menuTable.GetRowCount() - 1
		screen.menuTable.Select(lastRow, 0)
		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 0).Text, "drwxr-xr-x")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 1).Text, "501:20")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(1, 2).Text, "-  ")
		h.AssertContainsMatch(t, screen.fileExplorerTable.GetCell(1, 3).Text, "└── .*launch")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(2, 0).Text, "-rw-r--r--")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(2, 1).Text, "501:20")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(2, 2).Text, "32 B  ")
		h.AssertContains(t, screen.fileExplorerTable.GetCell(2, 3).Text, "    └── sbom.cdx.json")
	})
}
