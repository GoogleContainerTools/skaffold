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
	"bytes"
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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

var (
	inventoryTemplate = "inventory-template.yaml"
	kptHydrated       = ".kpt-hydrated"
	pipeline          = ".pipeline"
)

// KptDeployer deploys workflows with kpt CLI
type KptDeployer struct {
	*latest.KptDeploy

	insecureRegistries map[string]bool
	labels             map[string]string
	globalConfig       string
}

func NewKptDeployer(ctx Config, labels map[string]string) *KptDeployer {
	return &KptDeployer{
		KptDeploy:          ctx.Pipeline().Deploy.KptDeploy,
		insecureRegistries: ctx.GetInsecureRegistries(),
		labels:             labels,
		globalConfig:       ctx.GlobalConfig(),
	}
}

// Deploy hydrates the manifests using kustomizations and kpt functions as described in the render method,
// outputs them to the applyDir, and runs `kpt live apply` against applyDir to create resources in the cluster.
// `kpt live apply` supports automated pruning declaratively via resources in the applyDir.
func (k *KptDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]string, error) {
	manifests, err := k.renderManifests(ctx, out, builds)
	if err != nil {
		return nil, err
	}

	if len(manifests) == 0 {
		return nil, nil
	}

	namespaces, err := manifests.CollectNamespaces()
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not fetch deployed resource namespace. "+
			"This might cause port-forward and deploy health-check to fail: %w", err))
	}

	applyDir, err := k.getApplyDir(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting applyDir: %w", err)
	}

	outputRenderedManifests(manifests.String(), filepath.Join(applyDir, "resources.yaml"), out)

	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(applyDir, []string{"live", "apply"}, k.getKptLiveApplyArgs(), nil)...)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		return nil, err
	}

	return namespaces, nil
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

	// Kpt deployer assumes that the kustomization configuration to build lives directly under k.Dir.
	kustomizeDeps, err := dependenciesForKustomization(k.Dir)
	if err != nil {
		return nil, fmt.Errorf("finding kustomization directly under %s: %w", k.Dir, err)
	}

	deps.insert(kustomizeDeps...)

	return deps.toList(), nil
}

// Cleanup deletes what was deployed by calling `kpt live destroy`.
func (k *KptDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	applyDir, err := k.getApplyDir(ctx)
	if err != nil {
		return fmt.Errorf("getting applyDir: %w", err)
	}

	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(applyDir, []string{"live", "destroy"}, nil, nil)...)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		return err
	}

	return nil
}

// Render hydrates manifests using both kustomization and kpt functions.
func (k *KptDeployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, _ bool, filepath string) error {
	manifests, err := k.renderManifests(ctx, out, builds)
	if err != nil {
		return err
	}

	return outputRenderedManifests(manifests.String(), filepath, out)
}

// renderManifests handles a majority of the hydration process for manifests.
// This involves reading configs from a source directory, running kustomize build, running kpt pipelines,
// adding image digests, and adding run-id labels.
func (k *KptDeployer) renderManifests(ctx context.Context, _ io.Writer, builds []build.Artifact) (deploy.ManifestList, error) {
	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(k.globalConfig)
	if err != nil {
		return nil, fmt.Errorf("retrieving debug helpers registry: %w", err)
	}

	// .pipeline is a temp dir used to store output between steps of the desired workflow
	// This can be removed once kpt can fully support the desired workflow independently.
	if err := os.RemoveAll(filepath.Join(pipeline, k.Dir)); err != nil {
		return nil, fmt.Errorf("deleting temporary directory %s: %w", filepath.Join(pipeline, k.Dir), err)
	}
	// 0755 is a permission setting where the owner can read, write, and execute.
	// Others can read and execute but not modify the directory.
	if err := os.MkdirAll(filepath.Join(pipeline, k.Dir), 0755); err != nil {
		return nil, fmt.Errorf("creating temporary directory %s: %w", filepath.Join(pipeline, k.Dir), err)
	}

	if err := k.readConfigs(ctx); err != nil {
		return nil, fmt.Errorf("reading config manifests: %w", err)
	}

	if err := k.kustomizeBuild(ctx); err != nil {
		return nil, fmt.Errorf("kustomize build: %w", err)
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

// readConfigs uses `kpt fn source` to read config manifests from k.Dir
// and uses `kpt fn sink` to output those manifests to .pipeline.
func (k *KptDeployer) readConfigs(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(k.Dir, []string{"fn", "source"}, nil, nil)...)
	b, err := util.RunCmdOut(cmd)
	if err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "kpt", kptCommandArgs(filepath.Join(pipeline, k.Dir), []string{"fn", "sink"}, nil, nil)...)
	cmd.Stdin = bytes.NewBuffer(b)
	if _, err := util.RunCmdOut(cmd); err != nil {
		return err
	}

	return nil
}

