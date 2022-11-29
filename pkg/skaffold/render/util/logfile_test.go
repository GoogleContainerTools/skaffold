/*
Copyright 2022 The Skaffold Authors

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

package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestWithLogFile(t *testing.T) {
	logRenderSucceeded := " - fake/render_result created"
	logRenderFailed := " - failed to render"
	logFilename := "- writing log to " + filepath.Join(os.TempDir(), "skaffold", "render", "render.log")

	tests := []struct {
		description  string
		muted        Muted
		shouldErr    bool
		logsFound    []string
		logsNotFound []string
	}{
		{
			description:  "all logs",
			muted:        mutedRender(false),
			shouldErr:    false,
			logsFound:    []string{logRenderSucceeded},
			logsNotFound: []string{logFilename},
		},
		{
			description:  "mute render logs",
			muted:        mutedRender(true),
			shouldErr:    false,
			logsFound:    []string{logFilename},
			logsNotFound: []string{logRenderSucceeded},
		},
		{
			description:  "failed render - all logs",
			muted:        mutedRender(false),
			shouldErr:    true,
			logsFound:    []string{logRenderFailed},
			logsNotFound: []string{logFilename},
		},
		{
			description:  "failed render - mutedRender logs",
			muted:        mutedRender(true),
			shouldErr:    true,
			logsFound:    []string{logFilename},
			logsNotFound: []string{logRenderFailed},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var mockOut bytes.Buffer

			var renderer = mockRenderer{
				muted:     test.muted,
				shouldErr: test.shouldErr,
			}

			renderOut, postRenderFn, _ := WithLogFile("render.log", &mockOut, test.muted)
			err := renderer.Render(context.Background(), renderOut, nil, nil)
			postRenderFn()

			t.CheckError(test.shouldErr, err)
			for _, found := range test.logsFound {
				t.CheckContains(found, mockOut.String())
			}
			for _, notFound := range test.logsNotFound {
				t.CheckFalse(strings.Contains(mockOut.String(), notFound))
			}
		})
	}
}

// Used just to show how output gets routed to different writers with the log file
type mockRenderer struct {
	muted     Muted
	shouldErr bool
}

func (fd *mockRenderer) Render(ctx context.Context, out io.Writer, _ []graph.Artifact, _ manifest.ManifestList) error {
	if fd.shouldErr {
		fmt.Fprintln(out, " - failed to render")
		return errors.New("failed to render")
	}

	fmt.Fprintln(out, " - fake/render_result created")
	return nil
}

type mutedRender bool

func (m mutedRender) MuteRender() bool {
	return bool(m)
}
