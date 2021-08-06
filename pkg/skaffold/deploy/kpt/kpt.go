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

	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "sigs.k8s.io/yaml"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	component "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/component/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	kloader "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
	kstatus "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	inventoryTemplate = "inventory-template.yaml"
	kptHydrated       = ".kpt-hydrated"
	tmpKustomizeDir   = ".kustomize"
	kptFnAnnotation   = "config.kubernetes.io/function"
	kptFnLocalConfig  = "config.kubernetes.io/local-config"

	kptDownloadLink        = "https://googlecontainertools.github.io/kpt/installation/"
	kptMinVersionInclusive = "v0.38.1"
	kptMaxVersionExclusive = "v1.0.0"

	kustomizeDownloadLink  = "https://kubernetes-sigs.github.io/kustomize/installation/"
	kustomizeMinVersion    = "v3.2.3"
	kustomizeVersionRegexP = `{Version:(kustomize/)?(\S+) GitCommit:\S+ BuildDate:\S+ GoOs:\S+ GoArch:\S+}`
)

// Deployer deploys workflows with kpt CLI
type Deployer struct {
	*latestV1.KptDeploy

	accessor      access.Accessor
	logger        log.Logger
	debugger      debug.Debugger
	imageLoader   loader.ImageLoader
	statusMonitor status.Monitor
	syncer        sync.Syncer

	podSelector    *kubernetes.ImageList
	originalImages []graph.Artifact // the set of images parsed from the Deployer's manifest set
	localImages    []graph.Artifact // the set of images marked as "local" by the Runner

	insecureRegistries map[string]bool
	labels             map[string]string
	globalConfig       string
	hasKustomization   func(string) bool
	kubeContext        string
	kubeConfig         string
	namespace          string

	namespaces *[]string
}

type Config interface {
	kubectl.Config
	kstatus.Config
	portforward.Config
	kloader.Config
}

// NewDeployer generates a new Deployer object contains the kptDeploy schema.
func NewDeployer(cfg Config, labeller *label.DefaultLabeller, d *latestV1.KptDeploy) *Deployer {
	podSelector := kubernetes.NewImageList()
	kubectl := pkgkubectl.NewCLI(cfg, cfg.GetKubeNamespace())
	namespaces, err := deployutil.GetAllPodNamespaces(cfg.GetNamespace(), cfg.GetPipelines())
	if err != nil {
		logrus.Warnf("unable to parse namespaces - deploy might not work correctly!")
	}
	logger := component.NewLogger(cfg, kubectl, podSelector, &namespaces)
	return &Deployer{
		KptDeploy:          d,
		podSelector:        podSelector,
		namespaces:         &namespaces,
		accessor:           component.NewAccessor(cfg, cfg.GetKubeContext(), kubectl, podSelector, labeller, &namespaces),
		debugger:           component.NewDebugger(cfg.Mode(), podSelector, &namespaces, cfg.GetKubeContext()),
		imageLoader:        component.NewImageLoader(cfg, kubectl),
		logger:             logger,
		statusMonitor:      component.NewMonitor(cfg, cfg.GetKubeContext(), labeller, &namespaces),
		syncer:             component.NewSyncer(kubectl, &namespaces, logger.GetFormatter()),
		insecureRegistries: cfg.GetInsecureRegistries(),
		labels:             labeller.Labels(),
		globalConfig:       cfg.GlobalConfig(),
		hasKustomization:   hasKustomization,
		kubeContext:        cfg.GetKubeContext(),
		kubeConfig:         cfg.GetKubeConfig(),
		namespace:          cfg.GetKubeNamespace(),
	}
}

func (k *Deployer) trackNamespaces(namespaces []string) {
	*k.namespaces = deployutil.ConsolidateNamespaces(*k.namespaces, namespaces)
}

func (k *Deployer) GetAccessor() access.Accessor {
	return k.accessor
}

func (k *Deployer) GetDebugger() debug.Debugger {
	return k.debugger
}

