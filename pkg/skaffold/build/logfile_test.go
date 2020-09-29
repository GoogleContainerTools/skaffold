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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestWithLogFile(t *testing.T) {
	logBuildInProgress := "building img with tag img:123"
	logBuildFailed := "failed to build img with tag img:123"
	logFilename := " - writing logs to " + filepath.Join(os.TempDir(), "skaffold", "build", "img.log")

	tests := []struct {
		description    string
		builder        ArtifactBuilder
		muted          Muted
		shouldErr      bool
		expectedDigest string
		logsFound      []string
		logsNotFound   []string
	}{
		{
			description:    "all logs",
			builder:        fakeBuilder,
			muted:          muted(false),
			shouldErr:      false,
			expectedDigest: "digest",
			logsFound:      []string{logBuildInProgress},
			logsNotFound:   []string{logFilename},
		},
		{
			description:    "mute build logs",
			builder:        fakeBuilder,
			muted:          muted(true),
			shouldErr:      false,
			expectedDigest: "digest",
			logsFound:      []string{logFilename},
			logsNotFound:   []string{logBuildInProgress},
		},
		{
			description:    "failed build - all logs",
			builder:        fakeFailingBuilder,
			muted:          muted(false),
			shouldErr:      true,
			expectedDigest: "",
			logsFound:      []string{logBuildFailed},
			logsNotFound:   []string{logFilename},
		},
		{
			description:    "failed build - muted logs",
			builder:        fakeFailingBuilder,
			muted:          muted(true),
			shouldErr:      true,
			expectedDigest: "",
			logsFound:      []string{logFilename},
			logsNotFound:   []string{logBuildFailed},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var out bytes.Buffer

			builder := WithLogFile(test.builder, test.muted)
			digest, err := builder(context.Background(), &out, &latest.Artifact{ImageName: "img"}, "img:123")

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedDigest, digest)
			for _, found := range test.logsFound {
				t.CheckContains(found, out.String())
			}
			for _, notFound := range test.logsNotFound {
				t.CheckFalse(strings.Contains(out.String(), notFound))
			}
		})
	}
}

func fakeBuilder(_ context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	fmt.Fprintln(out, "building", a.ImageName, "with tag", tag)
	return "digest", nil
}

func fakeFailingBuilder(_ context.Context, out io.Writer, a *latest.Artifact, tag string) (string, error) {
	fmt.Fprintln(out, "failed to build", a.ImageName, "with tag", tag)
	return "", errors.New("bug")
}

type muted bool

func (m muted) MuteBuild() bool {
	return bool(m)
}
