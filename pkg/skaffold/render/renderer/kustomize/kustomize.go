/*
Copyright 2022 The Skaffold Authors

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

package kustomize

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kustomize/api/types"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/applysetters"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/validate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	sUtil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringset"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

type Kustomize struct {
	cfg  render.Config
	rCfg latest.RenderConfig

	configName string
	namespace  string
	injectNs   bool

	labels            map[string]string
	manifestOverrides map[string]string

	applySetters       applysetters.ApplySetters
	transformer        transform.Transformer
	validator          validate.Validator
	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func (k Kustomize) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {
	var manifests manifest.ManifestList
	kCLI := kubectl.NewCLI(k.cfg, "")
	useKubectlKustomize := !generate.KustomizeBinaryCheck() && generate.KubectlVersionCheck(kCLI)

	var kustomizePaths []string
	for _, kustomizePath := range k.rCfg.Kustomize.Paths {
		kPath, err := sUtil.ExpandEnvTemplate(kustomizePath, nil)
		if err != nil {
			return manifest.ManifestListByConfig{}, fmt.Errorf("unable to parse path %q: %w", kustomizePath, err)
		}
		kustomizePaths = append(kustomizePaths, kPath)
	}

	for _, kustomizePath := range sUtil.AbsolutePaths(k.cfg.GetWorkingDir(), kustomizePaths) {
		out, err := k.render(ctx, kustomizePath, useKubectlKustomize, kCLI)
		if err != nil {
			return manifest.ManifestListByConfig{}, err
		}
		if len(out) == 0 {
			continue
		}
		manifests.Append(out)
	}

	opts := util.GenerateHydratedManifestsOptions{
		TransformAllowList:         k.transformAllowlist,
		TransformDenylist:          k.transformDenylist,
		EnablePlatformNodeAffinity: k.cfg.EnablePlatformNodeAffinityInRenderedManifests(),
		EnableGKEARMNodeToleration: k.cfg.EnableGKEARMNodeTolerationInRenderedManifests(),
		Offline:                    offline,
		KubeContext:                k.cfg.GetKubeContext(),
		InjectNamespace:            k.injectNs,
	}

	ns := k.namespace
	if k.injectNs {
		ns = k.cfg.GetKubeNamespace()
	}
	manifests, err := util.BaseTransform(ctx, manifests, builds, opts, k.labels, ns)
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	manifestListByConfig := manifest.NewManifestListByConfig()
	manifestListByConfig.Add(k.configName, manifests)

	return manifestListByConfig, nil
}

func (k Kustomize) render(ctx context.Context, kustomizePath string, useKubectlKustomize bool, kCLI *kubectl.CLI) ([]byte, error) {
	var out []byte

	if (len(k.applySetters.Setters) > 0 || !k.transformer.IsEmpty()) && !sUtil.IsURL(kustomizePath) {
		temp, err := os.MkdirTemp("", "*")
		if err != nil {
			return out, err
		}
		fs := newTmpFS(temp)
		defer fs.Cleanup()

		if err := k.mirror(kustomizePath, fs); err == nil {
			kustomizePath = filepath.Join(temp, kustomizePath)
		} else {
			return out, err
		}
	}

	if useKubectlKustomize {
		return kCLI.Kustomize(ctx, kustomizeBuildArgs(k.rCfg.Kustomize.BuildArgs, kustomizePath))
	} else {
		cmd := exec.CommandContext(ctx, "kustomize", append([]string{"build"}, kustomizeBuildArgs(k.rCfg.Kustomize.BuildArgs, kustomizePath)...)...)
		return sUtil.RunCmdOut(ctx, cmd)
	}
}

func getKustomizationFile(kusDir string) (string, error) {
	for _, path := range constants.KustomizeFilePaths {
		kPath := filepath.Join(kusDir, path)
		_, err := os.Stat(kPath)
		if err != nil {
			continue
		}
		return kPath, nil
	}
	return "", fmt.Errorf("cannot locate kustomization file from provided directory: %s", kusDir)
}

func (k Kustomize) mirror(kusDir string, fs TmpFS) error {
	kFile, err := getKustomizationFile(kusDir)
	if err != nil {
		return err
	}

	bytes, err := ioutil.ReadFile(kFile)
	if err != nil {
		return err
	}

	if err := fs.WriteTo(kFile, bytes); err != nil {
		return err
	}

	kustomization := types.Kustomization{}
	if err := yaml.Unmarshal(bytes, &kustomization); err != nil {
		return err
	}
	if err := k.mirrorPatchesStrategicMerge(kusDir, fs, kustomization.PatchesStrategicMerge); err != nil {
		return err
	}
	if err := k.mirrorPatchesJSON6902(kusDir, fs, kustomization.PatchesJson6902); err != nil {
		return err
	}
	if err := k.mirrorPatches(kusDir, fs, kustomization.Patches); err != nil {
		return err
	}
	if err := k.mirrorResources(kusDir, fs, kustomization.Resources); err != nil {
		return err
	}
	if err := k.mirrorCrds(kusDir, fs, kustomization.Crds); err != nil {
		return err
	}
	if err := k.mirrorBases(kusDir, fs, kustomization.Bases); err != nil {
		return err
	}
	if err := k.mirrorConfigurations(kusDir, fs, kustomization.Configurations); err != nil {
		return err
	}
	if err := k.mirrorGenerators(kusDir, fs, kustomization.Generators); err != nil {
		return err
	}
	if err := k.mirrorTransformers(kusDir, fs, kustomization.Transformers); err != nil {
		return err
	}
	if err := k.mirrorValidators(kusDir, fs, kustomization.Validators); err != nil {
		return err
	}
	if err := k.mirrorSecretGenerators(kusDir, fs, kustomization.SecretGenerator); err != nil {
		return err
	}
	if err := k.mirrorConfigMapGenerators(kusDir, fs, kustomization.ConfigMapGenerator); err != nil {
		return err
	}

	return nil
}

func New(cfg render.Config, rCfg latest.RenderConfig, labels map[string]string, configName string, ns string, manifestOverrides map[string]string, injectNs bool) (Kustomize, error) {
	transformAllowlist, transformDenylist, err := util.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return Kustomize{}, err
	}

	var validator validate.Validator
	if rCfg.Validate != nil {
		validator, err = validate.NewValidator(*rCfg.Validate)
		if err != nil {
			return Kustomize{}, err
		}
	}

	var transformer transform.Transformer
	if rCfg.Transform != nil {
		transformer, err = transform.NewTransformer(*rCfg.Transform)
		if err != nil {
			return Kustomize{}, err
		}
	}

	var ass applysetters.ApplySetters
	if len(manifestOverrides) > 0 {
		for k, v := range manifestOverrides {
			ass.Setters = append(ass.Setters, applysetters.Setter{Name: k, Value: v})
		}
	}

	return Kustomize{
		cfg:               cfg,
		configName:        configName,
		namespace:         ns,
		injectNs:          injectNs,
		labels:            labels,
		rCfg:              rCfg,
		manifestOverrides: manifestOverrides,
		validator:         validator,
		transformer:       transformer,
		applySetters:      ass,

		transformAllowlist: transformAllowlist,
		transformDenylist:  transformDenylist,
	}, nil
}

func (k Kustomize) ManifestDeps() ([]string, error) {
	return kustomizeDependencies(k.cfg.GetWorkingDir(), k.rCfg.Kustomize.Paths)
}

func (k Kustomize) mirrorPatchesStrategicMerge(kusDir string, fs TmpFS, merges []types.PatchStrategicMerge) error {
	for _, p := range merges {
		if err := k.mirrorFile(kusDir, fs, string(p)); err != nil {
			return err
		}
	}
	return nil
}

func (k Kustomize) mirrorResources(kusDir string, fs TmpFS, resources []string) error {
	for _, r := range resources {
		// note that r is relative to kustomization file not working dir here
		rPath := filepath.Join(kusDir, r)
		stat, err := os.Stat(rPath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			if err := k.mirror(rPath, fs); err != nil {
				return err
			}
		} else {
			if err := k.mirrorFile(kusDir, fs, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Kustomize) mirrorFile(kusDir string, fs TmpFS, path string) error {
	if sUtil.IsURL(path) {
		return nil
	}
	pFile := filepath.Join(kusDir, path)
	bytes, err := ioutil.ReadFile(pFile)
	if err != nil {
		return err
	}
	if err := fs.WriteTo(pFile, bytes); err != nil {
		return err
	}
	fsPath, err := fs.GetPath(pFile)

	if err != nil {
		return err
	}

	err = k.transformer.TransformPath(fsPath)
	if err != nil {
		return err
	}

	err = k.applySetters.ApplyPath(fsPath)
	if err != nil {
		return fmt.Errorf("failed to apply setter to file %s, err: %v", pFile, err)
	}
	return nil
}

func (k Kustomize) mirrorFiles(kusDir string, fs TmpFS, paths []string) error {
	for _, path := range paths {
		err := k.mirrorFile(kusDir, fs, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Kustomize) mirrorBases(kusDir string, fs TmpFS, bases []string) error {
	for _, b := range bases {
		if err := k.mirror(filepath.Join(kusDir, b), fs); err != nil {
			return err
		}
	}
	return nil
}

func (k Kustomize) mirrorCrds(kusDir string, fs TmpFS, crds []string) error {
	return k.mirrorFiles(kusDir, fs, crds)
}

func (k Kustomize) mirrorConfigurations(kusDir string, fs TmpFS, configurations []string) error {
	return k.mirrorFiles(kusDir, fs, configurations)
}

func (k Kustomize) mirrorGenerators(kusDir string, fs TmpFS, generators []string) error {
	return k.mirrorFiles(kusDir, fs, generators)
}

func (k Kustomize) mirrorTransformers(kusDir string, fs TmpFS, transformers []string) error {
	return k.mirrorFiles(kusDir, fs, transformers)
}

func (k Kustomize) mirrorValidators(kusDir string, fs TmpFS, validators []string) error {
	return k.mirrorFiles(kusDir, fs, validators)
}

func (k Kustomize) mirrorPatches(kusDir string, fs TmpFS, patches []types.Patch) error {
	for _, patch := range patches {
		if err := k.mirrorFile(kusDir, fs, patch.Path); err != nil {
			return err
		}
	}
	return nil
}

func (k Kustomize) mirrorPatchesJSON6902(kusDir string, fs TmpFS, patches []types.Patch) error {
	return k.mirrorPatches(kusDir, fs, patches)
}

func (k Kustomize) mirrorSecretGenerators(kusDir string, fs TmpFS, args []types.SecretArgs) error {
	for _, arg := range args {
		if arg.FileSources != nil {
			for _, f := range arg.FileSources {
				if err := k.mirrorFile(kusDir, fs, f); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (k Kustomize) mirrorConfigMapGenerators(kusDir string, fs TmpFS, args []types.ConfigMapArgs) error {
	for _, arg := range args {
		if arg.FileSources != nil {
			for _, f := range arg.FileSources {
				if err := k.mirrorFile(kusDir, fs, f); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func kustomizeDependencies(workdir string, paths []string) ([]string, error) {
	deps := stringset.New()
	for _, kustomizePath := range paths {
		expandedKustomizePath, err := sUtil.ExpandEnvTemplate(kustomizePath, nil)
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

// DependenciesForKustomization finds common kustomize artifacts relative to the
// provided working dir, and collects them into a list of files to be passed
// to the file watcher.
func DependenciesForKustomization(dir string) ([]string, error) {
	var deps []string

	path, err := FindKustomizationConfig(dir)
	if err != nil {
		// No kustomization config found so assume it's remote and stop traversing
		return deps, nil
	}

	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := types.Kustomization{}
	if err := yaml.Unmarshal(buf, &content); err != nil {
		return nil, fmt.Errorf("kustomization parse error in %v: %w", path, err)
	}

	deps = append(deps, path)

	candidates := append(content.Bases, content.Resources...)
	candidates = append(candidates, content.Components...)

	for _, candidate := range candidates {
		// If the file doesn't exist locally, we can assume it's a remote file and
		// skip it, since we can't monitor remote files. Kustomize itself will
		// handle invalid/missing files.
		local, mode := pathExistsLocally(candidate, dir)
		if !local {
			continue
		}

		if mode.IsDir() {
			candidateDeps, err := DependenciesForKustomization(filepath.Join(dir, candidate))
			if err != nil {
				return nil, err
			}
			deps = append(deps, candidateDeps...)
		} else {
			deps = append(deps, filepath.Join(dir, candidate))
		}
	}

	for _, patch := range content.PatchesStrategicMerge {
		deps = append(deps, filepath.Join(dir, string(patch)))
	}

	deps = append(deps, sUtil.AbsolutePaths(dir, content.Crds)...)

	for _, patch := range content.Patches {
		if patch.Path != "" {
			local, _ := pathExistsLocally(patch.Path, dir)
			if !local {
				continue
			}
			deps = append(deps, filepath.Join(dir, patch.Path))
		}
	}

	for _, jsonPatch := range content.PatchesJson6902 {
		if jsonPatch.Path != "" {
			local, _ := pathExistsLocally(jsonPatch.Path, dir)
			if !local {
				continue
			}
			deps = append(deps, filepath.Join(dir, jsonPatch.Path))
		}
	}

	for _, generator := range content.ConfigMapGenerator {
		deps = append(deps, sUtil.AbsolutePaths(dir, generator.FileSources)...)
		envs := generator.EnvSources
		if generator.EnvSource != "" {
			envs = append(envs, generator.EnvSource)
		}
		deps = append(deps, sUtil.AbsolutePaths(dir, envs)...)
	}

	for _, generator := range content.SecretGenerator {
		deps = append(deps, sUtil.AbsolutePaths(dir, generator.FileSources)...)
		envs := generator.EnvSources
		if generator.EnvSource != "" {
			envs = append(envs, generator.EnvSource)
		}
		deps = append(deps, sUtil.AbsolutePaths(dir, envs)...)
	}

	return deps, nil
}

// FindKustomizationConfig finds the kustomization config relative to the provided dir.
// A Kustomization config must be at the root of the directory. Kustomize will
// error if more than one of these files exists so order doesn't matter.
func FindKustomizationConfig(dir string) (string, error) {
	for _, candidate := range constants.KustomizeFilePaths {
		if local, _ := pathExistsLocally(candidate, dir); local {
			return filepath.Join(dir, candidate), nil
		}
	}
	return "", fmt.Errorf("no Kustomization configuration found in directory: %s", dir)
}

func pathExistsLocally(filename string, workingDir string) (bool, os.FileMode) {
	path := filename
	if !filepath.IsAbs(filename) {
		path = filepath.Join(workingDir, filename)
	}
	if f, err := os.Stat(path); err == nil {
		return true, f.Mode()
	}
	return false, 0
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
