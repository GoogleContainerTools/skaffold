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
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/jib"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// For testing
var (
	WorkingDir = docker.RetrieveWorkingDir
	Labels     = docker.RetrieveLabels
	SyncMap    = syncMapForArtifact
)

func NewItem(ctx context.Context, a *latest.Artifact, e filemon.Events, builds []build.Artifact, insecureRegistries map[string]bool) (*Item, error) {
	if !e.HasChanged() || a.Sync == nil {
		return nil, nil
	}

	tag := latestTag(a.ImageName, builds)
	if tag == "" {
		return nil, fmt.Errorf("could not find latest tag for image %s in builds: %v", a.ImageName, builds)
	}

	switch {
	case len(a.Sync.Manual) > 0:
		return syncItem(a, tag, e, a.Sync.Manual, insecureRegistries)

	case a.Sync.Auto != nil:
		return autoSyncItem(ctx, a, tag, e, insecureRegistries)

	case len(a.Sync.Infer) > 0:
		return inferredSyncItem(a, tag, e, insecureRegistries)

	default:
		return nil, nil
	}
}

func syncItem(a *latest.Artifact, tag string, e filemon.Events, syncRules []*latest.SyncRule, insecureRegistries map[string]bool) (*Item, error) {
	containerWd, err := WorkingDir(tag, insecureRegistries)
	if err != nil {
		return nil, fmt.Errorf("retrieving working dir for %q: %w", tag, err)
	}

	toCopy, err := intersect(a.Workspace, containerWd, syncRules, append(e.Added, e.Modified...))
	if err != nil {
		return nil, fmt.Errorf("intersecting sync map and added, modified files: %w", err)
	}

	toDelete, err := intersect(a.Workspace, containerWd, syncRules, e.Deleted)
	if err != nil {
		return nil, fmt.Errorf("intersecting sync map and deleted files: %w", err)
	}

	// Something went wrong, don't sync, rebuild.
	if toCopy == nil || toDelete == nil {
		return nil, nil
	}

	return &Item{Image: tag, Copy: toCopy, Delete: toDelete}, nil
}

func inferredSyncItem(a *latest.Artifact, tag string, e filemon.Events, insecureRegistries map[string]bool) (*Item, error) {
	// deleted files are no longer contained in the syncMap, so we need to rebuild
	if len(e.Deleted) > 0 {
		return nil, nil
	}

	syncMap, err := SyncMap(a, insecureRegistries)
	if err != nil {
		return nil, fmt.Errorf("inferring syncmap for image %q: %w", a.ImageName, err)
	}

	toCopy := make(map[string][]string)
	for _, f := range append(e.Modified, e.Added...) {
		relPath, err := filepath.Rel(a.Workspace, f)
		if err != nil {
			return nil, fmt.Errorf("finding changed file %s relative to context %q: %w", f, a.Workspace, err)
		}

		matches := false
		for _, p := range a.Sync.Infer {
			matches, err = doublestar.PathMatch(filepath.FromSlash(p), relPath)
			if err != nil {
				return nil, fmt.Errorf("pattern error for %q: %w", relPath, err)
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

func syncMapForArtifact(a *latest.Artifact, insecureRegistries map[string]bool) (map[string][]string, error) {
	switch {
	case a.DockerArtifact != nil:
		return docker.SyncMap(a.Workspace, a.DockerArtifact.DockerfilePath, a.DockerArtifact.BuildArgs, insecureRegistries)

	case a.CustomArtifact != nil && a.CustomArtifact.Dependencies != nil && a.CustomArtifact.Dependencies.Dockerfile != nil:
		return docker.SyncMap(a.Workspace, a.CustomArtifact.Dependencies.Dockerfile.Path, a.CustomArtifact.Dependencies.Dockerfile.BuildArgs, insecureRegistries)

	case a.KanikoArtifact != nil:
		return docker.SyncMap(a.Workspace, a.KanikoArtifact.DockerfilePath, a.KanikoArtifact.BuildArgs, insecureRegistries)

	default:
		return nil, build.ErrSyncMapNotSupported{}
	}
}

func autoSyncItem(ctx context.Context, a *latest.Artifact, tag string, e filemon.Events, insecureRegistries map[string]bool) (*Item, error) {
	switch {
	case a.BuildpackArtifact != nil:
		labels, err := Labels(tag, insecureRegistries)
		if err != nil {
			return nil, fmt.Errorf("retrieving labels for %q: %w", tag, err)
		}

		rules, err := buildpacks.SyncRules(labels)
		if err != nil {
			return nil, fmt.Errorf("extracting sync rules from labels for %q: %w", tag, err)
		}

		return syncItem(a, tag, e, rules, insecureRegistries)

	case a.JibArtifact != nil:
		toCopy, toDelete, err := jib.GetSyncDiff(ctx, a.Workspace, a.JibArtifact, e)
		if err != nil {
			return nil, err
		}
		if toCopy == nil && toDelete == nil {
			// do a rebuild
			return nil, nil
		}
		return &Item{Image: tag, Copy: toCopy, Delete: toDelete}, nil

	default:
		// TODO: this error does appear a little late in the build, perhaps it could surface at first run, rather than first sync?
		return nil, fmt.Errorf("Sync: Auto is not supported by the build of %s", a.ImageName)
	}
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
			return nil, fmt.Errorf("changed file %s can't be found relative to context %q: %w", f, contextWd, err)
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
			return nil, fmt.Errorf("pattern error for %q: %w", relPath, err)
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

func (s *podSyncer) Sync(ctx context.Context, item *Item) error {
	if len(item.Copy) > 0 {
		logrus.Infoln("Copying files:", item.Copy, "to", item.Image)

		if err := Perform(ctx, item.Image, item.Copy, s.copyFileFn, s.namespaces); err != nil {
			return fmt.Errorf("copying files: %w", err)
		}
	}

	if len(item.Delete) > 0 {
		logrus.Infoln("Deleting files:", item.Delete, "from", item.Image)

		if err := Perform(ctx, item.Image, item.Delete, s.deleteFileFn, s.namespaces); err != nil {
			return fmt.Errorf("deleting files: %w", err)
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
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	numSynced := 0
	for _, ns := range namespaces {
		pods, err := client.CoreV1().Pods(ns).List(metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("getting pods for namespace %q: %w", ns, err)
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

	if err := errs.Wait(); err != nil {
		return err
	}

	if numSynced == 0 {
		return errors.New("didn't sync any files")
	}
	return nil
}

func Init(ctx context.Context, artifacts []*latest.Artifact) error {
	for _, a := range artifacts {
		if a.Sync == nil {
			continue
		}

		if a.Sync.Auto != nil && a.JibArtifact != nil {
			err := jib.InitSync(ctx, a.Workspace, a.JibArtifact)
			if err != nil {
				return fmt.Errorf("failed to initialize sync state for %q: %w", a.ImageName, err)
			}
		}
	}
	return nil
}
