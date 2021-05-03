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

package test

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	latest_v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	"k8s.io/client-go/kubernetes"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
)

type TriggerState struct {
	Build  bool
	Sync   bool
	Deploy bool
}

type Actions struct {
	Built    []string
	Synced   []string
	Tested   []string
	Deployed []string
}

type TestBench struct {
	BuildErrors   []error
	syncErrors    []error
	TestErrors    []error
	DeployErrors  []error
	namespaces    []string
	UserIntents   []func(intents *runner.Intents)
	Intents       *runner.Intents
	IntentTrigger bool

	DevLoop        func(context.Context, io.Writer, func() error) error
	FirstMonitor   func(bool) error
	Cycles         int
	CurrentCycle   int
	currentActions Actions
	actions        []Actions
	tag            int
}

func NewTestBench() *TestBench {
	return &TestBench{}
}

func (t *TestBench) WithBuildErrors(buildErrors []error) *TestBench {
	t.BuildErrors = buildErrors
	return t
}

func (t *TestBench) WithSyncErrors(syncErrors []error) *TestBench {
	t.syncErrors = syncErrors
	return t
}

func (t *TestBench) WithDeployErrors(deployErrors []error) *TestBench {
	t.DeployErrors = deployErrors
	return t
}

func (t *TestBench) WithDeployNamespaces(ns []string) *TestBench {
	t.namespaces = ns
	return t
}

func (t *TestBench) WithTestErrors(testErrors []error) *TestBench {
	t.TestErrors = testErrors
	return t
}

func (t *TestBench) TestDependencies(*latest_v1.Artifact) ([]string, error) { return nil, nil }
func (t *TestBench) Dependencies() ([]string, error)                        { return nil, nil }
func (t *TestBench) Cleanup(ctx context.Context, out io.Writer) error       { return nil }
func (t *TestBench) Prune(ctx context.Context, out io.Writer) error         { return nil }

func (t *TestBench) enterNewCycle() {
	t.actions = append(t.actions, t.currentActions)
	t.currentActions = Actions{}
}

func (t *TestBench) Build(_ context.Context, _ io.Writer, _ tag.ImageTags, artifacts []*latest_v1.Artifact) ([]graph.Artifact, error) {
	if len(t.BuildErrors) > 0 {
		err := t.BuildErrors[0]
		t.BuildErrors = t.BuildErrors[1:]
		if err != nil {
			return nil, err
		}
	}

	t.tag++

	var builds []graph.Artifact
	for _, artifact := range artifacts {
		builds = append(builds, graph.Artifact{
			ImageName: artifact.ImageName,
			Tag:       fmt.Sprintf("%s:%d", artifact.ImageName, t.tag),
		})
	}

	t.currentActions.Built = findTags(builds)
	return builds, nil
}

func (t *TestBench) Sync(_ context.Context, item *sync.Item) error {
	if len(t.syncErrors) > 0 {
		err := t.syncErrors[0]
		t.syncErrors = t.syncErrors[1:]
		if err != nil {
			return err
		}
	}

	t.currentActions.Synced = []string{item.Image}
	return nil
}

func (t *TestBench) Test(_ context.Context, _ io.Writer, artifacts []graph.Artifact) error {
	if len(t.TestErrors) > 0 {
		err := t.TestErrors[0]
		t.TestErrors = t.TestErrors[1:]
		if err != nil {
			return err
		}
	}

	t.currentActions.Tested = findTags(artifacts)
	return nil
}

func (t *TestBench) Deploy(_ context.Context, _ io.Writer, artifacts []graph.Artifact) ([]string, error) {
	if len(t.DeployErrors) > 0 {
		err := t.DeployErrors[0]
		t.DeployErrors = t.DeployErrors[1:]
		if err != nil {
			return nil, err
		}
	}

	t.currentActions.Deployed = findTags(artifacts)
	return t.namespaces, nil
}

func (t *TestBench) Render(context.Context, io.Writer, []graph.Artifact, bool, string) error {
	return nil
}

func (t *TestBench) Actions() []Actions {
	return append(t.actions, t.currentActions)
}

func (t *TestBench) WatchForChanges(ctx context.Context, out io.Writer, doDev func() error) error {
	// don't actually call the monitor here, because extra actions would be added
	if err := t.FirstMonitor(true); err != nil {
		return err
	}

	t.IntentTrigger = true
	for _, intent := range t.UserIntents {
		intent(t.Intents)
		if err := t.DevLoop(ctx, out, doDev); err != nil {
			return err
		}
	}

	t.IntentTrigger = false
	for i := 0; i < t.Cycles; i++ {
		t.enterNewCycle()
		t.CurrentCycle = i
		if err := t.DevLoop(ctx, out, doDev); err != nil {
			return err
		}
	}
	return nil
}

func (t *TestBench) LogWatchToUser(_ io.Writer) {}

func findTags(artifacts []graph.Artifact) []string {
	var tags []string
	for _, artifact := range artifacts {
		tags = append(tags, artifact.Tag)
	}
	return tags
}

func MockK8sClient() (kubernetes.Interface, error) {
	return fakekubeclientset.NewSimpleClientset(), nil
}
