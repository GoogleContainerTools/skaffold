// Package reloader provides a method to load a file whenever it changes.
package reloader

import (
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	blog "github.com/letsencrypt/boulder/log"
)

// Wrap time.Tick so we can override it in tests.
var makeTicker = func() (func(), <-chan time.Time) {
	t := time.NewTicker(1 * time.Second)
	return t.Stop, t.C
}

// Reloader represents an ongoing reloader task.
type Reloader struct {
	stopChan chan<- struct{}
}

// Stop stops an active reloader, release its resources.
func (r *Reloader) Stop() {
	r.stopChan <- struct{}{}
}

// A pointer we can override for testing.
var readFile = os.ReadFile
var statFile = os.Stat

// New loads the filename provided, and calls the callback.  It then spawns a
// goroutine to check for updates to that file, calling the callback again with
// any new contents. The first load, and the first call to callback, are run
// synchronously, so it is easy for the caller to check for errors and fail
// fast. New will return an error if it occurs on the first load. Otherwise all
// errors are sent to the callback.
func New(filename string, dataCallback func([]byte) error, logger blog.Logger) (*Reloader, error) {
	fileInfo, err := statFile(filename)
	if err != nil {
		return nil, fmt.Errorf("statting %s: %w", filename, err)
	}
	b, err := readFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}
	stopChan := make(chan struct{})
	tickerStop, tickChan := makeTicker()
	loop := func() {
		for {
			select {
			case <-stopChan:
				tickerStop()
				return
			case <-tickChan:
				currentFileInfo, err := statFile(filename)
				if err != nil {
					logger.Errf("statting %s: %s", filename, err)
					continue
				}
				if !currentFileInfo.ModTime().After(fileInfo.ModTime()) {
					continue
				}
				b, err := readFile(filename)
				if err != nil {
					logger.Errf("reading %s: %s", filename, err)
					continue
				}
				fileInfo = currentFileInfo
				err = dataCallback(b)
				if err != nil {
					logger.Errf("processing %s: %s", filename, err)
					continue
				}

				hash := sha256.Sum256(b)
				logger.Infof("reloaded %s. sha256: %x, modified: %s",
					filename, hash[:], currentFileInfo.ModTime())
			}
		}
	}
	err = dataCallback(b)
	if err != nil {
		tickerStop()
		return nil, err
	}
	go loop()
	return &Reloader{stopChan}, nil
}