func (k *Deployer) GetLogger() log.Logger {
	return k.logger
}

func (k *Deployer) GetStatusMonitor() status.Monitor {
	return k.statusMonitor
}

func (k *Deployer) GetSyncer() sync.Syncer {
	return k.syncer
}

func (k *Deployer) RegisterLocalImages(images []graph.Artifact) {
	k.localImages = images
}

func (k *Deployer) TrackBuildArtifacts(artifacts []graph.Artifact) {
	deployutil.AddTagsToPodSelector(artifacts, k.originalImages, k.podSelector)
	k.logger.RegisterArtifacts(artifacts)
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
	// kpt follows semver but does not have "v" prefix.
	version := "v" + strings.TrimSuffix(string(out), "\n")
	if !semver.IsValid(version) {
		return fmt.Errorf("unknown kpt version %v\nPlease install "+
			"kpt %v <= version < %v\nSee kpt installation: %v",
			string(out), kptMinVersionInclusive, kptMaxVersionExclusive, kptDownloadLink)
	}
	if semver.Compare(version, kptMinVersionInclusive) < 0 ||
		semver.Compare(version, kptMaxVersionExclusive) >= 0 {
		return fmt.Errorf("you are using kpt %q\nPlease install "+
			"kpt %v <= version < %v\nSee kpt installation: %v",
			version, kptMinVersionInclusive, kptMaxVersionExclusive, kptDownloadLink)
	}

	// Users can choose not to use kustomize in kpt deployer mode. We only check the kustomize
	// version when kustomization.yaml config is directed under .deploy.kpt.dir path.
	if hasKustomization(dir) {
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
			output.Yellow.Fprintf(stdout, "unable to determine kustomize version from %q\n"+
				"You can download the official kustomize (recommended >= %v) from %v\n",
				string(out), kustomizeMinVersion, kustomizeDownloadLink)
		} else if !semver.IsValid(match[2]) || semver.Compare(match[2], kustomizeMinVersion) < 0 {
			output.Yellow.Fprintf(stdout, "you are using kustomize version %q "+
				"(recommended >= %v). You can download the official kustomize from %v\n",
				match[2], kustomizeMinVersion, kustomizeDownloadLink)
		}
	}
	return nil
}

// Deploy hydrates the manifests using kustomizations and kpt functions as described in the render method,
// outputs them to the applyDir, and runs `kpt live apply` against applyDir to create resources in the cluster.
// `kpt live apply` supports automated pruning declaratively via resources in the applyDir.
func (k *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kpt",
	})

	// Check that the cluster is reachable.
	// This gives a better error message when the cluster can't
	// be reached.
	if err := kubernetes.FailIfClusterIsNotReachable(k.kubeContext); err != nil {
		return fmt.Errorf("unable to connect to Kubernetes: %w", err)
	}

	_, endTrace := instrumentation.StartTrace(ctx, "Deploy_sanityCheck")
	if err := sanityCheck(k.Dir, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_loadImages")
	if err := k.imageLoader.LoadImages(childCtx, out, k.localImages, k.originalImages, builds); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	_, endTrace = instrumentation.StartTrace(ctx, "Deploy_renderManifests")
	manifests, err := k.renderManifests(childCtx, builds)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	if len(manifests) == 0 {
		endTrace()
		return nil
	}
	endTrace()

	_, endTrace = instrumentation.StartTrace(ctx, "Deploy_CollectNamespaces")
	namespaces, err := manifests.CollectNamespaces()
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not fetch deployed resource namespace. "+
			"This might cause port-forward and deploy health-check to fail: %w", err))
	}
	endTrace()

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_getApplyDir")
	applyDir, err := k.getApplyDir(childCtx)
	if err != nil {
		return fmt.Errorf("getting applyDir: %w", err)
	}
	endTrace()

	_, endTrace = instrumentation.StartTrace(ctx, "Deploy_manifest.Write")
	if err = sink(ctx, []byte(manifests.String()), applyDir); err != nil {
		return err
	}
	endTrace()

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_execKptCommand")
	cmd := exec.CommandContext(childCtx, "kpt", kptCommandArgs(applyDir, []string{"live", "apply"}, k.getKptLiveApplyArgs(), nil)...)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	k.TrackBuildArtifacts(builds)
	endTrace()
	k.trackNamespaces(namespaces)
	return nil
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
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kpt",
	})

	applyDir, err := k.getApplyDir(ctx)
	if err != nil {
		return fmt.Errorf("getting applyDir: %w", err)
	}

	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(applyDir, []string{"live", "destroy"}, k.getGlobalFlags(), nil)...)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := util.RunCmd(cmd); err != nil {
		return err
	}

	return nil
}

