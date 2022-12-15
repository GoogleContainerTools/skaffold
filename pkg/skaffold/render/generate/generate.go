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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	rErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/kptfile"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringset"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringslice"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

// NewGenerator instantiates a Generator object.
func NewGenerator(workingDir string, config latest.Generate, hydrationDir string) Generator {
	return Generator{
		workingDir:   workingDir,
		config:       config,
		hydrationDir: hydrationDir,
	}
}

// Generator provides the functions for the manifest sources (raw manifests, helm charts, kustomize configs and remote packages).
type Generator struct {
	workingDir   string
	hydrationDir string
	config       latest.Generate
}

func localManifests(paths []string, workdir string) ([]string, error) {
	var localPaths []string
	for _, path := range paths {
		switch {
		case util.IsURL(path):
		case strings.HasPrefix(path, "gs://"):
		default:
			localPaths = append(localPaths, path)
		}
	}
	return util.ExpandPathsGlob(workdir, localPaths)
}

func resolveRemoteAndLocal(paths []string, workdir string) ([]string, error) {
	var localPaths []string
	var gcsManifests []string
	var urlManifests []string
	for _, path := range paths {
		switch {
		case util.IsURL(path):
			urlManifests = append(urlManifests, path)
		case strings.HasPrefix(path, "gs://"):
			gcsManifests = append(gcsManifests, path)
		default:
			// expand paths
			path, err := util.ExpandEnvTemplate(path, nil)
			if err != nil {
				return nil, err
			}
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
	if len(urlManifests) != 0 {
		paths, err := manifest.DownloadFromURL(urlManifests)
		if err != nil {
			return nil, err
		}
		list = append(list, paths...)
	}
	return list, nil
}

// Generate parses the config resources from the paths in .Generate.Manifests. This path can be the path to raw manifest,
// kustomize manifests, helm charts or kpt function configs. All should be file-watched.
func (g Generator) Generate(ctx context.Context, out io.Writer) (manifest.ManifestList, error) {
	var manifests manifest.ManifestList

	// Generate kustomize Manifests
	_, endTrace := instrumentation.StartTrace(ctx, "Render_kustomize")
	if g.config.Kustomize != nil && len(g.config.Kustomize.Paths) != 0 {
		log.Entry(ctx).Infof("rendering using kustomize")
		kustomizeManifests, err := g.generateKustomizeManifests(ctx)
		if err != nil {
			return nil, err
		}
		for _, m := range kustomizeManifests {
			manifests.Append(m)
		}
	}
	endTrace()

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
	if len(kptPathMap) != 0 {
		log.Entry(ctx).Infof("rendering using kpt")
	}
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
		manifestFileContent, err := os.ReadFile(nkPath)
		if err != nil {
			return nil, err
		}
		manifests.Append(manifestFileContent)
	}

	// Generate remote manifests
	for _, m := range g.config.RemoteManifests {
		manifest, err := g.readRemoteManifest(ctx, m)
		if err != nil {
			return nil, err
		}
		manifests.Append(manifest)
	}

	return manifests, nil
}

// readRemoteManifests will try to read manifests from the given kubernetes
// context in the specified namespace and for the specified type
func (g Generator) readRemoteManifest(ctx context.Context, rm latest.RemoteManifest) ([]byte, error) {
	var args []string
	ns := ""
	name := rm.Manifest
	if parts := strings.Split(name, ":"); len(parts) > 1 {
		ns = parts[0]
		name = parts[1]
	}
	args = append(args, name, "-o", "yaml")

	var manifest bytes.Buffer
	err := kubectl.NewCLI(NewKCfg(rm.KubeContext, "", ""), "").RunInNamespace(ctx, nil, &manifest, "get", ns, args...)
	if err != nil {
		return nil, rErrors.ReadRemoteManifestErr(fmt.Errorf("getting remote manifests: %w", err))
	}

	return manifest.Bytes(), nil
}

func (g Generator) generateKustomizeManifests(ctx context.Context) ([][]byte, error) {
	var manifests [][]byte

	_, endTrace := instrumentation.StartTrace(ctx, "Render_expandGlobKustomizeManifests")
	kustomizePaths, err := resolveRemoteAndLocal(g.config.Kustomize.Paths, g.workingDir)
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

	kCLI := kubectl.NewCLI(kCfg{}, "")
	useKubectlKustomize := !KustomizeBinaryCheck() && KubectlVersionCheck(kCLI)

	for kPath := range kustomizePathMap {
		var out []byte
		var err error
		if useKubectlKustomize {
			out, err = kCLI.Kustomize(ctx, kustomizeBuildArgs(g.config.Kustomize.BuildArgs, kPath))
		} else {
			cmd := exec.CommandContext(ctx, "kustomize", append([]string{"build"}, kustomizeBuildArgs(g.config.Kustomize.BuildArgs, kPath)...)...)
			out, err = util.RunCmdOut(ctx, cmd)
		}
		if err != nil {
			return nil, err
		}
		if len(out) == 0 {
			continue
		}
		manifests = append(manifests, out)
	}

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

// kustomizeBuildArgs returns a list of build args to be passed to kustomize build.
func kustomizeBuildArgs(buildArgs []string, kustomizePath string) []string {
	var args []string

	if len(buildArgs) > 0 {
		for _, v := range buildArgs {
			parts := strings.Split(v, " ")
			args = append(args, parts...)
		}
	}

	if len(kustomizePath) > 0 {
		args = append(args, kustomizePath)
	}

	return args
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

// walkLocalManifests finds out all the manifests from the `.manifests.generate`, so they can be registered in the file watcher.
// Note: the logic about manifest dependencies shall separate from the "Generate" function, which requires "context" and
// only be called when a rendering action is needed (normally happens after the file watcher registration).
func (g Generator) walkLocalManifests() ([]string, error) {
	var dependencyPaths []string
	var err error

	// Generate in-place hydrated kpt Manifests
	kptPaths, err := localManifests(g.config.Kpt, g.workingDir)
	if err != nil {
		return nil, err
	}
	dependencyPaths = append(dependencyPaths, kptPaths...)

	// Generate Raw Manifests
	sourceManifests, err := localManifests(g.config.RawK8s, g.workingDir)
	if err != nil {
		return nil, err
	}
	dependencyPaths = append(dependencyPaths, sourceManifests...)
	return dependencyPaths, nil
}

func (g Generator) ManifestDeps() ([]string, error) {
	var deps []string

	dependencyPaths, err := g.walkLocalManifests()
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
					deps = append(deps, p)
				}
				return nil
			})
		if err != nil {
			return nil, err
		}
	}
	// kustomize deps
	if g.config.Kustomize != nil {
		kDeps, err := kustomizeDependencies(g.workingDir, g.config.Kustomize.Paths)
		if err != nil {
			return nil, err
		}
		deps = append(deps, kDeps...)
	}

	return deps, nil
}

func kustomizeDependencies(workdir string, paths []string) ([]string, error) {
	deps := stringset.New()
	for _, kustomizePath := range paths {
		expandedKustomizePath, err := util.ExpandEnvTemplate(kustomizePath, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to parse path %q: %w", kustomizePath, err)
		}

		if !filepath.IsAbs(expandedKustomizePath) {
			expandedKustomizePath = filepath.Join(workdir, expandedKustomizePath)
		}
		depsForKustomization, err := DependenciesForKustomization(expandedKustomizePath)
		if err != nil {
			return nil, sErrors.NewError(err,
				&proto.ActionableErr{
					Message: err.Error(),
					ErrCode: proto.StatusCode_DEPLOY_KUSTOMIZE_USER_ERR,
				})
		}
		deps.Insert(depsForKustomization...)
	}
	return deps.ToList(), nil
}
