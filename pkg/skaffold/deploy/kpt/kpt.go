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

package kpt

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

	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	k8syaml "sigs.k8s.io/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	inventoryTemplate = "inventory-template.yaml"
	kptHydrated       = ".kpt-hydrated"
	tmpKustomizeDir   = ".kustomize"
	kptFnAnnotation   = "config.kubernetes.io/function"
	kptFnLocalConfig  = "config.kubernetes.io/local-config"

	kptDownloadLink = "https://googlecontainertools.github.io/kpt/installation/"
	kptMinVersion   = "0.34.0"

	kustomizeDownloadLink  = "https://kubernetes-sigs.github.io/kustomize/installation/"
	kustomizeMinVersion    = "v3.2.3"
	kustomizeVersionRegexP = `{Version:(kustomize/)?(\S+) GitCommit:\S+ BuildDate:\S+ GoOs:\S+ GoArch:\S+}`
)

// Deployer deploys workflows with kpt CLI
type Deployer struct {
	*latest.KptDeploy

	insecureRegistries map[string]bool
	labels             map[string]string
	globalConfig       string
}

// NewDeployer generates a new Deployer object contains the kptDeploy schema.
func NewDeployer(cfg types.Config, labels map[string]string, d *latest.KptDeploy) *Deployer {
	return &Deployer{
		KptDeploy:          d,
		insecureRegistries: cfg.GetInsecureRegistries(),
		labels:             labels,
		globalConfig:       cfg.GlobalConfig(),
	}
}

var sanityCheck = versionCheck

// versionCheck checks if the kpt and kustomize versions meet the minimum requirements.
func versionCheck(dir string, stdout io.Writer) error {
	kptCmd := exec.Command("kpt", "version")
	out, err := util.RunCmdOut(kptCmd)
	if err != nil {
		return fmt.Errorf("kpt is not installed yet\nSee kpt installation: %v",
			kptDownloadLink)
	}
	version := strings.TrimSuffix(string(out), "\n")
	// kpt follows semver but does not have "v" prefix.
	if !semver.IsValid("v" + version) {
		return fmt.Errorf("unknown kpt version %v\nPlease upgrade your "+
			"local kpt CLI to a version >= %v\nSee kpt installation: %v",
			string(out), kptMinVersion, kptDownloadLink)
	}
	if semver.Compare("v"+version, "v"+kptMinVersion) < 0 {
		return fmt.Errorf("you are using kpt %q\nPlease update your kpt version to"+
			" >= %v\nSee kpt installation: %v", version[0], kptMinVersion, kptDownloadLink)
	}

	// Users can choose not to use kustomize in kpt deployer mode. We only check the kustomize
	// version when kustomization.yaml config is directed under .deploy.kpt.dir path.
	_, err = kustomize.FindKustomizationConfig(dir)
	if err == nil {
		kustomizeCmd := exec.Command("kustomize", "version")
		out, err := util.RunCmdOut(kustomizeCmd)
		if err != nil {
			return fmt.Errorf("kustomize is not installed yet\nSee kpt installation: %v",
				kustomizeDownloadLink)
		}
		versionInfo := strings.TrimSuffix(string(out), "\n")
		// Kustomize version information is in the form of
		// {Version:$VERSION GitCommit:$COMMIT BuildDate:1970-01-01T00:00:00Z GoOs:darwin GoArch:amd64}
		re := regexp.MustCompile(kustomizeVersionRegexP)
		match := re.FindStringSubmatch(versionInfo)
		if len(match) != 3 {
			color.Yellow.Fprintf(stdout, "unable to determine kustomize version from %q\n"+
				"You can download the official kustomize (recommended >= %v) from %v\n",
				string(out), kustomizeMinVersion, kustomizeDownloadLink)
		} else if !semver.IsValid(match[2]) || semver.Compare(match[2], kustomizeMinVersion) < 0 {
			color.Yellow.Fprintf(stdout, "you are using kustomize version %q "+
				"(recommended >= %v). You can download the official kustomize from %v\n",
				match[2], kustomizeMinVersion, kustomizeDownloadLink)
		}
	}
	return nil
}

