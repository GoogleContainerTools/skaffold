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

package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	deploy "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	kptHydrated       = ".kpt-hydrated"
	inventoryTemplate = "inventory-template.yaml"
)

// KptDeployer deploys workflows with kpt CLI
type KptDeployer struct {
	*latest.KptDeploy

	insecureRegistries map[string]bool
	labels             map[string]string
	globalConfig       string
}

func NewKptDeployer(runCtx *runcontext.RunContext, labels map[string]string) *KptDeployer {
	return &KptDeployer{
		KptDeploy:          runCtx.Pipeline().Deploy.KptDeploy,
		insecureRegistries: runCtx.GetInsecureRegistries(),
		labels:             labels,
		globalConfig:       runCtx.GlobalConfig(),
	}
}

func (k *KptDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]string, error) {
	return nil, nil
}

// Dependencies returns a list of files that the deployer depends on. This does NOT include applyDir.
// In dev mode, a redeploy will be triggered if one of these files is updated.
func (k *KptDeployer) Dependencies() ([]string, error) {
	deps := newStringSet()
	if len(k.Fn.FnPath) > 0 {
		deps.insert(k.Fn.FnPath)
	}

	configDeps, err := getResources(k.Dir)
	if err != nil {
		return nil, fmt.Errorf("finding dependencies in %s: %w", k.Dir, err)
	}

	deps.insert(configDeps...)

	return deps.toList(), nil
}

// Cleanup deletes what was deployed by calling `kpt live destroy`.
func (k *KptDeployer) Cleanup(ctx context.Context, _ io.Writer) error {
	applyDir, err := k.getApplyDir(ctx)
	if err != nil {
		return fmt.Errorf("getting applyDir: %w", err)
	}

	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(applyDir, []string{"live", "destroy"}, nil, nil)...)
	out, err := util.RunCmdOut(cmd)
	if err != nil {
		// Kpt errors are written in STDOUT and surrounded by `\n`.
		return fmt.Errorf("kpt live destroy: %s", strings.Trim(string(out), "\n"))
	}

	return nil
}

func (k *KptDeployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, _ bool, filepath string) error {
	manifests, err := k.renderManifests(ctx, out, builds)
	if err != nil {
		return err
	}

	return outputRenderedManifests(manifests.String(), filepath, out)
}

func (k *KptDeployer) renderManifests(ctx context.Context, _ io.Writer, builds []build.Artifact) (deploy.ManifestList, error) {
	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(k.globalConfig)
	if err != nil {
		return nil, fmt.Errorf("retrieving debug helpers registry: %w", err)
	}

	manifests, err := k.kptFnRun(ctx)
	if err != nil {
		return nil, fmt.Errorf("running kpt functions: %w", err)
	}

	if len(manifests) == 0 {
		return nil, nil
	}

	manifests, err = manifests.ReplaceImages(builds)
	if err != nil {
		return nil, fmt.Errorf("replacing images in manifests: %w", err)
	}

	for _, transform := range manifestTransforms {
		manifests, err = transform(manifests, builds, Registries{k.insecureRegistries, debugHelpersRegistry})
		if err != nil {
			return nil, fmt.Errorf("unable to transform manifests: %w", err)
		}
	}

	return manifests.SetLabels(k.labels)
}

// kptFnRun does a dry run with the specified kpt functions (fn-path XOR image) against dir.
// If neither fn-path nor image are specified, functions will attempt to be discovered in dir.
// An error is returned when both fn-path and image are specified.
func (k *KptDeployer) kptFnRun(ctx context.Context) (deploy.ManifestList, error) {
	var manifests deploy.ManifestList

	// --dry-run sets the pipeline's output to STDOUT, otherwise output is set to sinkDir.
	// For now, k.Dir will be treated as sinkDir (and sourceDir).
	flags := []string{"--dry-run"}
	specifiedFnPath := false

	if len(k.Fn.FnPath) > 0 {
		flags = append(flags, "--fn-path", k.Fn.FnPath)
		specifiedFnPath = true
	}
	if len(k.Fn.Image) > 0 {
		if specifiedFnPath {
			return nil, errors.New("cannot specify both fn-path and image")
		}

		flags = append(flags, "--image", k.Fn.Image)
	}

	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(k.Dir, []string{"fn", "run"}, flags, nil)...)
	out, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil, err
	}

	if len(out) > 0 {
		manifests.Append(out)
	}

	return manifests, nil
}

// getApplyDir returns the path to applyDir if specified by the user. Otherwise, getApplyDir
// creates a hidden directory named .kpt-hydrated in place of applyDir.
func (k *KptDeployer) getApplyDir(ctx context.Context) (string, error) {
	if k.ApplyDir != "" {
		if _, err := os.Stat(k.ApplyDir); os.IsNotExist(err) {
			return "", err
		}
		return k.ApplyDir, nil
	}

	// 0755 is a permission setting where the owner can read, write, and execute.
	// Others can read and execute but not modify the file.
	if err := os.MkdirAll(kptHydrated, 0755); err != nil {
		return "", fmt.Errorf("applyDir was unspecified. creating applyDir: %w", err)
	}

	if _, err := os.Stat(filepath.Join(kptHydrated, inventoryTemplate)); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(kptHydrated, []string{"live", "init"}, nil, nil)...)
		if _, err := util.RunCmdOut(cmd); err != nil {
			return "", err
		}
	}

	return kptHydrated, nil
}

// kptCommandArgs returns a list of additional arguments for the kpt command.
func kptCommandArgs(dir string, commands, flags, globalFlags []string) []string {
	var args []string

	for _, v := range commands {
		parts := strings.Split(v, " ")
		args = append(args, parts...)
	}

	if len(dir) > 0 {
		args = append(args, dir)
	}

	for _, v := range flags {
		parts := strings.Split(v, " ")
		args = append(args, parts...)
	}

	for _, v := range globalFlags {
		parts := strings.Split(v, " ")
		args = append(args, parts...)
	}

	return args
}

// getResources returns a list of all file names in `root` that end in .yaml or .yml
// and all local kustomization dependencies under root.
func getResources(root string) ([]string, error) {
	var files []string

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, err
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, _ error) error {
		// Using regex match is not entirely accurate in deciding whether something is a resource or not.
		// Kpt should provide better functionality for determining whether files are resources.
		isResource, err := regexp.MatchString(`\.ya?ml$`, filepath.Base(path))
		if err != nil {
			return fmt.Errorf("matching %s with regex: %w", filepath.Base(path), err)
		}

		if info.IsDir() {
			depsForKustomization, err := dependenciesForKustomization(path)
			if err != nil {
				return err
			}

			files = append(files, depsForKustomization...)
		} else if isResource {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