// Render hydrates manifests using both kustomization and kpt functions.
func (k *Deployer) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, _ bool, filepath string) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kubectl",
	})

	_, endTrace := instrumentation.StartTrace(ctx, "Render_sanityCheck")

	if err := sanityCheck(k.Dir, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	childCtx, endTrace := instrumentation.StartTrace(ctx, "Render_renderManifests")
	manifests, err := k.renderManifests(childCtx, builds)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	_, endTrace = instrumentation.StartTrace(ctx, "Render_manifest.Write")
	defer endTrace()
	return manifest.Write(manifests.String(), filepath, out)
}

// renderManifests handles a majority of the hydration process for manifests.
// This involves reading configs from a source directory, running kustomize build, running kpt pipelines,
// adding image digests, and adding run-id labels.
func (k *Deployer) renderManifests(ctx context.Context, builds []graph.Artifact) (
	manifest.ManifestList, error) {
	flags, err := k.getKptFnRunArgs()
	if err != nil {
		return nil, err
	}

	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(k.globalConfig)
	if err != nil {
		return nil, fmt.Errorf("retrieving debug helpers registry: %w", err)
	}

	var buf []byte
	// Read the manifests under k.Dir as "source".
	cmd := exec.CommandContext(
		ctx, "kpt", kptCommandArgs(k.Dir, []string{"fn", "source"},
			nil, nil)...)
	if buf, err = util.RunCmdOut(cmd); err != nil {
		return nil, fmt.Errorf("reading config manifests: %w", err)
	}

	// Run kpt functions against the manifests read from source.
	cmd = exec.CommandContext(ctx, "kpt", kptCommandArgs("", []string{"fn", "run"}, flags, nil)...)
	cmd.Stdin = bytes.NewBuffer(buf)
	if buf, err = util.RunCmdOut(cmd); err != nil {
		return nil, fmt.Errorf("running kpt functions: %w", err)
	}

	// Run kustomize on the output from the kpt functions if a kustomization is found.
	// Note: kustomize cannot be used as a kpt fn yet and thus we run kustomize in a temp dir
	// in the kpt pipeline:
	// kpt source -->  kpt run --> (workaround if kustomization exists) kustomize build --> kpt sink.
	//
	// Note: Optimally the user would be able to control the order in which kpt functions and
	// Kustomize build happens, and even have Kustomize build happen between Kpt fn invocations.
	// However, since we currently don't expose an API supporting that level of control running
	// Kustomize build last seems like the best option.
	// Pros:
	// - Kustomize will remove all comments which breaks any Kpt fn relying on YAML comments. This
	//   includes https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/master/functions/go/apply-setters
	//   which is the upcoming replacement for kpt cfg set and it will likely receive wide usage.
	//   This is the main reason for Kustomize last approach winning.
	// - Kustomize mangles the directory structure so any Kpt fn relying on th relative file
	//   location of a resource as described by the config.kubernetes.io/path annotation will break
	//   if run after Kustomize.
	// - This allows Kpt fns to modify and even create Kustomizations.
	// Cons:
	// - Kpt fns that expects the output of kustomize build as input might not work as expected,
	//   especially if the Kustomization references resources outside of the kpt dir.
	// - Kpt fn run chokes on JSON patch files because the root node is an array. This can be worked
	//   around by avoiding the file extensions kpt fn reads from for such files (.yaml, .yml and
	//   .json) or inlining the patch.
	defer func() {
		if err = os.RemoveAll(tmpKustomizeDir); err != nil {
			fmt.Printf("Unable to delete temporary Kusomize directory: %v\n", err)
		}
	}()
	if err = sink(ctx, buf, tmpKustomizeDir); err != nil {
		return nil, err
	}

	// Only run kustomize if kustomization.yaml is found in the output from the kpt functions.
	if k.hasKustomization(tmpKustomizeDir) {
		cmd = exec.CommandContext(ctx, "kustomize", append([]string{"build"}, tmpKustomizeDir)...)
		if buf, err = util.RunCmdOut(cmd); err != nil {
			return nil, fmt.Errorf("kustomize build: %w", err)
		}
	}

	// Store the manipulated manifests to the sink dir.
	if k.Fn.SinkDir != "" {
		if err = sink(ctx, buf, k.Fn.SinkDir); err != nil {
			return nil, err
		}
		fmt.Printf("Manipulated resources are stored in your sink directory: %v\n", k.Fn.SinkDir)
	}

	var manifests manifest.ManifestList
	if len(buf) > 0 {
		manifests.Append(buf)
	}
	manifests, err = k.excludeKptFn(manifests)
	if err != nil {
		return nil, fmt.Errorf("excluding kpt functions from manifests: %w", err)
	}
	if k.originalImages == nil {
		k.originalImages, err = manifests.GetImages()
		if err != nil {
			return nil, err
		}
	}
	manifests, err = manifests.ReplaceImages(ctx, builds)
	if err != nil {
		return nil, fmt.Errorf("replacing images in manifests: %w", err)
	}

	if manifests, err = manifest.ApplyTransforms(manifests, builds, k.insecureRegistries, debugHelpersRegistry); err != nil {
		return nil, err
	}

	return manifests.SetLabels(k.labels)
}

