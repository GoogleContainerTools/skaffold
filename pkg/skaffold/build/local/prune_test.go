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

package local

import (
	"context"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDiskUsage(t *testing.T) {
	tests := []struct {
		ctxFunc             func() context.Context
		description         string
		fails               int
		expectedUtilization uint64
		shouldErr           bool
	}{
		{
			description:         "happy path",
			fails:               0,
			shouldErr:           false,
			expectedUtilization: testutil.TestUtilization,
		},
		{
			description:         "first attempts failed",
			fails:               usageRetries - 1,
			shouldErr:           false,
			expectedUtilization: testutil.TestUtilization,
		},
		{
			description:         "all attempts failed",
			fails:               usageRetries,
			shouldErr:           true,
			expectedUtilization: 0,
		},
		{
			description:         "context cancelled",
			fails:               1,
			shouldErr:           true,
			expectedUtilization: 0,
			ctxFunc: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{
				DUFails: test.fails,
			}), true)

			ctx := context.Background()
			if test.ctxFunc != nil {
				ctx = test.ctxFunc()
			}
			res, err := pruner.diskUsage(ctx)

			t.CheckError(test.shouldErr, err)
			if res != test.expectedUtilization {
				t.Errorf("invalid disk usage. got %d expected %d", res, test.expectedUtilization)
			}
		})
	}
}

func TestRunPruneOk(t *testing.T) {
	pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{}), true)
	err := pruner.runPrune(context.Background(), []string{"test"})
	if err != nil {
		t.Fatalf("Got an error: %v", err)
	}
}

func TestRunPruneDuFailed(t *testing.T) {
	pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{
		DUFails: -1,
	}), true)
	err := pruner.runPrune(context.Background(), []string{"test"})
	if err != nil {
		t.Fatalf("Got an error: %v", err)
	}
}

func TestRunPruneDuFailed2(t *testing.T) {
	pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{
		DUFails: 2,
	}), true)
	err := pruner.runPrune(context.Background(), []string{"test"})
	if err != nil {
		t.Fatalf("Got an error: %v", err)
	}
}

func TestRunPruneImageRemoveFailed(t *testing.T) {
	pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{
		ErrImageRemove: true,
	}), true)
	err := pruner.runPrune(context.Background(), []string{"test"})
	if err == nil {
		t.Fatal("An error expected here")
	}
}

func TestIsPruned(t *testing.T) {
	pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{}), true)
	err := pruner.runPrune(context.Background(),
		[]string{"test1", "test2", "test1"})
	if err != nil {
		t.Fatalf("Got an error: %v", err)
	}
	if !pruner.isPruned("test1") {
		t.Error("Image test1 is expected to be pruned")
	}
	if pruner.isPruned("test3") {
		t.Error("Image test3 is not expected to be pruned")
	}
}

func TestIsPrunedFail(t *testing.T) {
	pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{
		ErrImageRemove: true,
	}), true)

	err := pruner.runPrune(context.Background(), []string{"test1"})
	if err == nil {
		t.Fatal("An error expected here")
	}
	if pruner.isPruned("test1") {
		t.Error("Image test1 is not expected to be pruned")
	}
}

func TestCollectPruneImages(t *testing.T) {
	tests := []struct {
		description     string
		localImages     map[string][]string
		imagesToBuild   []string
		expectedToPrune []string
	}{
		{
			description: "test images to prune",
			localImages: map[string][]string{
				"foo": {"111", "222", "333", "444"},
				"bar": {"555", "666", "777"},
			},
			imagesToBuild:   []string{"foo", "bar"},
			expectedToPrune: []string{"111", "222", "333", "555", "666"},
		},
		{
			description: "dup image ref",
			localImages: map[string][]string{
				"foo": {"111", "222", "333", "444"},
			},
			imagesToBuild:   []string{"foo", "foo"},
			expectedToPrune: []string{"111", "222"},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			pruner := newPruner(fakeLocalDaemon(&testutil.FakeAPIClient{
				LocalImages: test.localImages,
			}), true)

			res := pruner.collectImagesToPrune(
				context.Background(), test.imagesToBuild)
			sort.Strings(test.expectedToPrune)
			sort.Strings(res)
			t.CheckDeepEqual(res, test.expectedToPrune)
		})
	}
}
