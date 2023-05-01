// Copyright 2020 Google LLC All Rights Reserved.
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

// Package editor implements a simple interface for interactive file editing.
// It most likely does not work on windows.
package editor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Edit opens a temporary file in the default editor (per $EDITOR, falling back
// to "vi") with the contents of the given io.Reader and a filename ending in
// the given extension (to give a hint to the editor for syntax highlighting).
//
// The contents of the edited file are returned, and the temporary file removed.
func Edit(input io.Reader, extension string) ([]byte, error) {
	f, err := os.CreateTemp("", fmt.Sprintf("%s-edit.*.%s", filepath.Base(os.Args[0]), extension))
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())

	if _, err := io.Copy(f, input); err != nil {
		return nil, err
	}
	f.Close()

	editor := "vi"
	if env := os.Getenv("EDITOR"); env != "" {
		editor = env
	}

	path, err := exec.LookPath(editor)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(path, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return os.ReadFile(f.Name())
}