// kustomizeBuild runs `kustomize build` if a kustomization config exists and outputs to .pipeline.
func (k *KptDeployer) kustomizeBuild(ctx context.Context) error {
	if _, err := findKustomizationConfig(k.Dir); err != nil {
		// No kustomization config was found directly under k.Dir, so there is no need to continue.
		return nil
	}

	cmd := exec.CommandContext(ctx, "kustomize", buildCommandArgs([]string{"-o", filepath.Join(pipeline, k.Dir)}, k.Dir)...)
	if _, err := util.RunCmdOut(cmd); err != nil {
		return err
	}

	deps, err := dependenciesForKustomization(k.Dir)
	if err != nil {
		return fmt.Errorf("finding kustomization dependencies: %w", err)
	}

	// Kustomize build outputs hydrated configs to .pipeline, so the dry configs must be removed.
	for _, v := range deps {
		if err := os.RemoveAll(filepath.Join(pipeline, v)); err != nil {
			return err
		}
	}

	return nil
}

// kptFnRun does a dry run with the specified kpt functions (fn-path XOR image) against .pipeline.
// If neither fn-path nor image are specified, functions will attempt to be discovered in .pipeline.
// An error occurs if both fn-path and image are specified.
func (k *KptDeployer) kptFnRun(ctx context.Context) (deploy.ManifestList, error) {
	var manifests deploy.ManifestList

	flags, err := k.getKptFnRunArgs()
	if err != nil {
		return nil, fmt.Errorf("getting kpt fn run args: %w", err)
	}

	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(pipeline, []string{"fn", "run"}, flags, nil)...)
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
	// Others can read and execute but not modify the directory.
	if err := os.MkdirAll(kptHydrated, 0755); err != nil {
		return "", fmt.Errorf("applyDir was unspecified. creating applyDir: %w", err)
	}

	if _, err := os.Stat(filepath.Join(kptHydrated, inventoryTemplate)); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(kptHydrated, []string{"live", "init"}, k.getKptLiveInitArgs(), nil)...)
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

// getResources returns a list of all file names in root that end in .yaml or .yml
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

		if !info.IsDir() && isResource {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// getKptFnRunArgs returns a list of arguments that the user specified for the `kpt fn run` command.
func (k *KptDeployer) getKptFnRunArgs() ([]string, error) {
	// --dry-run sets the pipeline's output to STDOUT, otherwise output is set to sinkDir.
	// For now, k.Dir will be treated as sinkDir (and sourceDir).
	flags := []string{"--dry-run"}

	if k.Fn.GlobalScope {
		flags = append(flags, "--global-scope")
	}

	if len(k.Fn.Mount) > 0 {
		flags = append(flags, "--mount", strings.Join(k.Fn.Mount, ","))
	}

	if k.Fn.Network {
		flags = append(flags, "--network")
	}

	if len(k.Fn.NetworkName) > 0 {
		flags = append(flags, "--network-name", k.Fn.NetworkName)
	}

	count := 0

	if len(k.Fn.FnPath) > 0 {
		flags = append(flags, "--fn-path", k.Fn.FnPath)
		count++
	}

	if len(k.Fn.Image) > 0 {
		flags = append(flags, "--image", k.Fn.Image)
		count++
	}

	if count > 1 {
		return nil, errors.New("only one of `fn-path` or `image` may be specified")
	}

	return flags, nil
}

// getKptLiveApplyArgs returns a list of arguments that the user specified for the `kpt live apply` command.
func (k *KptDeployer) getKptLiveApplyArgs() []string {
	var flags []string

	if len(k.Live.Apply.PollPeriod) > 0 {
		flags = append(flags, "--poll-period", k.Live.Apply.PollPeriod)
	}

	if len(k.Live.Apply.PrunePropagationPolicy) > 0 {
		flags = append(flags, "--prune-propagation-policy", k.Live.Apply.PrunePropagationPolicy)
	}

	if len(k.Live.Apply.PruneTimeout) > 0 {
		flags = append(flags, "--prune-timeout", k.Live.Apply.PruneTimeout)
	}

	if len(k.Live.Apply.ReconcileTimeout) > 0 {
		flags = append(flags, "--reconcile-timeout", k.Live.Apply.ReconcileTimeout)
	}

	return flags
}

// getKptLiveInitArgs returns a list of arguments that the user specified for the `kpt live init` command.
func (k *KptDeployer) getKptLiveInitArgs() []string {
	var flags []string

	if len(k.Live.InventoryID) > 0 {
		flags = append(flags, "--inventory-id", k.Live.InventoryID)
	}

	if len(k.Live.InventoryNamespace) > 0 {
		flags = append(flags, "--namespace", k.Live.InventoryNamespace)
	}

	return flags
}
