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
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/bmatcuk/doublestar"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// WorkingDir is here for testing
	WorkingDir = docker.RetrieveWorkingDir
)

func NewItem(a *latest.Artifact, e filemon.Events, builds []build.Artifact, insecureRegistries map[string]bool, destProvider DestinationProvider) (*Item, error) {
	if !e.HasChanged() || a.Sync == nil {
		return nil, nil
	}

	if len(a.Sync.Manual) > 0 {
		return manualSyncItem(a, e, builds, insecureRegistries)
	}

	if len(a.Sync.Infer) > 0 {
		return inferredSyncItem(a, e, builds, destProvider)
	}

	return nil, nil
}

func manualSyncItem(a *latest.Artifact, e filemon.Events, builds []build.Artifact, insecureRegistries map[string]bool) (*Item, error) {
	tag := latestTag(a.ImageName, builds)
	if tag == "" {
		return nil, fmt.Errorf("could not find latest tag for image %s in builds: %v", a.ImageName, builds)
	}

	containerWd, err := WorkingDir(tag, insecureRegistries)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving working dir for %s", tag)
	}

	toCopy, err := intersect(a.Workspace, containerWd, a.Sync.Manual, append(e.Added, e.Modified...))
	if err != nil {
		return nil, errors.Wrap(err, "intersecting sync map and added, modified files")
	}

	toDelete, err := intersect(a.Workspace, containerWd, a.Sync.Manual, e.Deleted)
	if err != nil {
		return nil, errors.Wrap(err, "intersecting sync map and deleted files")
	}

	// Something went wrong, don't sync, rebuild.
	if toCopy == nil || toDelete == nil {
		return nil, nil
	}

	return &Item{Image: tag, Copy: toCopy, Delete: toDelete}, nil
}

func inferredSyncItem(a *latest.Artifact, e filemon.Events, builds []build.Artifact, provider DestinationProvider) (*Item, error) {
	// deleted files are no longer contained in the syncMap, so we need to rebuild
	if len(e.Deleted) > 0 {
		return nil, nil
	}

	tag := latestTag(a.ImageName, builds)
	if tag == "" {
		return nil, fmt.Errorf("could not find latest tag for image %s in builds: %v", a.ImageName, builds)
	}

	syncMap, err := provider()
	if err != nil {
		return nil, errors.Wrapf(err, "inferring syncmap for image %s", a.ImageName)
	}

	toCopy := make(map[string][]string)
	for _, f := range append(e.Modified, e.Added...) {
		relPath, err := filepath.Rel(a.Workspace, f)
		if err != nil {
			return nil, errors.Wrapf(err, "finding changed file %s relative to context %s", f, a.Workspace)
		}

		matches := false
		for _, p := range a.Sync.Infer {
			matches, err = doublestar.PathMatch(filepath.FromSlash(p), relPath)
			if err != nil {
				return nil, errors.Wrapf(err, "pattern error for %s", relPath)
			}
			if matches {
				break
			}
		}
		if !matches {
			logrus.Infof("Changed file %s does not match any sync pattern. Skipping sync", relPath)
			return nil, nil
		}

		if dsts, ok := syncMap[relPath]; ok {
			toCopy[f] = dsts
		} else {
			logrus.Infof("Changed file %s is not syncable. Skipping sync", relPath)
			return nil, nil
		}
	}

	return &Item{Image: tag, Copy: toCopy}, nil
}

func latestTag(image string, builds []build.Artifact) string {
	for _, build := range builds {
		if build.ImageName == image {
			return build.Tag
		}
	}
	return ""
}

func intersect(contextWd, containerWd string, syncRules []*latest.SyncRule, files []string) (syncMap, error) {
	ret := make(syncMap)
	for _, f := range files {
		relPath, err := filepath.Rel(contextWd, f)
		if err != nil {
			return nil, errors.Wrapf(err, "changed file %s can't be found relative to context %s", f, contextWd)
		}

		dsts, err := matchSyncRules(syncRules, relPath, containerWd)
		if err != nil {
			return nil, err
		}

		if len(dsts) == 0 {
			logrus.Infof("Changed file %s does not match any sync pattern. Skipping sync", relPath)
			return nil, nil
		}

		ret[f] = dsts
	}
	return ret, nil
}

func matchSyncRules(syncRules []*latest.SyncRule, relPath, containerWd string) ([]string, error) {
	dsts := make([]string, 0, 1)
	for _, r := range syncRules {
		matches, err := doublestar.PathMatch(filepath.FromSlash(r.Src), relPath)
		if err != nil {
			return nil, errors.Wrapf(err, "pattern error for %s", relPath)
		}

		if !matches {
			continue
		}

		wd := ""
		if !path.IsAbs(r.Dest) {
			// Convert relative destinations to absolute via the working dir in the container.
			wd = containerWd
		}

		// Map the paths as a tree from the prefix.
		subPath := strings.TrimPrefix(filepath.ToSlash(relPath), r.Strip)
		dsts = append(dsts, path.Join(wd, r.Dest, subPath))
	}
	return dsts, nil
}

func (k *podSyncer) Sync(ctx context.Context, s *Item) error {
	if len(s.Copy) > 0 {
		logrus.Infoln("Copying files:", s.Copy, "to", s.Image)

		if err := Perform(ctx, s.Image, s.Copy, k.copyFileFn, k.namespaces); err != nil {
			return errors.Wrap(err, "copying files")
		}
	}

	if len(s.Delete) > 0 {
		logrus.Infoln("Deleting files:", s.Delete, "from", s.Image)

		if err := Perform(ctx, s.Image, s.Delete, k.deleteFileFn, k.namespaces); err != nil {
			return errors.Wrap(err, "deleting files")
		}
	}

	return nil
}

func Perform(ctx context.Context, image string, files syncMap, cmdFn func(context.Context, v1.Pod, v1.Container, syncMap) *exec.Cmd, namespaces []string) error {
	if len(files) == 0 {
		return nil
	}

	errs, ctx := errgroup.WithContext(ctx)

	client, err := kubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "getting kubernetes client")
	}

	numSynced := 0
	for _, ns := range namespaces {
		pods, err := client.CoreV1().Pods(ns).List(meta_v1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "getting pods for namespace "+ns)
		}

		for _, p := range pods.Items {
			if p.Status.Phase != v1.PodRunning {
				continue
			}

			for _, c := range p.Spec.Containers {
				if c.Image != image {
					continue
				}

				cmd := cmdFn(ctx, p, c, files)
				errs.Go(func() error {
					_, err := util.RunCmdOut(cmd)
					return err
				})
				numSynced++
			}
		}
	}

	if numSynced == 0 {
		return errors.New("didn't sync any files")
	}

	return errs.Wait()
}
