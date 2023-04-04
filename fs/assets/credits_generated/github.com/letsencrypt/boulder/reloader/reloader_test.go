package reloader

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"time"

	blog "github.com/letsencrypt/boulder/log"
)

func noop([]byte) error {
	return nil
}

func TestNoStat(t *testing.T) {
	filename := os.TempDir() + "/doesntexist.123456789"
	_, err := New(filename, noop, blog.NewMock())
	if err == nil {
		t.Fatalf("Expected New to return error when the file doesn't exist.")
	}
}

func TestNoRead(t *testing.T) {
	f, _ := os.CreateTemp("", "test-no-read.txt")
	defer os.Remove(f.Name())
	oldReadFile := readFile
	readFile = func(string) ([]byte, error) {
		return nil, fmt.Errorf("read failed")
	}
	_, err := New(f.Name(), noop, blog.NewMock())
	if err == nil {
		readFile = oldReadFile
		t.Fatalf("Expected New to return error when permission denied.")
	}
	readFile = oldReadFile
}

func TestFirstError(t *testing.T) {
	f, _ := os.CreateTemp("", "test-first-error.txt")
	defer os.Remove(f.Name())
	_, err := New(f.Name(), func([]byte) error {
		return fmt.Errorf("i die")
	}, blog.NewMock())
	if err == nil {
		t.Fatalf("Expected New to return error when the callback returned error the first time.")
	}
}

func TestFirstSuccess(t *testing.T) {
	f, _ := os.CreateTemp("", "test-first-success.txt")
	defer os.Remove(f.Name())
	r, err := New(f.Name(), func([]byte) error {
		return nil
	}, blog.NewMock())
	if err != nil {
		t.Errorf("Expected New to succeed, got %s", err)
	}
	r.Stop()
}

// Override makeTicker for testing.
// Returns a channel on which to send artificial ticks, and a function to
// restore the default makeTicker.
func makeFakeMakeTicker() (chan<- time.Time, func()) {
	origMakeTicker := makeTicker
	fakeTickChan := make(chan time.Time)
	makeTicker = func() (func(), <-chan time.Time) {
		return func() {}, fakeTickChan
	}
	return fakeTickChan, func() {
		makeTicker = origMakeTicker
	}
}

func TestReload(t *testing.T) {
	// Mock out makeTicker
	fakeTick, restoreMakeTicker := makeFakeMakeTicker()
	defer restoreMakeTicker()

	f, _ := os.CreateTemp("", "test-reload.txt")
	filename := f.Name()
	defer os.Remove(filename)

	_, _ = f.Write([]byte("first body"))
	_ = f.Close()

	var bodies []string
	reloads := make(chan []byte, 1)
	r, err := New(filename, func(b []byte) error {
		bodies = append(bodies, string(b))
		reloads <- b
		return nil
	}, blog.NewMock())
	if err != nil {
		t.Fatalf("Expected New to succeed, got %s", err)
	}
	defer r.Stop()
	expected := []string{"first body"}
	if !reflect.DeepEqual(bodies, expected) {
		t.Errorf("Expected bodies = %#v, got %#v", expected, bodies)
	}
	fakeTick <- time.Now()
	<-reloads
	if !reflect.DeepEqual(bodies, expected) {
		t.Errorf("Expected bodies = %#v, got %#v", expected, bodies)
	}

	// Write to the file, expect a reload. Sleep a few milliseconds first so the
	// timestamps actually differ.
	time.Sleep(1 * time.Second)
	err = os.WriteFile(filename, []byte("second body"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	fakeTick <- time.Now()
	<-reloads
	expected = []string{"first body", "second body"}
	if !reflect.DeepEqual(bodies, expected) {
		t.Errorf("Expected bodies = %#v, got %#v", expected, bodies)
	}

	// Send twice on this blocking channel to make sure we go through at least on
	// iteration of the reloader's loop.
	fakeTick <- time.Now()
	fakeTick <- time.Now()
	if !reflect.DeepEqual(bodies, expected) {
		t.Errorf("Expected bodies = %#v, got %#v", expected, bodies)
	}
}

// existingFile implements fs.FileInfo / os.FileInfo and returns information
// as if it were a basic file that existed. This is used to mock out os.Stat.
type existingFile struct{}

func (e existingFile) Name() string       { return "example" }
func (e existingFile) Size() int64        { return 10 }
func (e existingFile) Mode() fs.FileMode  { return 0 }
func (e existingFile) ModTime() time.Time { return time.Now() }
func (e existingFile) IsDir() bool        { return false }
func (e existingFile) Sys() any           { return nil }

func TestReloadFailure(t *testing.T) {
	// Mock out makeTicker
	fakeTick, restoreMakeTicker := makeFakeMakeTicker()

	f, _ := os.CreateTemp("", "test-reload-failure.txt")
	filename := f.Name()
	defer func() {
		restoreMakeTicker()
		_ = os.Remove(filename)
	}()

	_, _ = f.Write([]byte("first body"))
	_ = f.Close()

	type res struct {
		b   []byte
		err error
	}

	reloads := make(chan res, 1)
	log := blog.NewMock()
	_, err := New(filename, func(b []byte) error {
		reloads <- res{b, nil}
		return nil
	}, log)
	if err != nil {
		t.Fatalf("Expected New to succeed.")
	}
	<-reloads
	os.Remove(filename)
	fakeTick <- time.Now()
	time.Sleep(50 * time.Millisecond)
	err = log.ExpectMatch("statting .* no such file or directory")
	if err != nil {
		t.Error(err)
	}

	log.Clear()

	// Mock a file with no permissions
	oldReadFile := readFile
	readFile = func(string) ([]byte, error) {
		return nil, fmt.Errorf("permission denied")
	}
	oldStatFile := statFile
	statFile = func(string) (fs.FileInfo, error) {
		return existingFile{}, nil
	}

	fakeTick <- time.Now()
	time.Sleep(50 * time.Millisecond)
	err = log.ExpectMatch("reading .* permission denied")
	if err != nil {
		t.Error(err)
	}
	readFile = oldReadFile
	statFile = oldStatFile

	err = os.WriteFile(filename, []byte("third body"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	fakeTick <- time.Now()
	select {
	case r := <-reloads:
		if r.err != nil {
			t.Errorf("Unexpected error: %s", r.err)
		}
		if string(r.b) != "third body" {
			t.Errorf("Expected 'third body' reading file after restoring it.")
		}
		return
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for successful reload")
	}
}
