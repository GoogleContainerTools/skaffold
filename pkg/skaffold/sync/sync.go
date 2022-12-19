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
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/buildpacks"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/jib"
	kosync "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/ko/sync"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/hooks"
	kubernetesclient "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/client"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// For testing
var (
	WorkingDir = docker.RetrieveWorkingDir
	Labels     = docker.RetrieveLabels
	SyncMap    = syncMapForArtifact
)

func NewItem(ctx context.Context, a *latest.Artifact, e filemon.Events, builds []graph.Artifact, cfg docker.Config, dependentArtifactsCount int) (*Item, error) {
	if !e.HasChanged() || a.Sync == nil {
		return nil, nil
	}

	if dependentArtifactsCount > 0 {
		log.Entry(ctx).Warnf("Ignoring sync rules for image %q as it is being used as a required artifact for other images.", a.ImageName)
		return nil, nil
	}

	tag := latestTag(a.ImageName, builds)
	if tag == "" {
		return nil, fmt.Errorf("could not find latest tag for image %s in builds: %v", a.ImageName, builds)
	}

	switch {
	case len(a.Sync.Manual) > 0:
		return syncItem(ctx, a, tag, e, a.Sync.Manual, cfg)

	case a.Sync.Auto != nil:
		return autoSyncItem(ctx, a, tag, e, cfg)

	case len(a.Sync.Infer) > 0:
		return inferredSyncItem(ctx, a, tag, e, cfg)

	default:
		return nil, nil
	}
}

func syncItem(ctx context.Context, a *latest.Artifact, tag string, e filemon.Events, syncRules []*latest.SyncRule, cfg docker.Config) (*Item, error) {
	containerWd, err := WorkingDir(ctx, tag, cfg)
	if err != nil {
		return nil, fmt.Errorf("retrieving working dir for %q: %w", tag, err)
	}

	toCopy, err := intersect(ctx, a.Workspace, containerWd, syncRules, append(e.Added, e.Modified...))
	if err != nil {
		return nil, fmt.Errorf("intersecting sync map and added, modified files: %w", err)
	}

	toDelete, err := intersect(ctx, a.Workspace, containerWd, syncRules, e.Deleted)
	if err != nil {
		return nil, fmt.Errorf("intersecting sync map and deleted files: %w", err)
	}

	// Something went wrong, don't sync, rebuild.
	if toCopy == nil || toDelete == nil {
		return nil, nil
	}

	return &Item{Image: tag, Artifact: a, Copy: toCopy, Delete: toDelete}, nil
}

func inferredSyncItem(ctx context.Context, a *latest.Artifact, tag string, e filemon.Events, cfg docker.Config) (*Item, error) {
	// the ko builder doesn't need or use a sync map
	if a.KoArtifact != nil {
		log.Entry(ctx).Debugf("ko inferred sync %+v", e)
		toCopy, toDelete, err := kosync.Infer(ctx, a, e)
		if err != nil {
			return nil, err
		}
		return &Item{
			Image:    tag,
			Artifact: a,
			Copy:     toCopy,
			Delete:   toDelete,
		}, nil
	}

	// deleted files are no longer contained in the syncMap, so we need to rebuild
	if len(e.Deleted) > 0 {
		return nil, nil
	}

	syncMap, err := SyncMap(ctx, a, cfg)
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
			log.Entry(ctx).Infof("Changed file %s does not match any sync pattern. Skipping sync", relPath)
			return nil, nil
		}

		if dsts, ok := syncMap[relPath]; ok {
			toCopy[f] = dsts
		} else {
			log.Entry(ctx).Infof("Changed file %s is not syncable. Skipping sync", relPath)
			return nil, nil
		}
	}

	return &Item{Image: tag, Artifact: a, Copy: toCopy}, nil
}

func syncMapForArtifact(ctx context.Context, a *latest.Artifact, cfg docker.Config) (map[string][]string, error) {
	switch {
	case a.DockerArtifact != nil:
		return docker.SyncMap(ctx, a.Workspace, a.DockerArtifact.DockerfilePath, a.DockerArtifact.BuildArgs, cfg)

	case a.CustomArtifact != nil:
		if a.CustomArtifact.Dependencies == nil || a.CustomArtifact.Dependencies.Dockerfile == nil {
			return nil, build.ErrCustomBuildNoDockerfile{}
		}
		return docker.SyncMap(ctx, a.Workspace, a.CustomArtifact.Dependencies.Dockerfile.Path, a.CustomArtifact.Dependencies.Dockerfile.BuildArgs, cfg)

	case a.KanikoArtifact != nil:
		return docker.SyncMap(ctx, a.Workspace, a.KanikoArtifact.DockerfilePath, a.KanikoArtifact.BuildArgs, cfg)

	default:
		return nil, build.ErrSyncMapNotSupported{}
	}
}

