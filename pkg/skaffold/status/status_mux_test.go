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

package status

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestMonitorMux(t *testing.T) {
	tests := []struct {
		description string
		monitor     func(context.Context, io.Writer) error
		shouldErr   bool
	}{
		{
			description: "passing",
			monitor: func(c context.Context, w io.Writer) error {
				return nil
			},
		},
		{
			description: "failing",
			monitor: func(c context.Context, w io.Writer) error {
				return errors.New("error")
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			var m MonitorMux
			for i := 0; i < 3; i++ {
				m = append(m, &MockMonitor{monitor: test.monitor})
			}
			err := m.Check(context.Background(), ioutil.Discard)
			if test.shouldErr {
				t.CheckError(true, err)
			} else {
				t.CheckNoError(err)
				for _, mi := range m {
					t.CheckTrue(mi.(*MockMonitor).run)
				}
				m.Reset()
				for _, mi := range m {
					t.CheckFalse(mi.(*MockMonitor).run)
				}
			}
		})
	}
}

type MockMonitor struct {
	run     bool
	monitor func(context.Context, io.Writer) error
}

func (m *MockMonitor) Check(ctx context.Context, out io.Writer) error {
	m.run = true
	return m.monitor(ctx, out)
}

func (m *MockMonitor) Reset() { m.run = false }
