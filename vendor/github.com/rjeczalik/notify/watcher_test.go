// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build darwin linux freebsd dragonfly netbsd openbsd windows solaris

package notify

import (
	"os"
	"testing"
	"time"
)

// NOTE Set NOTIFY_DEBUG env var or debug build tag for extra debugging info.

func TestWatcher(t *testing.T) {
	w := NewWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	cases := [...]WCase{
		create(w, "src/github.com/ppknap/link/include/coost/.link.hpp.swp"),
		create(w, "src/github.com/rjeczalik/fs/fs_test.go"),
		create(w, "src/github.com/rjeczalik/fs/binfs/"),
		create(w, "src/github.com/rjeczalik/fs/binfs.go"),
		create(w, "src/github.com/rjeczalik/fs/binfs_test.go"),
		remove(w, "src/github.com/rjeczalik/fs/binfs/"),
		create(w, "src/github.com/rjeczalik/fs/binfs/"),
		create(w, "src/github.com/rjeczalik/fs/virfs"),
		remove(w, "src/github.com/rjeczalik/fs/virfs"),
		create(w, "file"),
		create(w, "dir/"),
	}

	w.ExpectAny(cases[:])
}

// Simulates the scenario, where outside of the programs control the base dir
// is removed. This is detected and the watch removed. Then the directory is
// restored and a new watch set up.
func TestStopPathNotExists(t *testing.T) {
	w := NewWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	if err := os.RemoveAll(w.root); err != nil {
		panic(err)
	}
	Sync()

	// Don't check the returned error, as the public function (notify.Stop)
	// does not return a potential error. As long as everything later on
	// works as inteded, that's fine
	time.Sleep(time.Duration(100) * time.Millisecond)
	w.Watcher.Unwatch(w.root)
	time.Sleep(time.Duration(100) * time.Millisecond)

	if err := os.Mkdir(w.root, 0777); err != nil {
		panic(err)
	}
	Sync()
	w.Watch("", All)

	drainall(w.C)
	cases := [...]WCase{
		create(w, "file"),
		create(w, "dir/"),
	}
	w.ExpectAny(cases[:])
}

func TestWatcherUnwatch(t *testing.T) {
	w := NewWatcherTest(t, "testdata/vfs.txt")
	defer w.Close()

	remove(w, "src/github.com/ppknap/link/test/test_circular_calls.cpp").Action()
	w.Unwatch("")

	w.Watch("", All)

	drainall(w.C)
	cases := [...]WCase{
		create(w, "file"),
		create(w, "dir/"),
	}
	w.ExpectAny(cases[:])
}
