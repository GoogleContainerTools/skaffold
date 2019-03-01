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

	"github.com/bmatcuk/doublestar"
	"github.com/sirupsen/logrus"

	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/watch"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	lenDigest = 71
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

func NewItem(a *latest.Artifact, e watch.Events, builds []build.Artifact) (*Item, error) {
	// If there are no changes, short circuit and don't sync anything
	if !e.HasChanged() || len(a.Sync) == 0 {
		return nil, nil
	}

	tag := latestTag(a.ImageName, builds)
	if tag == "" {
		return nil, fmt.Errorf("could not find latest tag for image %s in builds: %v", a.ImageName, builds)
	}

	wd, err := WorkingDir(a.ImageName, tag)
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

func retrieveWorkingDir(image, tagged string) (string, error) {
	fullyQualifedImage := stripTagIfDigestPresent(image, tagged)
	cf, err := docker.RetrieveRemoteConfig(fullyQualifedImage)
	if err != nil {
		return "", errors.Wrap(err, "retrieving remote config")

	}
	if cf.Config.WorkingDir == "" {
		return "/", nil
	}
	return cf.Config.WorkingDir, nil
}

// stripTagIfDigestPresent removes the tag from the image if there is a tag and a digest
func stripTagIfDigestPresent(image, tagged string) string {
	// try to parse the reference, return image if it works
	_, err := name.ParseReference(tagged, name.WeakValidation)
	if err == nil {
		return image
	}
	// strip out the tag
	digestIndex := strings.Index(tagged, "sha256:")
	if digestIndex == -1 {
		return image
	}
	digest := tagged[digestIndex : digestIndex+lenDigest]
	return fmt.Sprintf("%s@%s", image, digest)
}

func latestTag(image string, builds []build.Artifact) string {
	for _, build := range builds {
		if build.ImageName == image {
			return build.Tag
		}
	}
	return ""
}

func intersect(context string, syncMap map[string]string, files []string, workingDir string) (map[string]string, error) {
	ret := map[string]string{}

	for _, f := range files {
		relPath, err := filepath.Rel(context, f)
		if err != nil {
			return nil, errors.Wrapf(err, "changed file %s can't be found relative to context %s", f, context)
		}
		var matches bool
		for p, dst := range syncMap {
			match, err := doublestar.PathMatch(filepath.FromSlash(p), relPath)
			if err != nil {
				return nil, errors.Wrapf(err, "pattern error for %s", relPath)
			}
			if match {
				if filepath.IsAbs(dst) {
					dst = filepath.ToSlash(filepath.Join(dst, filepath.Base(relPath)))
				} else {
					dst = filepath.ToSlash(filepath.Join(workingDir, dst, filepath.Base(relPath)))
				}
				// Every file must match at least one sync pattern, if not we'll have to
				// skip the entire sync
				matches = true
				ret[f] = dst
			}
		}
		if !matches {
			logrus.Infof("Changed file %s does not match any sync pattern. Skipping sync", relPath)
			return nil, nil
		}

	}
	return ret, nil
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
					if err := util.RunCmd(cmd); err != nil {
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
