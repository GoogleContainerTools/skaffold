package termui

import (
	"archive/tar"
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/buildpacks/pack/pkg/dist"
)

type Dive struct {
	app               app
	menuTable         *tview.Table
	fileExplorerTable *tview.Table
	buildpackInfo     []dist.ModuleInfo
	buildpacksTreeMap map[string]*tview.TreeNode
	escHandler        func()
}

func NewDive(app app, buildpackInfo []dist.ModuleInfo, selectedBuildpack dist.ModuleInfo, nodes map[string]*tview.TreeNode, escHandler func()) *Dive {
	menu := initMenu(buildpackInfo, nodes)
	fileExplorerTable := initFileExplorer()

	screen := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(menu, 0, 1, true).
		AddItem(fileExplorerTable, 0, 2, false)

	d := &Dive{
		app:               app,
		menuTable:         menu,
		fileExplorerTable: fileExplorerTable,
		buildpackInfo:     buildpackInfo,
		buildpacksTreeMap: nodes,
		escHandler:        escHandler,
	}

	d.handle()

	for row := 0; row < d.menuTable.GetRowCount(); row++ {
		if strings.Contains(d.menuTable.GetCell(row, 0).Text, selectedBuildpack.FullName()) {
			d.menuTable.Select(row, 0)
		}
	}

	d.app.SetRoot(screen, true)
	return d
}

func (d *Dive) handle() {
	selectionFunc := func(nodeKey string) func(row, column int) {
		return func(row, column int) {
			node := d.fileExplorerTable.GetCell(row, 3).GetReference().(*tview.TreeNode)

			if !node.GetReference().(*tar.Header).FileInfo().IsDir() {
				return
			}

			if node.IsExpanded() {
				node.Collapse()
			} else {
				node.Expand()
			}

			d.loadFileExplorerData(nodeKey)
		}
	}

	d.menuTable.SetSelectionChangedFunc(func(row, column int) {
		// protect from panic
		if row < 0 {
			return
		}

		// if SBOM
		if row == d.menuTable.GetRowCount()-1 {
			nodeKey := "layers/sbom"

			d.loadFileExplorerData(nodeKey)

			d.fileExplorerTable.ScrollToBeginning()

			d.fileExplorerTable.SetSelectedFunc(selectionFunc(nodeKey))
			return
		}

		// if buildpack
		selectedBuildpack := d.buildpackInfo[row-4]
		nodeKey := "layers/" + strings.ReplaceAll(selectedBuildpack.ID, "/", "_")

		d.loadFileExplorerData(nodeKey)

		d.fileExplorerTable.ScrollToBeginning()

		d.fileExplorerTable.SetSelectedFunc(selectionFunc(nodeKey))
	})

	d.menuTable.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			d.escHandler()
		case tcell.KeyTab:
			screen := tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(d.menuTable, 0, 1, false).
				AddItem(d.fileExplorerTable, 0, 2, true)
			d.app.SetRoot(screen, true)
		}
	})

	d.fileExplorerTable.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			d.escHandler()
		case tcell.KeyTab:
			screen := tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(d.menuTable, 0, 1, true).
				AddItem(d.fileExplorerTable, 0, 2, false)
			d.app.SetRoot(screen, true)
		}
	})
}

