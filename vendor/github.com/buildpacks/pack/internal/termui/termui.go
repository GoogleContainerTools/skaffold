package termui

import (
	"archive/tar"
	"bufio"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/container"
	"github.com/buildpacks/pack/pkg/dist"
)

var (
	backgroundColor = tcell.NewRGBColor(5, 30, 40)
)

type app interface {
	SetRoot(root tview.Primitive, fullscreen bool) *tview.Application
	Draw() *tview.Application
	QueueUpdateDraw(f func()) *tview.Application
	Run() error
}

type buildr interface {
	BaseImageName() string
	Buildpacks() []dist.BuildpackInfo
	LifecycleDescriptor() builder.LifecycleDescriptor
	Stack() builder.StackMetadata
}

type page interface {
	Handle(txt string)
	Stop()
	SetNodes(nodes map[string]*tview.TreeNode)
}

type Termui struct {
	app         app
	bldr        buildr
	currentPage page

	appName       string
	runImageName  string
	exitCode      int64
	textChan      chan string
	buildpackChan chan dist.BuildpackInfo
	nodes         map[string]*tview.TreeNode
}

func NewTermui(appName string, bldr *builder.Builder, runImageName string) *Termui {
	return &Termui{
		appName:       appName,
		bldr:          bldr,
		runImageName:  runImageName,
		app:           tview.NewApplication(),
		buildpackChan: make(chan dist.BuildpackInfo, 50),
		textChan:      make(chan string, 50),
		nodes:         map[string]*tview.TreeNode{},
	}
}

// Run starts the terminal UI process in the foreground
// and the passed in function in the background
func (s *Termui) Run(funk func()) error {
	go func() {
		funk()
		s.showBuildStatus()
	}()
	go s.handle()
	defer s.stop()

	s.currentPage = NewDetect(s.app, s.buildpackChan, s.bldr)
	return s.app.Run()
}

func (s *Termui) stop() {
	close(s.textChan)
}

func (s *Termui) handle() {
	var detectLogs []string

	for txt := range s.textChan {
		switch {
		// We need a line that signals when detect phase is completed.
		// Since the phase order is: analyze -> detect -> restore -> build -> ...
		// "===> RESTORING" would be the best option. But since restore is optional,
		// "===> BUILDING" serves as the next best option.
		case strings.Contains(txt, "===> BUILDING"):
			s.currentPage.Stop()

			s.currentPage = NewDashboard(s.app, s.appName, s.bldr, s.runImageName, collect(s.buildpackChan), detectLogs)
			s.currentPage.Handle(txt)
		default:
			detectLogs = append(detectLogs, txt)
			s.currentPage.Handle(txt)
		}
	}
}

func (s *Termui) Handler() container.Handler {
	return func(bodyChan <-chan dcontainer.ContainerWaitOKBody, errChan <-chan error, reader io.Reader) error {
		var (
			copyErr = make(chan error)
			r, w    = io.Pipe()
			scanner = bufio.NewScanner(r)
		)

		go func() {
			defer w.Close()

			_, err := stdcopy.StdCopy(w, ioutil.Discard, reader)
			if err != nil {
				copyErr <- err
			}
		}()

		for {
			select {
			//TODO: errors should show up on screen
			//      instead of halting loop
			//See: https://github.com/buildpacks/pack/issues/1262
			case err := <-copyErr:
				return err
			case err := <-errChan:
				return err
			case body := <-bodyChan:
				s.exitCode = body.StatusCode
				return nil
			default:
				if scanner.Scan() {
					s.textChan <- scanner.Text()
					continue
				}

				if err := scanner.Err(); err != nil {
					return err
				}
			}
		}
	}
}

func (s *Termui) ReadLayers(reader io.ReadCloser) error {
	defer reader.Close()

	tr := tar.NewReader(reader)

	for {
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case err == io.EOF:
			if s.currentPage != nil {
				s.currentPage.SetNodes(s.nodes)
			}
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue

		default:
			name := path.Clean(header.Name)
			dir, base := filepath.Split(name)
			dir = strings.TrimSuffix(dir, "/")

			if s.nodes[dir] == nil {
				s.nodes[dir] = tview.NewTreeNode(dir)
			}

			node := tview.NewTreeNode(base).SetReference(header)
			s.nodes[name] = node
			s.nodes[dir].AddChild(node)
		}
	}
}

func (s *Termui) showBuildStatus() {
	if s.exitCode == 0 {
		s.textChan <- "[green::b]\n\nBUILD SUCCEEDED"
		return
	}

	s.textChan <- "[red::b]\n\nBUILD FAILED"
}

func collect(buildpackChan chan dist.BuildpackInfo) []dist.BuildpackInfo {
	close(buildpackChan)

	var result []dist.BuildpackInfo
	for txt := range buildpackChan {
		result = append(result, txt)
	}

	return result
}
