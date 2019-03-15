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

package sync

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Syncer interface {
	Sync(context.Context, *Item) error
}

type Item struct {
	Image  string
	Copy   map[string][]string
	Delete map[string][]string
}

type DependencyResolver func() (map[string][]string, error)

func NewItem(a *latest.Artifact, e watch.Events, builds []build.Artifact, deps DependencyResolver) (*Item, error) {
	// If there are no changes, short circuit and don't sync anything
	if !e.HasChanged() || len(a.Sync) == 0 {
		return nil, nil
	}

	tag := latestTag(a.ImageName, builds)
	if tag == "" {
		return nil, fmt.Errorf("could not find latest tag for image %s in builds: %v", a.ImageName, builds)
	}

	dependencies, err := deps()
	if err != nil {
		return nil, errors.Wrapf(err, "resolving dependencies for %s", tag)
	}

	toCopy := intersect(a.Workspace, dependencies, append(e.Added, e.Modified...))
	toDelete := intersect(a.Workspace, dependencies, e.Deleted)

	// Something went wrong, don't sync, rebuild.
	if toCopy == nil || toDelete == nil {
		return nil, nil
	}

	return &Item{
		Image:  tag,
		Copy:   toCopy,
		Delete: toDelete,
	}, nil
}

func latestTag(image string, builds []build.Artifact) string {
	for _, build := range builds {
		if build.ImageName == image {
			return build.Tag
		}
	}
	return ""
}

func intersect(context string, deps map[string][]string, files []string) map[string][]string {
	ret := map[string][]string{}
	for _, f := range files {
		var matches bool
		for p, dsts := range deps {
			if p == f {
				// Every file must match at least one sync pattern, if not we'll have to
				// skip the entire sync
				matches = true
				ret[f] = dsts
			}
		}
		if !matches {
			relPath, _ := filepath.Rel(context, f)
			logrus.Infof("Changed file %s does not match any sync pattern. Skipping sync", relPath)
			return nil
		}
	}
	return ret
}

func Perform(ctx context.Context, image string, files map[string][]string, cmdFn func(context.Context, v1.Pod, v1.Container, map[string][]string) *exec.Cmd, namespaces []string) error {
	if len(files) == 0 {
		return nil
	}

	client, err := kubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "getting k8s client")
	}

	for _, ns := range namespaces {
		pods, err := client.CoreV1().Pods(ns).List(meta_v1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "getting pods for namespace "+ns)
		}

		for _, p := range pods.Items {
			for _, c := range p.Spec.Containers {
				if c.Image != image {
					continue
				}

				cmd := cmdFn(ctx, p, c, files)
				if err := util.RunCmd(cmd); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
