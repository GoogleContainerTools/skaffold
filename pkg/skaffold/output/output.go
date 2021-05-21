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

type SkaffoldWriter struct {
	MainWriter  io.Writer
	EventWriter io.Writer
}

func (s SkaffoldWriter) Write(p []byte) (int, error) {
	for _, w := range []io.Writer{s.MainWriter, s.EventWriter} {
		n, err := w.Write(p)
		if err != nil {
			return n, err
		}
		if n < len(p) {
			return n, io.ErrShortWrite
		}
	}

	return len(p), nil
}

func SetupOutput(out io.Writer, defaultColor int, forceColors bool) io.Writer {
	return SkaffoldWriter{
		MainWriter: SetupColors(out, defaultColor, forceColors),
		// TODO(marlongamez): Replace this once event writer is implemented
		EventWriter: ioutil.Discard,
	}
}

func IsStdout(out io.Writer) bool {
	sw, isSW := out.(SkaffoldWriter)
	if isSW {
		cw, isCW := sw.MainWriter.(colorableWriter)
		if isCW {
			return cw.Writer == os.Stdout
		}
		return sw.MainWriter == os.Stdout
	}
	return out == os.Stdout
}

// GetWriter returns the underlying writer if out is a colorableWriter
func GetWriter(out io.Writer) io.Writer {
	sw, isSW := out.(SkaffoldWriter)
	if isSW {
		cw, isCW := sw.MainWriter.(colorableWriter)
		if isCW {
			return cw.Writer
		}
		return sw.MainWriter
	}
	return out
}
