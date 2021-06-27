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
package generate

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// NewGenerator instantiates a Generator object.
func NewGenerator(workingDir string, config latestV2.Generate) *Generator {
	return &Generator{
		workingDir: workingDir,
		config:     config,
	}
}

// Generator provides the functions for the manifest sources (raw manifests, helm charts, kustomize configs and remote packages).
type Generator struct {
	workingDir string
	config     latestV2.Generate
}

// Generate parses the config resources from the paths in .Generate.Manifests. This path can be the path to raw manifest,
// kustomize manifests, helm charts or kpt function configs. All should be file-watched.
func (g *Generator) Generate(ctx context.Context) (manifest.ManifestList, error) {
	// exclude remote url.
	var paths []string
	// TODO(yuwenma): Apply new UX, kustomize kpt and helm
	for _, path := range g.config.RawK8s {
		switch {
		case util.IsURL(path):
			// TODO(yuwenma): remote URL should be changed to use kpt package management approach, via API Schema
			//  `render.generate.remotePackages`
		case strings.HasPrefix(path, "gs://"):
			// TODO(yuwenma): handle GS packages.
		default:
			paths = append(paths, path)
		}
	}
	// expend the glob paths.
	expanded, err := util.ExpandPathsGlob(g.workingDir, paths)
	if err != nil {
		return nil, err
	}

	// Parse kustomize manifests and non-kustomize manifests.  We may also want to parse (and exclude) kpt function manifests later.
	// TODO: Update `kustomize build` to kustomize kpt-fn once https://github.com/GoogleContainerTools/kpt/issues/1447 is fixed.
	kustomizePathMap := make(map[string]bool)
	var nonKustomizePaths []string
	for _, path := range expanded {
		if dir, ok := isKustomizeDir(path); ok {
			kustomizePathMap[dir] = true
		}
	}
	for _, path := range expanded {
		kustomizeDirDup := false
		for kPath := range kustomizePathMap {
			// Before kustomize kpt-fn can provide a way to parse the kustomize content, we assume the users do not place non-kustomize manifests under the kustomization.yaml directory.
			if strings.HasPrefix(path, kPath) {
				kustomizeDirDup = true
				break
			}
		}
		if !kustomizeDirDup {
			nonKustomizePaths = append(nonKustomizePaths, path)
		}
	}

	var manifests manifest.ManifestList
	for kPath := range kustomizePathMap {
		// TODO:  support kustomize buildArgs (shall we support it in kpt-fn)?
		cmd := exec.CommandContext(ctx, "kustomize", "build", kPath)
		out, err := util.RunCmdOut(cmd)
		if err != nil {
			return nil, err
		}
		if len(out) == 0 {
			continue
		}
		manifests.Append(out)
	}
	for _, nkPath := range nonKustomizePaths {
		if !kubernetes.HasKubernetesFileExtension(nkPath) {
			if !util.StrSliceContains(g.config.RawK8s, nkPath) {
				logrus.Infof("refusing to deploy/delete non {json, yaml} file %s", nkPath)
				logrus.Info("If you still wish to deploy this file, please specify it directly, outside a glob pattern.")
				continue
			}
		}
		manifestFileContent, err := ioutil.ReadFile(nkPath)
		if err != nil {
			return nil, err
		}
		manifests.Append(manifestFileContent)
	}
	// TODO(yuwenma): helm resources. `render.generate.helmCharts`
	return manifests, nil
}

// isKustomizeDir checks if the path is managed by kustomize. A more reliable approach is parsing the kustomize content
// resources, bases, overlays. However, this switches the manifests parsing from kustomize/kpt to skaffold. To avoid
// skaffold render.generate mis-use, we expect the users do not place non-kustomize manifests under the kustomization.yaml directory, so as the kpt manifests.
func isKustomizeDir(path string) (string, bool) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	var dir string
	switch mode := fileInfo.Mode(); {
	// TODO: Check if regular file contains kpt functions. if so, we may want to abstract that info as well.
	case mode.IsDir():
		dir = path
	case mode.IsRegular():
		dir = filepath.Dir(path)
	}

	for _, base := range kustomize.KustomizeFilePaths {
		if _, err := os.Stat(filepath.Join(dir, base)); os.IsNotExist(err) {
			continue
		}
		return dir, true
	}
	return "", false
}
