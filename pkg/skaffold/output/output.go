/*
Copyright 2021 The Skaffold Authors

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

package output

import (
	"io"
	"io/ioutil"
	"os"
)

type skaffoldWriter struct {
	MainWriter  io.Writer
	EventWriter io.Writer
}

func (s skaffoldWriter) Write(p []byte) (int, error) {
	n, err := s.MainWriter.Write(p)
	if err != nil {
		return n, err
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}

	s.EventWriter.Write(p)

	return len(p), nil
}

func GetWriter(out io.Writer, defaultColor int, forceColors bool) io.Writer {
	if _, isSW := out.(skaffoldWriter); isSW {
		return out
	}

	return skaffoldWriter{
		MainWriter: SetupColors(out, defaultColor, forceColors),
		// TODO(marlongamez): Replace this once event writer is implemented
		EventWriter: ioutil.Discard,
	}
}

func IsStdout(out io.Writer) bool {
	sw, isSW := out.(skaffoldWriter)
	if isSW {
		out = sw.MainWriter
	}
	cw, isCW := out.(colorableWriter)
	if isCW {
		out = cw.Writer
	}
	return out == os.Stdout
}

// GetUnderlyingWriter returns the underlying writer if out is a colorableWriter
func GetUnderlyingWriter(out io.Writer) io.Writer {
	sw, isSW := out.(skaffoldWriter)
	if isSW {
		out = sw.MainWriter
	}
	cw, isCW := out.(colorableWriter)
	if isCW {
		out = cw.Writer
	}
	return out
}
