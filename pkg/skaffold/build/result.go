/*
Copyright 2020 The Skaffold Authors

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

package build

import (
	"bufio"
	"fmt"
	"io"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

const bufferedLinesPerArtifact = 10000

// For testing
var (
	buffSize = bufferedLinesPerArtifact
)

// logWriter provides an interface to create an output writer for each artifact build and later aggregate the logs in build order.
// The order of output is not guaranteed between multiple builds running concurrently.
type logWriter interface {
	// GetWriter returns an output writer tracked by the logWriter
	GetWriter() (io.WriteCloser, error)
	// PrintInOrder prints the output from each allotted writer in build order.
	// It blocks until the instantiated capacity of io writers have been all allotted and closed.
	PrintInOrder(out io.Writer)
}

type logWriterImpl struct {
	messages   chan chan string
	size       int
	capacity   int
	countMutex sync.Mutex
}

func (l *logWriterImpl) GetWriter() (io.WriteCloser, error) {
	if err := l.checkCapacity(); err != nil {
		return nil, err
	}
	r, w := io.Pipe()
	ch := make(chan string, buffSize)
	l.messages <- ch
	// write the build output to a buffered channel.
	go l.writeToChannel(r, ch)
	return w, nil
}

func (l *logWriterImpl) PrintInOrder(out io.Writer) {
	defer close(l.messages)
	for i := 0; i < l.capacity; i++ {
		// read from each build's message channel and write to the given output.
		printResult(out, <-l.messages)
	}
}

func (l *logWriterImpl) checkCapacity() error {
	l.countMutex.Lock()
	defer l.countMutex.Unlock()
	if l.size == l.capacity {
		return fmt.Errorf("failed to create writer: capacity exceeded")
	}
	l.size++
	return nil
}
func printResult(out io.Writer, output chan string) {
	for line := range output {
		fmt.Fprintln(out, line)
	}
}

func (l *logWriterImpl) writeToChannel(r io.Reader, lines chan string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	close(lines)
}

func newLogWriter(capacity int) logWriter {
	return &logWriterImpl{capacity: capacity, messages: make(chan chan string, capacity)}
}

// resultStore stores the results of each artifact build.
type resultStore interface {
	Record(a *latest.Artifact, tag string, err error)
	GetTag(a *latest.Artifact) (string, error)
}

func newResultStore() resultStore {
	return &resultStoreImpl{m: new(sync.Map)}
}

type resultStoreImpl struct {
	m *sync.Map
}

func (r *resultStoreImpl) Record(a *latest.Artifact, tag string, err error) {
	if err != nil {
		r.m.Store(a.ImageName, err)
	} else {
		r.m.Store(a.ImageName, tag)
	}
}

func (r *resultStoreImpl) GetTag(a *latest.Artifact) (string, error) {
	v, ok := r.m.Load(a.ImageName)
	if !ok {
		return "", fmt.Errorf("could not find build result for image %s", a.ImageName)
	}
	switch t := v.(type) {
	case error:
		return "", fmt.Errorf("couldn't build %q: %w", a.ImageName, t)
	case string:
		return t, nil
	default:
		return "", fmt.Errorf("could not find build result for image %s", a.ImageName)
	}
}
