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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// NewGenerator instantiates a Generator object.
func NewGenerator(workingDir string, config latestV2.Generate, hydrationDir string) *Generator {
	return &Generator{
		workingDir:   workingDir,
		hydrationDir: hydrationDir,
		config:       config,
	}
}

// Generator provides the functions for the manifest sources (raw manifests, helm charts, kustomize configs and remote packages).
type Generator struct {
	workingDir   string
	hydrationDir string
	config       latestV2.Generate
}

func excludeRemote(paths []string) []string {
	var localPaths []string
	for _, path := range paths {
		switch {
		case util.IsURL(path):
			// TODO(yuwenma): remote URL should be changed to use kpt package management approach, via API Schema
			//  `render.generate.remotePackages`
		case strings.HasPrefix(path, "gs://"):
			// TODO(yuwenma): handle GS packages.
		default:
			localPaths = append(localPaths, path)
		}
	}
	return localPaths
}

// Generate parses the config resources from the paths in .Generate.Manifests. This path can be the path to raw manifest,
// kustomize manifests, helm charts or kpt function configs. All should be file-watched.
func (g *Generator) Generate(ctx context.Context, out io.Writer) (manifest.ManifestList, error) {
	var manifests manifest.ManifestList

	// Generate kustomize Manifests
	_, endTrace := instrumentation.StartTrace(ctx, "Render_expandGlobKustomizeManifests")
	kustomizePaths := excludeRemote(g.config.Kustomize)
	kustomizePaths, err := util.ExpandPathsGlob(g.workingDir, kustomizePaths)
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not expand the glob kustomize manifests: %w", err))
		return nil, err
	}
	endTrace()
	kustomizePathMap := make(map[string]bool)
	for _, path := range kustomizePaths {
		if dir, ok := isKustomizeDir(path); ok {
			kustomizePathMap[dir] = true
		}
	}
	for kPath := range kustomizePathMap {
		// TODO: kustomize kpt-fn not available yet. See https://github.com/GoogleContainerTools/kpt/issues/1447
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

	// Generate in-place hydrated kpt Manifests
	kptPaths := excludeRemote(g.config.Kpt)
	_, endTrace = instrumentation.StartTrace(ctx, "Render_expandGlobKptManifests")
	kptPaths, err = util.ExpandPathsGlob(g.workingDir, kptPaths)
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not expand the glob kpt manifests: %w", err))
		return nil, err
	}
	endTrace()
	kptPathMap := make(map[string]bool)
	for _, path := range kptPaths {
		if dir, ok := isKptDir(path); ok {
			kptPathMap[dir] = true
		}
	}
	var rawManifestsFromKpt []string
	for kPath := range kptPathMap {
		// kpt manifests will be hydrated and stored in the subdir of the hydrated dir, where the subdir name
		// matches the kPath dir name.
		outputDir := filepath.Join(g.hydrationDir, filepath.Base(kPath))
		_, endTrace := instrumentation.StartTrace(ctx, "Render_generateKptManifests")
		cmd := exec.CommandContext(ctx, "kpt", "fn", "render", kPath,
			fmt.Sprintf("--output=%v", outputDir))
		cmd.Stderr = out
		if err = util.RunCmd(cmd); err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}
		rawManifestsFromKpt = append(rawManifestsFromKpt, outputDir)
	}

	// Generate Raw Manifests
	rawK8sPaths := excludeRemote(g.config.RawK8s)
	rawK8sPaths = append(rawK8sPaths, rawManifestsFromKpt...)
	_, endTrace = instrumentation.StartTrace(ctx, "Render_expandGlobRawManifests")
	rawK8sPaths, err = util.ExpandPathsGlob(g.workingDir, rawK8sPaths)
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not expand the glob raw manifests: %w", err))
		return nil, err
	}
	endTrace()
	for _, nkPath := range rawK8sPaths {
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

func isKptDir(path string) (string, bool) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	var dir string
	switch mode := fileInfo.Mode(); {
	case mode.IsDir():
		dir = path
	case mode.IsRegular():
		dir = filepath.Dir(path)
	}
	if _, err := os.Stat(filepath.Join(dir, kptfile.KptFileName)); os.IsNotExist(err) {
		return "", false
	}
	return dir, true
}
