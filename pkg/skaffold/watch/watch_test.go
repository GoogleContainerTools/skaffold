/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package watch

import (
	"context"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestWatch(t *testing.T) {
	var tests = []struct {
		description string
		setup       func(folder *testutil.TempDir)
		update      func(folder *testutil.TempDir)
	}{
		{
			description: "file change",
			setup: func(folder *testutil.TempDir) {
				folder.Write("file", "content")
			},
			update: func(folder *testutil.TempDir) {
				folder.Chtimes("file", time.Now().Add(2*time.Second))
			},
		},
		{
			description: "file delete",
			setup: func(folder *testutil.TempDir) {
				folder.Write("file", "content")
			},
			update: func(folder *testutil.TempDir) {
				folder.Remove("file")
			},
		},
		{
			description: "file create",
			setup: func(folder *testutil.TempDir) {
				folder.Write("file", "content")
			},
			update: func(folder *testutil.TempDir) {
				folder.Write("new", "content")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			folder, cleanup := testutil.NewTempDir(t)
			defer cleanup()

			test.setup(folder)
			folderChanged := newCallback()
			somethingChanged := newCallback()

			// Watch folder
			watcher := NewWatcher(&pollTrigger{
				Interval: 10 * time.Millisecond,
			})
			err := watcher.Register(folder.List, folderChanged.call)
			testutil.CheckError(t, false, err)

			// Run the watcher
			ctx, cancel := context.WithCancel(context.Background())
			var stopped sync.WaitGroup
			stopped.Add(1)
			go func() {
				err = watcher.Run(ctx, ioutil.Discard, somethingChanged.callNoErr)
				stopped.Done()
				testutil.CheckError(t, false, err)
			}()

			test.update(folder)

			// Wait for the callbacks
			folderChanged.wait()
			somethingChanged.wait()
			cancel()
			stopped.Wait() // Make sure the watcher is stopped before deleting the tmp folder
		})
	}
}

type callback struct {
	wg *sync.WaitGroup
}

func newCallback() *callback {
	var wg sync.WaitGroup
	wg.Add(1)

	return &callback{
		wg: &wg,
	}
}

func (c *callback) call(e Events) {
	c.wg.Done()
}

func (c *callback) callNoErr() error {
	c.wg.Done()
	return nil
}

func (c *callback) wait() {
	c.wg.Wait()
}
