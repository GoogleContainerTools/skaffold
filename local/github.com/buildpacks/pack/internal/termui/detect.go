package termui

import (
	"regexp"
	"time"

	"github.com/rivo/tview"

	"github.com/buildpacks/pack/pkg/dist"
)

type Detect struct {
	app  app
	bldr buildr

	textView       *tview.TextView
	buildpackRegex *regexp.Regexp
	buildpackChan  chan dist.ModuleInfo
	doneChan       chan bool
}

func NewDetect(app app, buildpackChan chan dist.ModuleInfo, bldr buildr) *Detect {
	d := &Detect{
		app:            app,
		textView:       detectStatusTV(),
		buildpackRegex: regexp.MustCompile(`^(\S+)\s+([\d\.]+)$`),
		buildpackChan:  buildpackChan,
		doneChan:       make(chan bool, 1),
		bldr:           bldr,
	}

	go d.start()
	grid := centered(d.textView)
	d.app.SetRoot(grid, true)
	return d
}

func (d *Detect) Handle(txt string) {
	m := d.buildpackRegex.FindStringSubmatch(txt)
	if len(m) == 3 {
		d.buildpackChan <- d.find(m[1], m[2])
	}
}

func (d *Detect) Stop() {
	d.doneChan <- true
}

func (d *Detect) SetNodes(map[string]*tview.TreeNode) {
	// no-op
	// This method is a side effect of the ill-fitting 'page interface'
	// Trying to create a cleaner interface between the main termui controller
	// and child pages like this one is currently a work-in-progress
}

func (d *Detect) start() {
	var (
		i        = 0
		ticker   = time.NewTicker(250 * time.Millisecond)
		doneText = "⌛️ Detected!"
		texts    = []string{
			"⏳️ Detecting",
			"⏳️ Detecting.",
			"⏳️ Detecting..",
			"⏳️ Detecting...",
		}
	)

	for {
		select {
		case <-ticker.C:
			d.app.QueueUpdateDraw(func() {
				d.textView.SetText(texts[i])
			})

			i++
			if i == len(texts) {
				i = 0
			}
		case <-d.doneChan:
			ticker.Stop()

			d.app.QueueUpdateDraw(func() {
				d.textView.SetText(doneText)
			})
			return
		}
	}
}

func (d *Detect) find(buildpackID, buildpackVersion string) dist.ModuleInfo {
	for _, buildpack := range d.bldr.Buildpacks() {
		if buildpack.ID == buildpackID && buildpack.Version == buildpackVersion {
			return buildpack
		}
	}

	return dist.ModuleInfo{
		ID:      buildpackID,
		Version: buildpackVersion,
	}
}

func detectStatusTV() *tview.TextView {
	tv := tview.NewTextView()
	tv.SetBackgroundColor(backgroundColor)
	return tv
}

func centered(p tview.Primitive) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, 20, 0).
		SetRows(0, 1, 0).
		AddItem(tview.NewBox().SetBackgroundColor(backgroundColor), 0, 0, 1, 1, 0, 0, true).
		AddItem(tview.NewBox().SetBackgroundColor(backgroundColor), 0, 1, 1, 1, 0, 0, true).
		AddItem(tview.NewBox().SetBackgroundColor(backgroundColor), 0, 2, 1, 1, 0, 0, true).
		AddItem(tview.NewBox().SetBackgroundColor(backgroundColor), 1, 0, 1, 1, 0, 0, true).
		AddItem(p, 1, 1, 1, 1, 0, 0, true).
		AddItem(tview.NewBox().SetBackgroundColor(backgroundColor), 1, 2, 1, 1, 0, 0, true).
		AddItem(tview.NewBox().SetBackgroundColor(backgroundColor), 2, 0, 1, 1, 0, 0, true).
		AddItem(tview.NewBox().SetBackgroundColor(backgroundColor), 2, 1, 1, 1, 0, 0, true).
		AddItem(tview.NewBox().SetBackgroundColor(backgroundColor), 2, 2, 1, 1, 0, 0, true)
}