func (d *Dive) loadFileExplorerData(nodeKey string) {
	// Configure tree
	root := tview.NewTreeNode("[::b]Filetree[::-]")
	for _, child := range d.buildpacksTreeMap[nodeKey].GetChildren() {
		root.AddChild(child)
	}

	d.fileExplorerTable.Clear()

	// Configure Table
	d.fileExplorerTable.SetCell(0, 0, tview.NewTableCell("[::b]Permission[::-]").SetSelectable(false))
	d.fileExplorerTable.SetCell(0, 1, tview.NewTableCell("[::b]UID:GID[::-]").SetSelectable(false).
		SetAlign(tview.AlignRight))
	d.fileExplorerTable.SetCell(0, 2, tview.NewTableCell("[::b]Size[::-]  ").SetSelectable(false).
		SetAlign(tview.AlignRight))
	d.fileExplorerTable.SetCell(0, 3, tview.NewTableCell("[::b]Filetree[::-]").SetSelectable(false))

	branchesMapping := map[*tview.TreeNode]Branches{
		root: {},
	}

	root.Walk(func(node, parent *tview.TreeNode) bool {
		if node == root {
			return true
		}

		childCount := len(parent.GetChildren())
		isLast := parent.GetChildren()[childCount-1] == node

		if isLast {
			branchesMapping[node] = branchesMapping[parent].Add(NoBranchSymbol)
		} else {
			branchesMapping[node] = branchesMapping[parent].Add(BranchSymbol)
		}

		return true
	})

	var tableRow = 0
	root.Walk(func(node, parent *tview.TreeNode) bool {
		if node == root {
			tableRow++
			return true
		}

		ref := node.GetReference().(*tar.Header)

		collapseIcon := ""
		if !node.IsExpanded() {
			collapseIcon = " ⬥ "
		}

		size := "-"
		if ref.Typeflag != tar.TypeDir {
			size = humanize.Bytes(uint64(ref.Size))
		}

		color := ""
		switch {
		case ref.FileInfo().IsDir():
			color = "[mediumturquoise::b]"
		case ref.Typeflag == tar.TypeSymlink, ref.Typeflag == tar.TypeLink:
			color = "[purple]"
		case ref.FileInfo().Mode().Perm()&0111 != 0:
			color = "[yellow]"
		}

		childCount := len(parent.GetChildren())
		isLast := parent.GetChildren()[childCount-1] == node
		branches := branchesMapping[node].String()

		currentBranch := MiddleBranchSymbol.String()
		if isLast {
			currentBranch = LastBranchSymbol.String()
		}

		withLink := ""
		if ref.Typeflag == tar.TypeSymlink || ref.Typeflag == tar.TypeLink {
			withLink = "[-:-:-] → " + ref.Linkname
		}

		d.fileExplorerTable.SetCell(tableRow, 0, tview.NewTableCell(color+ref.FileInfo().Mode().String()))
		d.fileExplorerTable.SetCell(tableRow, 1, tview.NewTableCell(color+fmt.Sprintf("%d:%d", ref.Uid, ref.Gid)).
			SetAlign(tview.AlignRight))
		d.fileExplorerTable.SetCell(tableRow, 2, tview.NewTableCell(color+size+"  ").
			SetAlign(tview.AlignRight))
		d.fileExplorerTable.SetCell(tableRow, 3, tview.NewTableCell(branches+currentBranch+color+ref.FileInfo().Name()+withLink+collapseIcon).
			SetReference(node))

		tableRow++
		return node.IsExpanded()
	})
}

func initMenu(buildpackInfos []dist.ModuleInfo, nodes map[string]*tview.TreeNode) *tview.Table {
	style := tcell.StyleDefault.
		Foreground(tcell.ColorMediumTurquoise).
		Background(tcell.ColorDarkSlateGray).
		Attributes(tcell.AttrBold)

	table := tview.NewTable()
	table.
		SetSelectable(true, false).
		SetSelectedStyle(style).
		SetBorder(true).
		SetBorderPadding(1, 1, 2, 1).
		SetTitle("| [::b]phases[::-] |").
		SetTitleAlign(tview.AlignLeft).
		SetBackgroundColor(backgroundColor)

	var i int
	for _, phase := range []string{"ANALYZE", "DETECT", "RESTORE", "BUILD"} {
		table.SetCell(i, 0,
			tview.NewTableCell(phase).
				SetTextColor(tcell.ColorDarkGray).
				SetSelectable(false))
		i++
	}

	for _, buildpackInfo := range buildpackInfos {
		table.SetCell(i, 0,
			tview.NewTableCell(" ↳ "+buildpackInfo.FullName()).
				SetTextColor(tcell.ColorMediumTurquoise).
				SetSelectable(true))
		i++
	}

	table.SetCell(i, 0,
		tview.NewTableCell("EXPORT").
			SetTextColor(tcell.ColorDarkGray).
			SetSelectable(false))

	// set spacing
	i++
	i++
	sbomTextColor := tcell.ColorMediumTurquoise
	sbomSelectable := true
	if _, ok := nodes["layers/sbom"]; !ok {
		sbomTextColor = tcell.ColorDarkGray
		sbomSelectable = false
	}

	table.SetCell(i, 0,
		tview.NewTableCell("SBOM").
			SetTextColor(sbomTextColor).
			SetSelectable(sbomSelectable))
	return table
}

func initFileExplorer() *tview.Table {
	style := tcell.StyleDefault.
		Foreground(tcell.ColorMediumTurquoise).
		Background(tcell.ColorDarkSlateGray).
		Attributes(tcell.AttrBold)

	tbl := tview.NewTable()
	tbl.SetFixed(1, 0).
		SetSelectedStyle(style).
		SetSelectable(true, false).
		SetBackgroundColor(backgroundColor).
		SetBorderPadding(1, 1, 4, 0)
	return tbl
}
