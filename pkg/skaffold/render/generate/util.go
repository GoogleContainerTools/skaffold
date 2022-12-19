/*
Copyright 2020 The Skaffold Authors

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
	"os"
	"os/exec"
	"path/filepath"

	yamlv3 "gopkg.in/yaml.v3"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/warnings"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

// for testing
var (
	KubectlVersionCheck  = kubectlVersion
	KustomizeBinaryCheck = kustomizeBinary
)

// kustomization is the content of a kustomization.yaml file.
type kustomization struct {
	Components            []string              `yaml:"components"`
	Bases                 []string              `yaml:"bases"`
	Resources             []string              `yaml:"resources"`
	Patches               []patchWrapper        `yaml:"patches"`
	PatchesStrategicMerge []strategicMergePatch `yaml:"patchesStrategicMerge"`
	CRDs                  []string              `yaml:"crds"`
	PatchesJSON6902       []patchJSON6902       `yaml:"patchesJson6902"`
	ConfigMapGenerator    []configMapGenerator  `yaml:"configMapGenerator"`
	SecretGenerator       []secretGenerator     `yaml:"secretGenerator"`
}

type patchPath struct {
	Path  string `yaml:"path"`
	Patch string `yaml:"patch"`
}

type patchWrapper struct {
	*patchPath
}

type strategicMergePatch struct {
	Path  string
	Patch string
}

type patchJSON6902 struct {
	Path string `yaml:"path"`
}

type configMapGenerator struct {
	Files []string `yaml:"files"`
	Env   string   `yaml:"env"`
	Envs  []string `yaml:"envs"`
}

type secretGenerator struct {
	Files []string `yaml:"files"`
	Env   string   `yaml:"env"`
	Envs  []string `yaml:"envs"`
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

	content := kustomization{}
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
		if patch.Path != "" {
			deps = append(deps, filepath.Join(dir, patch.Path))
		}
	}

	deps = append(deps, util.AbsolutePaths(dir, content.CRDs)...)

	for _, patch := range content.Patches {
		if patch.Path != "" {
			deps = append(deps, filepath.Join(dir, patch.Path))
		}
	}

	for _, jsonPatch := range content.PatchesJSON6902 {
		if jsonPatch.Path != "" {
			deps = append(deps, filepath.Join(dir, jsonPatch.Path))
		}
	}

	for _, generator := range content.ConfigMapGenerator {
		deps = append(deps, util.AbsolutePaths(dir, generator.Files)...)
		envs := generator.Envs
		if generator.Env != "" {
			envs = append(envs, generator.Env)
		}
		deps = append(deps, util.AbsolutePaths(dir, envs)...)
	}

	for _, generator := range content.SecretGenerator {
		deps = append(deps, util.AbsolutePaths(dir, generator.Files)...)
		envs := generator.Envs
		if generator.Env != "" {
			envs = append(envs, generator.Env)
		}
		deps = append(deps, util.AbsolutePaths(dir, envs)...)
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

type kCfg struct {
	context   string
	config    string
	namespace string
}

func NewKCfg(context, config, namespace string) kCfg {
	return kCfg{
		context:   context,
		config:    config,
		namespace: namespace,
	}
}

func (k kCfg) GetKubeContext() string {
	return k.context
}
func (k kCfg) GetKubeConfig() string {
	return k.config
}
func (k kCfg) GetKubeNamespace() string {
	return k.namespace
}

func kustomizeBinary() bool {
	_, err := exec.LookPath("kustomize")
	return err == nil
}

// Check that kubectl version is valid to use kubectl kustomize
func kubectlVersion(kubectl *kubectl.CLI) bool {
	gt, err := kubectl.CompareVersionTo(context.Background(), 1, 14)
	if err != nil {
		return false
	}

	return gt == 1
}

// Values of `patchesStrategicMerge` can be either:
// + a file path, referenced as a plain string
// + an inline patch referenced as a string literal
func (p *strategicMergePatch) UnmarshalYAML(node *yamlv3.Node) error {
	if node.Style == 0 || node.Style == yamlv3.DoubleQuotedStyle || node.Style == yamlv3.SingleQuotedStyle {
		p.Path = node.Value
	} else {
		p.Patch = node.Value
	}

	return nil
}

// UnmarshalYAML implements JSON unmarshalling by reading an inline yaml fragment.
func (p *patchWrapper) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	pp := &patchPath{}
	if err := unmarshal(&pp); err != nil {
		var oldPathString string
		if err := unmarshal(&oldPathString); err != nil {
			return err
		}
		warnings.Printf("list of file paths deprecated: see https://github.com/kubernetes-sigs/kustomize/blob/master/docs/plugins/builtins.md#patchtransformer")
		pp.Path = oldPathString
	}
	p.patchPath = pp
	return nil
}
