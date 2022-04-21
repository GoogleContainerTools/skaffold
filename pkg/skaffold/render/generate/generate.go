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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/render/kptfile"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringset"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
)

// NewGenerator instantiates a Generator object.
func NewGenerator(workingDir string, config latest.Generate) Generator {
	return Generator{
		workingDir: workingDir,
		config:     config,
	}
}

// Generator provides the functions for the manifest sources (raw manifests, helm charts, kustomize configs and remote packages).
type Generator struct {
	workingDir   string
	hydrationDir string
	config       latest.Generate
}

func resolveRemoteAndLocal(paths []string, workdir string) ([]string, error) {
	var localPaths []string
	var gcsManifests []string
	for _, path := range paths {
		switch {
		case util.IsURL(path):
		case strings.HasPrefix(path, "gs://"):
			gcsManifests = append(gcsManifests, path)
		default:
			localPaths = append(localPaths, path)
		}
	}
	list, err := util.ExpandPathsGlob(workdir, localPaths)
	if err != nil {
		return nil, err
	}
	if len(gcsManifests) != 0 {
		// return tmp dir of the downloaded manifests
		tmpDir, err := manifest.DownloadFromGCS(gcsManifests)
		if err != nil {
			return nil, fmt.Errorf("downloading from GCS: %w", err)
		}
		l, err := util.ExpandPathsGlob(tmpDir, []string{"*"})
		if err != nil {
			return nil, fmt.Errorf("expanding kubectl manifest paths: %w", err)
		}
		list = append(list, l...)
	}
	return list, nil
}

// Generate parses the config resources from the paths in .Generate.Manifests. This path can be the path to raw manifest,
// kustomize manifests, helm charts or kpt function configs. All should be file-watched.
func (g Generator) Generate(ctx context.Context, out io.Writer) (manifest.ManifestList, error) {
	var manifests manifest.ManifestList

	// Generate kustomize Manifests
	_, endTrace := instrumentation.StartTrace(ctx, "Render_expandGlobKustomizeManifests")
	kustomizePaths, err := resolveRemoteAndLocal(g.config.Kustomize, g.workingDir)
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
		out, err := util.RunCmdOut(ctx, cmd)
		if err != nil {
			return nil, err
		}
		if len(out) == 0 {
			continue
		}
		manifests.Append(out)
	}

	// Generate in-place hydrated kpt Manifests
	_, endTrace = instrumentation.StartTrace(ctx, "Render_expandGlobKptManifests")
	kptPaths, err := resolveRemoteAndLocal(g.config.Kpt, g.workingDir)
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
	var kptManifests []string
	for kPath := range kptPathMap {
		// kpt manifests will be hydrated and stored in the subdir of the hydrated dir, where the subdir name
		// matches the kPath dir name.
		outputDir := filepath.Join(g.hydrationDir, filepath.Base(kPath))
		tCtx, endTrace := instrumentation.StartTrace(ctx, "Render_generateKptManifests")
		cmd := exec.CommandContext(tCtx, "kpt", "fn", "render", kPath,
			fmt.Sprintf("--output=%v", outputDir))
		cmd.Stderr = out
		if err = util.RunCmd(ctx, cmd); err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}
		kptManifests = append(kptManifests, outputDir)
	}

	// Generate Raw Manifests
	sourceManifests, err := resolveRemoteAndLocal(g.config.RawK8s, g.workingDir)
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not expand the glob raw manifests: %w", err))
		return nil, err
	}

	hydratedManifests := append(sourceManifests, kptManifests...)
	for _, nkPath := range hydratedManifests {
		if !kubernetes.HasKubernetesFileExtension(nkPath) {
			if !stringslice.Contains(g.config.RawK8s, nkPath) {
				log.Entry(ctx).Infof("refusing to deploy/delete non {json, yaml} file %s", nkPath)
				log.Entry(ctx).Info("If you still wish to deploy this file, please specify it directly, outside a glob pattern.")
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

	for _, base := range constants.KustomizeFilePaths {
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

// walkManifests finds out all the manifests from the `.manifests.generate`, so they can be registered in the file watcher.
// Note: the logic about manifest dependencies shall separate from the "Generate" function, which requires "context" and
// only be called when a renderig action is needed (normally happens after the file watcher registration).
func (g Generator) walkManifests() ([]string, error) {
	var dependencyPaths []string
	// Generate kustomize Manifests
	kustomizePaths, err := resolveRemoteAndLocal(g.config.Kustomize, g.workingDir)
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not expand the glob kustomize manifests: %w", err))
		return nil, err
	}
	dependencyPaths = append(dependencyPaths, kustomizePaths...)

	// Generate in-place hydrated kpt Manifests
	kptPaths, err := resolveRemoteAndLocal(g.config.Kpt, g.workingDir)
	kptPaths, err = util.ExpandPathsGlob(g.workingDir, kptPaths)
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not expand the glob kpt manifests: %w", err))
		return nil, err
	}
	dependencyPaths = append(dependencyPaths, kptPaths...)

	// Generate Raw Manifests
	sourceManifests, err := resolveRemoteAndLocal(g.config.RawK8s, g.workingDir)
	sourceManifests, err = util.ExpandPathsGlob(g.workingDir, sourceManifests)
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not expand the glob raw manifests: %w", err))
		return nil, err
	}
	dependencyPaths = append(dependencyPaths, sourceManifests...)
	return dependencyPaths, nil
}

func (g Generator) ManifestDeps() ([]string, error) {
	deps := stringset.New()

	dependencyPaths, err := g.walkManifests()
	if err != nil {
		return nil, err
	}
	for _, path := range dependencyPaths {
		err := filepath.Walk(path,
			func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				fname := filepath.Base(p)
				if strings.HasSuffix(fname, ".yaml") || strings.HasSuffix(fname, ".yml") || fname == kptfile.KptFileName {
					deps.Insert(p)
				}
				return nil
			})
		if err != nil {
			return nil, err
		}
	}
	return deps.ToList(), nil
}
