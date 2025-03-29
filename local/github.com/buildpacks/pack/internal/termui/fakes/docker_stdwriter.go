package fakes

import (
	"io"
	"time"

	"github.com/docker/docker/pkg/stdcopy"
)

type DockerStdWriter struct {
	wOut io.Writer
	wErr io.Writer
}

func NewDockerStdWriter(w io.Writer) *DockerStdWriter {
	return &DockerStdWriter{
		wOut: stdcopy.NewStdWriter(w, stdcopy.Stdout),
		wErr: stdcopy.NewStdWriter(w, stdcopy.Stderr),
	}
}

func (w *DockerStdWriter) WriteStdoutln(contents string) {
	w.write(contents+"\n", stdcopy.Stdout)
}

func (w *DockerStdWriter) WriteStderrln(contents string) {
	w.write(contents+"\n", stdcopy.Stderr)
}

func (w *DockerStdWriter) write(contents string, t stdcopy.StdType) {
	switch t {
	case stdcopy.Stdout:
		w.wOut.Write([]byte(contents))
	case stdcopy.Stderr:
		w.wErr.Write([]byte(contents))
	}

	// guard against race conditions
	time.Sleep(time.Millisecond)
}
