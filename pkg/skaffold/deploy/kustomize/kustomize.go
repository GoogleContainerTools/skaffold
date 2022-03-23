/*
Copyright 2019 The Skaffold Authors

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
	"os"
	"os/exec"
	"path/filepath"

	"github.com/segmentio/textio"
	yamlv3 "gopkg.in/yaml.v3"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	component "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/component/kubernetes"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/hooks"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	kstatus "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/loader"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	olog "github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringset"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/warnings"
)

var (
	DefaultKustomizePath = "."
	KustomizeFilePaths   = []string{"kustomization.yaml", "kustomization.yml", "Kustomization"}
	basePath             = "base"
	KustomizeBinaryCheck = kustomizeBinaryExists // For testing
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

// Deployer deploys workflows using kustomize CLI.
type Deployer struct {
	*latestV1.KustomizeDeploy

	accessor      access.Accessor
	logger        log.Logger
	imageLoader   loader.ImageLoader
	debugger      debug.Debugger
	statusMonitor kstatus.Monitor
	syncer        sync.Syncer
	hookRunner    hooks.Runner

	podSelector    *kubernetes.ImageList
	originalImages []graph.Artifact // the set of images parsed from the Deployer's manifest set
	localImages    []graph.Artifact // the set of images marked as "local" by the Runner

	kubectl             kubectl.CLI
	insecureRegistries  map[string]bool
	labels              map[string]string
	globalConfig        string
	useKubectlKustomize bool

	namespaces *[]string

	transformableAllowlist map[apimachinery.GroupKind]latestV1.ResourceFilter
	transformableDenylist  map[apimachinery.GroupKind]latestV1.ResourceFilter
}

func NewDeployer(cfg kubectl.Config, labeller *label.DefaultLabeller, d *latestV1.KustomizeDeploy) (*Deployer, error) {
	defaultNamespace := ""
	if d.DefaultNamespace != nil {
		var err error
		defaultNamespace, err = util.ExpandEnvTemplate(*d.DefaultNamespace, nil)
		if err != nil {
			return nil, err
		}
	}

	kubectl := kubectl.NewCLI(cfg, d.Flags, defaultNamespace)
	// if user has kustomize binary, prioritize that over kubectl kustomize
	useKubectlKustomize := !KustomizeBinaryCheck() && kubectlVersionCheck(kubectl)

	podSelector := kubernetes.NewImageList()
	namespaces, err := deployutil.GetAllPodNamespaces(cfg.GetNamespace(), cfg.GetPipelines())
	if err != nil {
		olog.Entry(context.TODO()).Warn("unable to parse namespaces - deploy might not work correctly!")
	}
	logger := component.NewLogger(cfg, kubectl.CLI, podSelector, &namespaces)
	transformableAllowlist, transformableDenylist, err := deployutil.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return nil, err
	}
	return &Deployer{
		KustomizeDeploy:        d,
		podSelector:            podSelector,
		namespaces:             &namespaces,
		accessor:               component.NewAccessor(cfg, cfg.GetKubeContext(), kubectl.CLI, podSelector, labeller, &namespaces),
		debugger:               component.NewDebugger(cfg.Mode(), podSelector, &namespaces, cfg.GetKubeContext()),
		hookRunner:             hooks.NewDeployRunner(kubectl.CLI, d.LifecycleHooks, &namespaces, logger.GetFormatter(), hooks.NewDeployEnvOpts(labeller.GetRunID(), kubectl.KubeContext, namespaces)),
		imageLoader:            component.NewImageLoader(cfg, kubectl.CLI),
		logger:                 logger,
		statusMonitor:          component.NewMonitor(cfg, cfg.GetKubeContext(), labeller, &namespaces),
		syncer:                 component.NewSyncer(kubectl.CLI, &namespaces, logger.GetFormatter()),
		kubectl:                kubectl,
		insecureRegistries:     cfg.GetInsecureRegistries(),
		globalConfig:           cfg.GlobalConfig(),
		labels:                 labeller.Labels(),
		useKubectlKustomize:    useKubectlKustomize,
		transformableAllowlist: transformableAllowlist,
		transformableDenylist:  transformableDenylist,
	}, nil
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

// Check for existence of kustomize binary in user's PATH
func kustomizeBinaryExists() bool {
	_, err := exec.LookPath("kustomize")

	return err == nil
}

func (k *Deployer) HasRunnableHooks() bool {
	return len(k.KustomizeDeploy.LifecycleHooks.PreHooks) > 0 || len(k.KustomizeDeploy.LifecycleHooks.PostHooks) > 0
}

func (k *Deployer) PreDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PreHooks")
	if err := k.hookRunner.RunPreHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

func (k *Deployer) PostDeployHooks(ctx context.Context, out io.Writer) error {
	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_PostHooks")
	if err := k.hookRunner.RunPostHooks(childCtx, out); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()
	return nil
}

// Check that kubectl version is valid to use kubectl kustomize
func kubectlVersionCheck(kubectl kubectl.CLI) bool {
	gt, err := kubectl.CompareVersionTo(context.Background(), 1, 14)
	if err != nil {
		return false
	}

	return gt == 1
}

// Deploy runs `kubectl apply` on the manifest generated by kustomize.
func (k *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kustomize",
	})

	// Check that the cluster is reachable.
	// This gives a better error message when the cluster can't
	// be reached.
	if err := kubernetes.FailIfClusterIsNotReachable(k.kubectl.KubeContext); err != nil {
		return fmt.Errorf("unable to connect to Kubernetes: %w", err)
	}

	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_renderManifests")
	manifests, err := k.renderManifests(childCtx, out, builds)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	if len(manifests) == 0 {
		endTrace()
		return nil
	}
	endTrace()

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_LoadImages")
	if err := k.imageLoader.LoadImages(childCtx, out, k.localImages, k.originalImages, builds); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	_, endTrace = instrumentation.StartTrace(ctx, "Deploy_CollectNamespaces")
	namespaces, err := manifests.CollectNamespaces()
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not fetch deployed resource namespace. "+
			"This might cause port-forward and deploy health-check to fail: %w", err))
	}
	endTrace()

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_WaitForDeletions")
	if err := k.kubectl.WaitForDeletions(childCtx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_Apply")
	if err := k.kubectl.Apply(childCtx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}

	k.TrackBuildArtifacts(builds)
	k.statusMonitor.RegisterDeployManifests(manifests)
	endTrace()

	k.trackNamespaces(namespaces)
	return nil
}

func (k *Deployer) renderManifests(ctx context.Context, out io.Writer, builds []graph.Artifact) (manifest.ManifestList, error) {
	if err := k.kubectl.CheckVersion(ctx); err != nil {
		output.Default.Fprintln(out, "kubectl client version:", k.kubectl.Version(ctx))
		output.Default.Fprintln(out, err)
	}

	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(k.globalConfig)
	if err != nil {
		return nil, deployerr.DebugHelperRetrieveErr(err)
	}

	manifests, err := k.readManifests(ctx)
	if err != nil {
		return nil, err
	}

	if len(manifests) == 0 {
		return nil, nil
	}

	if len(k.originalImages) == 0 {
		k.originalImages, err = manifests.GetImages(manifest.NewResourceSelectorImages(k.transformableAllowlist, k.transformableDenylist))
		if err != nil {
			return nil, err
		}
	}

	manifests, err = manifests.ReplaceImages(ctx, builds, manifest.NewResourceSelectorImages(k.transformableAllowlist, k.transformableDenylist))
	if err != nil {
		return nil, err
	}

	if manifests, err = manifest.ApplyTransforms(manifests, builds, k.insecureRegistries, debugHelpersRegistry); err != nil {
		return nil, err
	}

	return manifests.SetLabels(k.labels, manifest.NewResourceSelectorLabels(k.transformableAllowlist, k.transformableDenylist))
}

// Cleanup deletes what was deployed by calling Deploy.
func (k *Deployer) Cleanup(ctx context.Context, out io.Writer, dryRun bool) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kustomize",
	})
	manifests, err := k.readManifests(ctx)
	if err != nil {
		return err
	}
	if dryRun {
		for _, manifest := range manifests {
			output.White.Fprintf(out, "---\n%s", manifest)
		}
		return nil
	}
	if err := k.kubectl.Delete(ctx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		return err
	}

	return nil
}

// Dependencies lists all the files that describe what needs to be deployed.
func (k *Deployer) Dependencies() ([]string, error) {
	deps := stringset.New()
	for _, kustomizePath := range k.KustomizePaths {
		expandedKustomizePath, err := util.ExpandEnvTemplate(kustomizePath, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to parse path %q: %w", kustomizePath, err)
		}
		depsForKustomization, err := DependenciesForKustomization(expandedKustomizePath)
		if err != nil {
			return nil, userErr(err)
		}
		deps.Insert(depsForKustomization...)
	}
	return deps.ToList(), nil
}

func (k *Deployer) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, filepath string) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kustomize",
	})

	childCtx, endTrace := instrumentation.StartTrace(ctx, "Render_renderManifests")
	manifests, err := k.renderManifests(childCtx, out, builds)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	k.statusMonitor.RegisterDeployManifests(manifests)

	_, endTrace = instrumentation.StartTrace(ctx, "Render_manifest.Write")
	defer endTrace()
	return manifest.Write(manifests.String(), filepath, out)
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

func (k *Deployer) readManifests(ctx context.Context) (manifest.ManifestList, error) {
	var manifests manifest.ManifestList
	for _, kustomizePath := range k.KustomizePaths {
		var out []byte
		var err error

		expandedKustomizePath, err := util.ExpandEnvTemplate(kustomizePath, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to parse path %q: %w", kustomizePath, err)
		}

		if k.useKubectlKustomize {
			out, err = k.kubectl.Kustomize(ctx, BuildCommandArgs(k.BuildArgs, expandedKustomizePath))
		} else {
			cmd := exec.CommandContext(ctx, "kustomize", append([]string{"build"}, BuildCommandArgs(k.BuildArgs, expandedKustomizePath)...)...)
			out, err = util.RunCmdOut(ctx, cmd)
		}

		if err != nil {
			return nil, userErr(err)
		}

		if len(out) == 0 {
			continue
		}
		manifests.Append(out)
	}
	return manifests, nil
}

func IsKustomizationBase(path string) bool {
	return filepath.Dir(path) == basePath
}

func IsKustomizationPath(path string) bool {
	filename := filepath.Base(path)
	for _, candidate := range KustomizeFilePaths {
		if filename == candidate {
			return true
		}
	}
	return false
}
