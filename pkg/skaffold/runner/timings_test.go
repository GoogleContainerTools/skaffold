/*
Copyright 2019 The Skaffold Authors

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

package runner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type mockBuilder struct {
	build.Builder
	err bool
}

func (m *mockBuilder) Build(context.Context, io.Writer, tag.ImageTags, []*latestV2.Artifact) ([]graph.Artifact, error) {
	if m.err {
		return nil, errors.New("Unable to build")
	}
	return nil, nil
}

func (m *mockBuilder) Prune(context.Context, io.Writer) error {
	if m.err {
		return errors.New("Unable to prune")
	}
	return nil
}

type mockTester struct {
	test.Tester
	err bool
}

func (m *mockTester) Test(context.Context, io.Writer, []graph.Artifact) error {
	if m.err {
		return errors.New("Unable to test")
	}
	return nil
}

type mockRenderer struct {
	test.Tester
	err bool
}

func (m *mockRenderer) Render(context.Context, io.Writer, []graph.Artifact, bool, string) error {
	if m.err {
		return errors.New("Unable to render")
	}
	return nil
}

func (m *mockRenderer) ManifestDeps() ([]string, error) {
	if m.err {
		return nil, errors.New("Unable to get manifest dependencies")
	}
	return nil, nil
}

type mockDeployer struct {
	deploy.Deployer
	err bool
}

func (m *mockDeployer) Deploy(context.Context, io.Writer, []graph.Artifact) ([]string, error) {
	if m.err {
		return nil, errors.New("Unable to deploy")
	}
	return nil, nil
}

func (m *mockDeployer) Cleanup(context.Context, io.Writer) error {
	if m.err {
		return errors.New("Unable to cleanup")
	}
	return nil
}

func TestTimingsBuild(t *testing.T) {
	tests := []struct {
		description  string
		shouldOutput string
		shouldLog    string
		shouldErr    bool
	}{
		{
			description:  "build success",
			shouldOutput: "",
			shouldLog:    "Build completed in .+$",
			shouldErr:    false,
		},
		{
			description:  "build failure",
			shouldOutput: "",
			shouldErr:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			hook := logrustest.NewGlobal()

			b := &mockBuilder{err: test.shouldErr}
			builder, _, _, _ := WithTimings(b, nil, nil, nil, false)

			var out bytes.Buffer
			_, err := builder.Build(context.Background(), &out, nil, nil)

			t.CheckError(test.shouldErr, err)
			t.CheckMatches(test.shouldOutput, out.String())
			t.CheckMatches(test.shouldLog, lastInfoEntry(hook))
		})
	}
}

func TestTimingsPrune(t *testing.T) {
	tests := []struct {
		description  string
		shouldOutput string
		shouldLog    string
		shouldErr    bool
	}{
		{
			description:  "test success",
			shouldOutput: "(?m)^Pruning images...\n",
			shouldLog:    "Image prune completed in .+$",
			shouldErr:    false,
		},
		{
			description:  "test failure",
			shouldOutput: "^Pruning images...\n$",
			shouldErr:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			hook := logrustest.NewGlobal()

			b := &mockBuilder{err: test.shouldErr}
			builder, _, _, _ := WithTimings(b, nil, nil, nil, false)

			var out bytes.Buffer
			err := builder.Prune(context.Background(), &out)

			t.CheckError(test.shouldErr, err)
			t.CheckMatches(test.shouldOutput, out.String())
			t.CheckMatches(test.shouldLog, lastInfoEntry(hook))
		})
	}
}

func TestTimingsTest(t *testing.T) {
	tests := []struct {
		description  string
		shouldOutput string
		shouldLog    string
		shouldErr    bool
	}{
		{
			description:  "test success",
			shouldOutput: "",
			shouldLog:    "Test completed in .+$",
			shouldErr:    false,
		},
		{
			description:  "test failure",
			shouldOutput: "",
			shouldErr:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			hook := logrustest.NewGlobal()

			tt := &mockTester{err: test.shouldErr}
			_, tester, _, _ := WithTimings(nil, tt, nil, nil, false)

			var out bytes.Buffer
			err := tester.Test(context.Background(), &out, nil)

			t.CheckError(test.shouldErr, err)
			t.CheckMatches(test.shouldOutput, out.String())
			t.CheckMatches(test.shouldLog, lastInfoEntry(hook))
		})
	}
}

func TestTimingsRender(t *testing.T) {
	tests := []struct {
		description  string
		shouldOutput string
		shouldLog    string
		shouldErr    bool
	}{
		{
			description:  "render success",
			shouldOutput: "",
			shouldLog:    "Render completed in .+$",
			shouldErr:    false,
		},
		{
			description:  "render failure",
			shouldOutput: "",
			shouldErr:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			hook := logrustest.NewGlobal()

			r := &mockRenderer{err: test.shouldErr}
			_, _, render, _ := WithTimings(nil, nil, r, nil, false)

			var out bytes.Buffer
			err := render.Render(context.Background(), &out, nil, false, "")

			t.CheckError(test.shouldErr, err)
			t.CheckMatches(test.shouldOutput, out.String())
			t.CheckMatches(test.shouldLog, lastInfoEntry(hook))
		})
	}
}

func TestTimingsDeploy(t *testing.T) {
	tests := []struct {
		description  string
		shouldOutput string
		shouldLog    string
		shouldErr    bool
	}{
		{
			description:  "prune success",
			shouldOutput: "(?m)^Starting deploy...\n",
			shouldLog:    "Deploy completed in .+$",
			shouldErr:    false,
		},
		{
			description:  "prune failure",
			shouldOutput: "^Starting deploy...\n$",
			shouldErr:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			hook := logrustest.NewGlobal()

			d := &mockDeployer{err: test.shouldErr}
			_, _, _, deployer := WithTimings(nil, nil, nil, d, false)

			var out bytes.Buffer
			_, err := deployer.Deploy(context.Background(), &out, nil)

			t.CheckError(test.shouldErr, err)
			t.CheckMatches(test.shouldOutput, out.String())
			t.CheckMatches(test.shouldLog, lastInfoEntry(hook))
		})
	}
}

func TestTimingsCleanup(t *testing.T) {
	tests := []struct {
		description  string
		shouldOutput string
		shouldLog    string
		shouldErr    bool
	}{
		{
			description:  "cleanup success",
			shouldOutput: "(?m)^Cleaning up...\n",
			shouldLog:    "Cleanup completed in .+$",
			shouldErr:    false,
		},
		{
			description:  "cleanup failure",
			shouldOutput: "^Cleaning up...\n$",
			shouldErr:    true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			hook := logrustest.NewGlobal()

			d := &mockDeployer{err: test.shouldErr}
			_, _, _, deployer := WithTimings(nil, nil, nil, d, false)

			var out bytes.Buffer
			err := deployer.Cleanup(context.Background(), &out)

			t.CheckError(test.shouldErr, err)
			t.CheckMatches(test.shouldOutput, out.String())
			t.CheckMatches(test.shouldLog, lastInfoEntry(hook))
		})
	}
}

func lastInfoEntry(hook *logrustest.Hook) string {
	for _, entry := range hook.AllEntries() {
		if entry.Level == logrus.InfoLevel {
			return entry.Message
		}
	}
	return ""
}
