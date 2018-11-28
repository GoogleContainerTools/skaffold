// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stream

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func TestStreamVsBuffer(t *testing.T) {
	var n, wantSize int64 = 10000, 49
	newBlob := func() io.ReadCloser { return ioutil.NopCloser(bytes.NewReader(bytes.Repeat([]byte{'a'}, int(n)))) }
	wantDigest := "sha256:3d7c465be28d9e1ed810c42aeb0e747b44441424f566722ba635dc93c947f30e"
	wantDiffID := "sha256:27dd1f61b867b6a0f6e9d8a41c43231de52107e53ae424de8f847b821db4b711"

	// Check that streaming some content results in the expected digest/diffID/size.
	l := NewLayer(newBlob())
	if c, err := l.Compressed(); err != nil {
		t.Errorf("Compressed: %v", err)
	} else {
		if _, err := io.Copy(ioutil.Discard, c); err != nil {
			t.Errorf("error reading Compressed: %v", err)
		}
		if err := c.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}
	if d, err := l.Digest(); err != nil {
		t.Errorf("Digest: %v", err)
	} else if d.String() != wantDigest {
		t.Errorf("stream Digest got %q, want %q", d.String(), wantDigest)
	}
	if d, err := l.DiffID(); err != nil {
		t.Errorf("DiffID: %v", err)
	} else if d.String() != wantDiffID {
		t.Errorf("stream DiffID got %q, want %q", d.String(), wantDiffID)
	}
	if s, err := l.Size(); err != nil {
		t.Errorf("Size: %v", err)
	} else if s != wantSize {
		t.Errorf("stream Size got %d, want %d", s, wantSize)
	}

	// Test that buffering the same contents and using
	// tarball.LayerFromOpener results in the same digest/diffID/size.
	tl, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) { return newBlob(), nil })
	if err != nil {
		t.Fatalf("LayerFromOpener: %v", err)
	}
	if d, err := tl.Digest(); err != nil {
		t.Errorf("Digest: %v", err)
	} else if d.String() != wantDigest {
		t.Errorf("tarball Digest got %q, want %q", d.String(), wantDigest)
	}
	if d, err := tl.DiffID(); err != nil {
		t.Errorf("DiffID: %v", err)
	} else if d.String() != wantDiffID {
		t.Errorf("tarball DiffID got %q, want %q", d.String(), wantDiffID)
	}
	if s, err := tl.Size(); err != nil {
		t.Errorf("Size: %v", err)
	} else if s != wantSize {
		t.Errorf("stream Size got %d, want %d", s, wantSize)
	}
}

func TestLargeStream(t *testing.T) {
	var n, wantSize int64 = 100000000, 100007653 // "Compressing" n random bytes results in this many bytes.
	sl := NewLayer(ioutil.NopCloser(io.LimitReader(rand.Reader, n)))
	rc, err := sl.Compressed()
	if err != nil {
		t.Fatalf("Uncompressed: %v", err)
	}
	if _, err := io.Copy(ioutil.Discard, rc); err != nil {
		t.Fatalf("Reading layer: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if dig, err := sl.Digest(); err != nil {
		t.Errorf("Digest: %v", err)
	} else if dig.String() == (v1.Hash{}).String() {
		t.Errorf("Digest got %q, want anything else", (v1.Hash{}).String())
	}
	if diffID, err := sl.DiffID(); err != nil {
		t.Errorf("DiffID: %v", err)
	} else if diffID.String() == (v1.Hash{}).String() {
		t.Errorf("DiffID got %q, want anything else", (v1.Hash{}).String())
	}
	if size, err := sl.Size(); err != nil {
		t.Errorf("Size: %v", err)
	} else if size != wantSize {
		t.Errorf("Size got %d, want %d", size, n)
	}
}

func TestStreamableLayerFromTarball(t *testing.T) {
	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)
	go func() {
		// "Stream" a bunch of files into the layer.
		pw.CloseWithError(func() error {
			for i := 0; i < 1000; i++ {
				name := fmt.Sprintf("file-%d.txt", i)
				body := fmt.Sprintf("i am file number %d", i)
				if err := tw.WriteHeader(&tar.Header{
					Name:     name,
					Mode:     0600,
					Size:     int64(len(body)),
					Typeflag: tar.TypeReg,
				}); err != nil {
					return err
				}
				if _, err := tw.Write([]byte(body)); err != nil {
					return err
				}
			}
			return nil
		}())
	}()

	l := NewLayer(pr)
	rc, err := l.Compressed()
	if err != nil {
		t.Fatalf("Compressed: %v", err)
	}
	if _, err := io.Copy(ioutil.Discard, rc); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	wantDigest := "sha256:f53d6a164ab476294212843f267740bd12f79e00abd8050c24ce8a9bceaa36b0"
	if got, err := l.Digest(); err != nil {
		t.Errorf("Digest: %v", err)
	} else if got.String() != wantDigest {
		t.Errorf("Digest: got %q, want %q", got.String(), wantDigest)
	}
}

// TestNotComputed tests that Digest/DiffID/Size return ErrNotComputed before
// the stream has been consumed.
func TestNotComputed(t *testing.T) {
	l := NewLayer(ioutil.NopCloser(bytes.NewBufferString("hi")))

	// All methods should return ErrNotComputed until the stream has been
	// consumed and closed.
	if _, err := l.Size(); err != ErrNotComputed {
		t.Errorf("Size: got %v, want %v", err, ErrNotComputed)
	}
	if _, err := l.Digest(); err == nil {
		t.Errorf("Digest: got %v, want %v", err, ErrNotComputed)
	}
	if _, err := l.DiffID(); err == nil {
		t.Errorf("DiffID: got %v, want %v", err, ErrNotComputed)
	}
}

// TestConsumed tests that Compressed returns ErrConsumed when the stream has
// already been consumed.
func TestConsumed(t *testing.T) {
	l := NewLayer(ioutil.NopCloser(strings.NewReader("hello")))
	rc, err := l.Compressed()
	if err != nil {
		t.Errorf("Compressed: %v", err)
	}
	if _, err := io.Copy(ioutil.Discard, rc); err != nil {
		t.Errorf("Error reading contents: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	if _, err := l.Compressed(); err != ErrConsumed {
		t.Errorf("Compressed() after consuming; got %v, want %v", err, ErrConsumed)
	}
}