// Deploy hydrates the manifests using kustomizations and kpt functions as described in the render method,
// outputs them to the applyDir, and runs `kpt live apply` against applyDir to create resources in the cluster.
// `kpt live apply` supports automated pruning declaratively via resources in the applyDir.
func (k *Deployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]string, error) {
	if err := sanityCheck(k.Dir, out); err != nil {
		return nil, err
	}
	flags, err := k.getKptFnRunArgs()
	if err != nil {
		return []string{}, err
	}
	manifests, err := k.renderManifests(ctx, out, builds, flags)
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

	manifest.Write(manifests.String(), filepath.Join(applyDir, "resources.yaml"), out)
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
func (k *Deployer) Dependencies() ([]string, error) {
	deps := util.NewStringSet()

	// Add the app configuration manifests. It may already include kpt functions and kustomize
	// config files.
	configDeps, err := getResources(k.Dir)
	if err != nil {
		return nil, fmt.Errorf("finding dependencies in %s: %w", k.Dir, err)
	}
	deps.Insert(configDeps...)

	// Add the kustomize resources which lives directly under k.Dir.
	kustomizeDeps, err := kustomize.DependenciesForKustomization(k.Dir)
	if err != nil {
		return nil, fmt.Errorf("finding kustomization directly under %s: %w", k.Dir, err)
	}
	deps.Insert(kustomizeDeps...)

	// Add the kpt function resources when they are outside of the k.Dir directory.
	if len(k.Fn.FnPath) > 0 {
		if rel, err := filepath.Rel(k.Dir, k.Fn.FnPath); err != nil {
			return nil, fmt.Errorf("finding relative path from "+
				".deploy.kpt.fn.fnPath %v to deploy.kpt.Dir %v: %w", k.Fn.FnPath, k.Dir, err)
		} else if strings.HasPrefix(rel, "..") {
			// kpt.FnDir is outside the config .Dir.
			fnDeps, err := getResources(k.Fn.FnPath)
			if err != nil {
				return nil, fmt.Errorf("finding kpt function outside %s: %w", k.Dir, err)
			}
			deps.Insert(fnDeps...)
		}
	}

	return deps.ToList(), nil
}

// Cleanup deletes what was deployed by calling `kpt live destroy`.
func (k *Deployer) Cleanup(ctx context.Context, out io.Writer) error {
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
func (k *Deployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, _ bool, filepath string) error {
	if err := sanityCheck(k.Dir, out); err != nil {
		return err
	}
	flags, err := k.getKptFnRunArgs()
	if err != nil {
		return err
	}

	manifests, err := k.renderManifests(ctx, out, builds, flags)
	if err != nil {
		return err
	}

	return manifest.Write(manifests.String(), filepath, out)
}

// renderManifests handles a majority of the hydration process for manifests.
// This involves reading configs from a source directory, running kustomize build, running kpt pipelines,
// adding image digests, and adding run-id labels.
func (k *Deployer) renderManifests(ctx context.Context, _ io.Writer, builds []build.Artifact,
	flags []string) (manifest.ManifestList, error) {
	var err error
	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(k.globalConfig)
	if err != nil {
		return nil, fmt.Errorf("retrieving debug helpers registry: %w", err)
	}

	var buf []byte
	// Read the manifests under k.Dir as "source".
	cmd := exec.CommandContext(
		ctx, "kpt", kptCommandArgs(k.Dir, []string{"fn", "source"},
			nil, nil)...)
	buf, err = util.RunCmdOut(cmd)
	if err != nil {
		return nil, fmt.Errorf("reading config manifests: %w", err)
	}

	// A workaround for issue https://github.com/GoogleContainerTools/kpt/issues/1149
	// Problem: fn-path cannot be recognized in kpt pipeline mode, and it results in that
	// the kpt functions in are ignored.
	// Solution: pull kpt functions specifically from the kpt source inputs (getKptFunc) and
	// adds it back to the pipeline after kustomize build finishes (append kptFn).
	var kptFnBuf []byte
	if len(k.Fn.FnPath) > 0 {
		cmd = exec.CommandContext(
			ctx, "kpt", kptCommandArgs(k.Fn.FnPath, []string{"fn", "source"},
				nil, nil)...)
		if kptFnBuf, err = util.RunCmdOut(cmd); err != nil {
			return nil, fmt.Errorf("kpt source the fn-path config %v", err)
		}
	} else {
		kptFnBuf = buf
	}
	kptFn, err := k.getKptFunc(kptFnBuf)
	if err != nil {
		return nil, err
	}

	// Hydrate the manifests source.
	_, err = kustomize.FindKustomizationConfig(k.Dir)
	// Only run kustomize if kustomization.yaml is found.
	if err == nil {
		// Note: A tmp dir is used to provide kustomize the manifest directory.
		// Once the unified kpt/kustomize is done, kustomize can be run as a kpt fn step and
		// this additional directory creation/deletion will no longer be needed.
		if err := os.RemoveAll(tmpKustomizeDir); err != nil {
			return nil, fmt.Errorf("removing %v:%w", tmpKustomizeDir, err)
		}
		if err := os.MkdirAll(tmpKustomizeDir, os.ModePerm); err != nil {
			return nil, err
		}
		defer func() {
			os.RemoveAll(tmpKustomizeDir)
		}()

		err = k.sink(ctx, buf, tmpKustomizeDir)
		if err != nil {
			return nil, err
		}

		cmd := exec.CommandContext(ctx, "kustomize", append([]string{"build"}, tmpKustomizeDir)...)
		buf, err = util.RunCmdOut(cmd)
		if err != nil {
			return nil, fmt.Errorf("kustomize build: %w", err)
		}
	}
	// Run kpt functions against the hydrated manifests.
	cmd = exec.CommandContext(ctx, "kpt", kptCommandArgs("", []string{"fn", "run"}, flags, nil)...)
	buf = append(buf, []byte("---\n")...)
	buf = append(buf, kptFn...)
	cmd.Stdin = bytes.NewBuffer(buf)
	buf, err = util.RunCmdOut(cmd)
	if err != nil {
		return nil, fmt.Errorf("running kpt functions: %w", err)
	}

	// Store the manipulated manifests to the sink dir.
	if k.Fn.SinkDir != "" {
		if err := os.RemoveAll(k.Fn.SinkDir); err != nil {
			return nil, fmt.Errorf("deleting sink directory %s: %w", k.Fn.SinkDir, err)
		}

		if err := os.MkdirAll(k.Fn.SinkDir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("creating sink directory %s: %w", k.Fn.SinkDir, err)
		}
		err = k.sink(ctx, buf, k.Fn.SinkDir)
		if err != nil {
			return nil, fmt.Errorf("sinking to directory %s: %w", k.Fn.SinkDir, err)
		}
		fmt.Printf(
			"Manipulated resources are flattended and stored in your sink directory: %v\n",
			k.Fn.SinkDir)
	}

	var manifests manifest.ManifestList
	if len(buf) > 0 {
		manifests.Append(buf)
	}
	manifests, err = k.excludeKptFn(manifests)
	if err != nil {
		return nil, fmt.Errorf("excluding kpt functions from manifests: %w", err)
	}
	manifests, err = manifests.ReplaceImages(builds)
	if err != nil {
		return nil, fmt.Errorf("replacing images in manifests: %w", err)
	}

	if manifests, err = manifest.ApplyTransforms(manifests, builds, k.insecureRegistries, debugHelpersRegistry); err != nil {
		return nil, err
	}

	return manifests.SetLabels(k.labels)
}

func (k *Deployer) getKptFunc(buf []byte) ([]byte, error) {
	input := bytes.NewBufferString(string(buf))
	rl := framework.ResourceList{
		Reader: input,
	}
	// Manipulate the kustomize "Rnode"(Kustomize term) and pulls out the "Items"
	// from ResourceLists.
	if err := rl.Read(); err != nil {
		return nil, fmt.Errorf("reading ResourceList %w", err)
	}
	var kptFn []byte
	for i := range rl.Items {
		item, err := rl.Items[i].String()
		if err != nil {
			return nil, fmt.Errorf("reading Item %w", err)
		}
		var obj unstructured.Unstructured
		jByte, err := k8syaml.YAMLToJSON([]byte(item))
		if err != nil {
			continue
		}
		if err := obj.UnmarshalJSON(jByte); err != nil {
			continue
		}
		// Found kpt fn.
		if _, ok := obj.GetAnnotations()[kptFnAnnotation]; ok {
			kptFn = append(kptFn, []byte(item)...)
		}
	}
	return kptFn, nil
}

func (k *Deployer) sink(ctx context.Context, buf []byte, sinkDir string) error {
	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(sinkDir, []string{"fn", "sink"}, nil, nil)...)
	cmd.Stdin = bytes.NewBuffer(buf)
	_, err := util.RunCmdOut(cmd)
	return err
}

// excludeKptFn adds an annotation "config.kubernetes.io/local-config: 'true'" to kpt function.
// This will exclude kpt functions from deployed to the cluster in `kpt live apply`.
func (k *Deployer) excludeKptFn(originalManifest manifest.ManifestList) (manifest.ManifestList, error) {
	var newManifest manifest.ManifestList
	for _, yByte := range originalManifest {
		// Convert yaml byte config to unstructured.Unstructured
		jByte, err := k8syaml.YAMLToJSON(yByte)
		if err != nil {
			return nil, fmt.Errorf("yaml to json error: %w", err)
		}
		var obj unstructured.Unstructured
		if err := obj.UnmarshalJSON(jByte); err != nil {
			return nil, fmt.Errorf("unmarshaling config: %w", err)
		}
		// skip if the resource is not kpt fn config.
		if _, ok := obj.GetAnnotations()[kptFnAnnotation]; !ok {
			newManifest = append(newManifest, yByte)
			continue
		}
		// skip if the kpt fn has local-config annotation specified.
		if _, ok := obj.GetAnnotations()[kptFnLocalConfig]; ok {
			newManifest = append(newManifest, yByte)
			continue
		}

		// Add "local-config" annotation to kpt fn config.
		anns := obj.GetAnnotations()
		anns[kptFnLocalConfig] = "true"
		obj.SetAnnotations(anns)
		jByte, err = obj.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshaling to json: %w", err)
		}
		newYByte, err := k8syaml.JSONToYAML(jByte)
		if err != nil {
			return nil, fmt.Errorf("converting json to yaml: %w", err)
		}
		newManifest.Append(newYByte)
	}
	return newManifest, nil
}

