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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/bmatcuk/doublestar"
	registry_v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// WorkingDir is here for testing
	WorkingDir = retrieveWorkingDir
)

type Syncer interface {
	Sync(context.Context, *Item) error
}

type Item struct {
	Image  string
	Copy   map[string]string
	Delete map[string]string
}

func NewItem(a *latest.Artifact, e watch.Events, builds []build.Artifact, insecureRegistries map[string]bool) (*Item, error) {
	// If there are no changes, short circuit and don't sync anything
	if !e.HasChanged() || len(a.Sync) == 0 {
		return nil, nil
	}

	tag := latestTag(a.ImageName, builds)
	if tag == "" {
		return nil, fmt.Errorf("could not find latest tag for image %s in builds: %v", a.ImageName, builds)
	}

	wd, err := WorkingDir(tag, insecureRegistries)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving working dir for %s", tag)
	}

	toCopy, err := intersect(a.Workspace, a.Sync, append(e.Added, e.Modified...), wd)
	if err != nil {
		return nil, errors.Wrap(err, "intersecting sync map and added, modified files")
	}

	toDelete, err := intersect(a.Workspace, a.Sync, e.Deleted, wd)
	if err != nil {
		return nil, errors.Wrap(err, "intersecting sync map and deleted files")
	}

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

func retrieveWorkingDir(tagged string, insecureRegistries map[string]bool) (string, error) {
	var cf *registry_v1.ConfigFile
	var err error

	localDocker, err := docker.NewAPIClient(false, insecureRegistries)
	if err != nil {
		// No local Docker is available
		cf, err = docker.RetrieveRemoteConfig(tagged, insecureRegistries)
	} else {
		cf, err = localDocker.ConfigFile(context.Background(), tagged)
	}
	if err != nil {
		return "", errors.Wrap(err, "retrieving image config")
	}

	if cf.Config.WorkingDir == "" {
		return "/", nil
	}
	return cf.Config.WorkingDir, nil
}

func latestTag(image string, builds []build.Artifact) string {
	for _, build := range builds {
		if build.ImageName == image {
			return build.Tag
		}
	}
	return ""
}

// Note that we always use Unix-style paths in our destination.
func slashJoin(pfx, sfx string) string {
	if pfx == "." || pfx == "" {
		return sfx
	}
	elems := []string{
		strings.TrimSuffix(pfx, "/"),
		sfx,
	}
	return strings.Join(elems, "/")
}

func intersect(context string, syncMap map[string]string, files []string, workingDir string) (map[string]string, error) {
	ret := map[string]string{}

	tripleStarSyncMap, otherSyncMap := segregateSyncMaps(syncMap)
	for _, f := range files {
		relPath, err := filepath.Rel(context, f)
		if err != nil {
			return nil, errors.Wrapf(err, "changed file %s can't be found relative to context %s", f, context)
		}

		var match bool

		// First try all tripleStarSyncMaps.
		match, dst, err := matchTripleStarSyncMap(tripleStarSyncMap, relPath)
		if err != nil {
			return nil, err
		}

		if !match {
			// Try matching the other rules.
			match, dst, err = matchOtherSyncMap(otherSyncMap, relPath)
			if err != nil {
				return nil, err
			}
		}

		if !match {
			logrus.Infof("Changed file %s does not match any sync pattern. Skipping sync", relPath)
			return nil, nil
		}

		// Convert relative destinations to absolute via the workingDir.
		if dst[0] != '/' {
			dst = slashJoin(workingDir, dst)
		}

		// Record the final destination.
		ret[f] = dst
	}

	return ret, nil
}

func segregateSyncMaps(syncMap map[string]string) (tripleStarPattern, doubleStarPattern map[string]string) {
	tripleStarPattern = make(map[string]string)
	doubleStarPattern = make(map[string]string)
	for p, dst := range syncMap {
		if strings.Contains(p, "***") {
			tripleStarPattern[p] = dst
		} else {
			doubleStarPattern[p] = dst
		}
	}
	return
}

func matchTripleStarSyncMap(syncMap map[string]string, relPath string) (bool, string, error) {
	for p, dst := range syncMap {
		pat := strings.Replace(p, "***", "**", -1)
		match, err := doublestar.PathMatch(filepath.FromSlash(pat), relPath)
		if err != nil {
			return false, "", errors.Wrapf(err, "pattern error for %s", relPath)
		}

		if match {
			// Map the paths as a tree from the prefix.
			subtreePrefix := strings.Split(p, "***")[0]
			subPath := strings.TrimPrefix(filepath.ToSlash(relPath), subtreePrefix)
			return true, slashJoin(dst, subPath), nil
		}
	}
	return false, "", nil
}

func matchOtherSyncMap(syncMap map[string]string, relPath string) (bool, string, error) {
	for p, dst := range syncMap {
		match, err := doublestar.PathMatch(filepath.FromSlash(p), relPath)
		if err != nil {
			return false, "", errors.Wrapf(err, "pattern error for %s", relPath)
		}

		if match {
			// Collapse the paths.
			subPath := filepath.Base(relPath)
			return true, slashJoin(dst, filepath.ToSlash(subPath)), nil
		}
	}
	return false, "", nil
}

func Perform(ctx context.Context, image string, files map[string]string, cmdFn func(context.Context, v1.Pod, v1.Container, map[string]string) []*exec.Cmd, namespaces []string) error {
	if len(files) == 0 {
		return nil
	}

	client, err := kubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "getting k8s client")
	}

	numSynced := 0
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

				cmds := cmdFn(ctx, p, c, files)
				for _, cmd := range cmds {
					if err := util.RunCmdOut(cmd); err != nil {
						return err
					}
					numSynced++
				}
			}
		}
	}

	if numSynced == 0 {
		return errors.New("didn't sync any files")
	}

	return nil
}