func autoSyncItem(ctx context.Context, a *latest.Artifact, tag string, e filemon.Events, cfg docker.Config) (*Item, error) {
	switch {
	case a.BuildpackArtifact != nil:
		labels, err := Labels(ctx, tag, cfg)
		if err != nil {
			return nil, fmt.Errorf("retrieving labels for %q: %w", tag, err)
		}

		rules, err := buildpacks.SyncRules(labels)
		if err != nil {
			return nil, fmt.Errorf("extracting sync rules from labels for %q: %w", tag, err)
		}

		return syncItem(ctx, a, tag, e, rules, cfg)

	case a.JibArtifact != nil:
		toCopy, toDelete, err := jib.GetSyncDiff(ctx, a.Workspace, a.JibArtifact, e)
		if err != nil {
			return nil, err
		}
		if toCopy == nil && toDelete == nil {
			// do a rebuild
			return nil, nil
		}
		return &Item{Image: tag, Artifact: a, Copy: toCopy, Delete: toDelete}, nil

	default:
		// TODO: this error does appear a little late in the build, perhaps it could surface at first run, rather than first sync?
		return nil, fmt.Errorf("Sync: Auto is not supported by the build of %s", a.ImageName)
	}
}

func latestTag(image string, builds []graph.Artifact) string {
	for _, build := range builds {
		if build.ImageName == image {
			return build.Tag
		}
	}
	return ""
}

func intersect(ctx context.Context, contextWd, containerWd string, syncRules []*latest.SyncRule, files []string) (syncMap, error) {
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
			log.Entry(ctx).Infof("Changed file %s does not match any sync pattern. Skipping sync", relPath)
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

func (s *PodSyncer) Sync(ctx context.Context, out io.Writer, item *Item) error {
	if !item.HasChanges() {
		return nil
	}

	var copy, delete []string
	for k := range item.Copy {
		copy = append(copy, k)
	}
	for k := range item.Delete {
		delete = append(delete, k)
	}

	opts, err := hooks.NewSyncEnvOpts(item.Artifact, item.Image, copy, delete, *s.namespaces, s.kubectl.KubeContext)
	if err != nil {
		return err
	}
	hooksRunner := hooks.NewSyncRunner(s.kubectl, item.Artifact.ImageName, item.Image, *s.namespaces, s.formatter, item.Artifact.Sync.LifecycleHooks, opts)
	if err := hooksRunner.RunPreHooks(ctx, out); err != nil {
		return fmt.Errorf("pre-sync hooks failed for artifact %q: %w", item.Artifact.ImageName, err)
	}
	if err := s.sync(ctx, item); err != nil {
		return err
	}
	if err := hooksRunner.RunPostHooks(ctx, out); err != nil {
		return fmt.Errorf("post-sync hooks failed for artifact %q: %w", item.Artifact.ImageName, err)
	}
	return nil
}

func (s *PodSyncer) sync(ctx context.Context, item *Item) error {
	if len(item.Copy) > 0 {
		log.Entry(ctx).Info("Copying files:", item.Copy, "to", item.Image)

		if err := Perform(ctx, item.Image, item.Copy, s.copyFileFn, *s.namespaces, s.kubectl.KubeContext); err != nil {
			return fmt.Errorf("copying files: %w", err)
		}
	}

	if len(item.Delete) > 0 {
		log.Entry(ctx).Info("Deleting files:", item.Delete, "from", item.Image)

		if err := Perform(ctx, item.Image, item.Delete, s.deleteFileFn, *s.namespaces, s.kubectl.KubeContext); err != nil {
			return fmt.Errorf("deleting files: %w", err)
		}
	}

	return nil
}

func Perform(ctx context.Context, image string, files syncMap, cmdFn func(context.Context, v1.Pod, v1.Container, syncMap) *exec.Cmd, namespaces []string, kubeContext string) error {
	if len(files) == 0 {
		return nil
	}

	errs, ctx := errgroup.WithContext(ctx)

	client, err := kubernetesclient.Client(kubeContext)
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	numSynced := 0
	for _, ns := range namespaces {
		pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
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
					_, err := util.RunCmdOut(ctx, cmd)
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
		log.Entry(ctx).Warnf("sync failed for artifact %q", image)
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