// getApplyDir returns the path to applyDir if specified by the user. Otherwise, getApplyDir
// creates a hidden directory named .kpt-hydrated in place of applyDir.
func (k *Deployer) getApplyDir(ctx context.Context) (string, error) {
	if k.Live.Apply.Dir != "" {
		if _, err := os.Stat(k.Live.Apply.Dir); os.IsNotExist(err) {
			return "", err
		}
		return k.Live.Apply.Dir, nil
	}

	// 0755 is a permission setting where the owner can read, write, and execute.
	// Others can read and execute but not modify the directory.
	if err := os.MkdirAll(kptHydrated, os.ModePerm); err != nil {
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
func (k *Deployer) getKptFnRunArgs() ([]string, error) {
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
	// fn-path is not supported due to kpt issue https://github.com/GoogleContainerTools/kpt/issues/1149
	if len(k.Fn.FnPath) > 0 {
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
func (k *Deployer) getKptLiveApplyArgs() []string {
	var flags []string

	if len(k.Live.Options.PollPeriod) > 0 {
		flags = append(flags, "--poll-period", k.Live.Options.PollPeriod)
	}

	if len(k.Live.Options.PrunePropagationPolicy) > 0 {
		flags = append(flags, "--prune-propagation-policy", k.Live.Options.PrunePropagationPolicy)
	}

	if len(k.Live.Options.PruneTimeout) > 0 {
		flags = append(flags, "--prune-timeout", k.Live.Options.PruneTimeout)
	}

	if len(k.Live.Options.ReconcileTimeout) > 0 {
		flags = append(flags, "--reconcile-timeout", k.Live.Options.ReconcileTimeout)
	}

	return flags
}

// getKptLiveInitArgs returns a list of arguments that the user specified for the `kpt live init` command.
func (k *Deployer) getKptLiveInitArgs() []string {
	var flags []string

	if len(k.Live.Apply.InventoryID) > 0 {
		flags = append(flags, "--inventory-id", k.Live.Apply.InventoryID)
	}

	if len(k.Live.Apply.InventoryNamespace) > 0 {
		flags = append(flags, "--namespace", k.Live.Apply.InventoryNamespace)
	}

	return flags
}
