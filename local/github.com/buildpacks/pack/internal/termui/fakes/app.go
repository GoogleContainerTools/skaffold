package fakes

import (
	"github.com/rivo/tview"
)

type App struct {
	SetRootCallCount int
	DrawCallCount    int

	doneChan chan bool
}

func NewApp() *App {
	return &App{
		doneChan: make(chan bool, 1),
	}
}

func (a *App) SetRoot(root tview.Primitive, fullscreen bool) *tview.Application {
	a.SetRootCallCount++
	return nil
}

func (a *App) Draw() *tview.Application {
	a.DrawCallCount++
	return nil
}

func (a *App) QueueUpdateDraw(f func()) *tview.Application {
	f()
	a.DrawCallCount++
	return nil
}

func (a *App) Run() error {
	<-a.doneChan
	return nil
}

func (a *App) StopRunning() {
	a.doneChan <- true
}

func (a *App) ResetDrawCount() {
	a.DrawCallCount = 0
}
