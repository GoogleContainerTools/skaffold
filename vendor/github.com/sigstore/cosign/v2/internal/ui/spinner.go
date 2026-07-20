// Copyright 2025 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/moby/term"
)

// Spinner shows progress for long-running operations in the terminal
type Spinner struct {
	done chan struct{}
}

// NewSpinner starts a spinner in a goroutine and returns it.
func NewSpinner(ctx context.Context, message string) *Spinner {
	s := &Spinner{
		done: make(chan struct{}),
	}

	go func() {
		// Don't show spinner if not in a terminal
		fd := os.Stderr.Fd()
		if !term.IsTerminal(fd) {
			Infof(ctx, "%s", message)
			return
		}

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		spinnerChars := []rune{'|', '/', '-', '\\'}
		i := 0
		for {
			select {
			case <-ticker.C:
				i++
				fmt.Fprintf(os.Stderr, "\r%s %c ", message, spinnerChars[i%len(spinnerChars)])
			case <-s.done:
				fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(message)+3))
				return
			}
		}
	}()
	return s
}

func (s *Spinner) Stop() {
	close(s.done)
}