func sink(ctx context.Context, buf []byte, sinkDir string) error {
	if err := os.RemoveAll(sinkDir); err != nil {
		return fmt.Errorf("deleting sink directory %s: %w", sinkDir, err)
	}

	if err := os.MkdirAll(sinkDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating sink directory %s: %w", sinkDir, err)
	}

	cmd := exec.CommandContext(ctx, "kpt", kptCommandArgs(sinkDir, []string{"fn", "sink"}, nil, nil)...)
	cmd.Stdin = bytes.NewBuffer(buf)
	if _, err := util.RunCmdOut(cmd); err != nil {
		return fmt.Errorf("sinking to directory %s: %w", sinkDir, err)
	}
	return nil
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
	var flags []string

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

	flags = append(flags, k.getGlobalFlags()...)
	return flags
}

// getKptLiveInitArgs returns a list of arguments that the user specified for the `kpt live init` command.
func (k *Deployer) getKptLiveInitArgs() []string {
	var flags []string

	if len(k.Live.Apply.InventoryID) > 0 {
		flags = append(flags, "--inventory-id", k.Live.Apply.InventoryID)
	}

	flags = append(flags, k.getGlobalFlags()...)
	return flags
}
func (k *Deployer) getGlobalFlags() []string {
	var flags []string

	if k.kubeContext != "" {
		flags = append(flags, "--context", k.kubeContext)
	}
	if k.kubeConfig != "" {
		flags = append(flags, "--kubeconfig", k.kubeConfig)
	}
	if len(k.Live.Apply.InventoryNamespace) > 0 {
		flags = append(flags, "--namespace", k.Live.Apply.InventoryNamespace)
	} else if k.namespace != "" {
		// Note: UI duplication.
		flags = append(flags, "--namespace", k.namespace)
	}

	return flags
}

func hasKustomization(dir string) bool {
	_, err := kustomize.FindKustomizationConfig(dir)
	return err == nil
}
