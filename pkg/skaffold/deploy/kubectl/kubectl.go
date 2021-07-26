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

package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/segmentio/textio"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/access"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/log"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Deployer deploys workflows using kubectl CLI.
type Deployer struct {
	*latestV2.KubectlDeploy

	accessor      access.Accessor
	logger        log.Logger
	debugger      debug.Debugger
	statusMonitor status.Monitor
	syncer        sync.Syncer

	originalImages     []graph.Artifact
	podSelector        *kubernetes.ImageList
	hydratedManifests  []string
	workingDir         string
	globalConfig       string
	gcsManifestDir     string
	defaultRepo        *string
	kubectl            CLI
	insecureRegistries map[string]bool
	labels             map[string]string
	skipRender         bool
	hydrationDir       string
}

// NewDeployer returns a new Deployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewDeployer(cfg Config, labels map[string]string, provider deploy.ComponentProvider, d *latestV2.KubectlDeploy, hydrationDir string) (*Deployer, error) {
	defaultNamespace := ""
	if d.DefaultNamespace != nil {
		var err error
		defaultNamespace, err = util.ExpandEnvTemplate(*d.DefaultNamespace, nil)
		if err != nil {
			return nil, err
		}
	}

	podSelector := kubernetes.NewImageList()

	return &Deployer{
		KubectlDeploy:      d,
		podSelector:        podSelector,
		accessor:           provider.Accessor.GetKubernetesAccessor(cfg, podSelector),
		debugger:           provider.Debugger.GetKubernetesDebugger(podSelector),
		logger:             provider.Logger.GetKubernetesLogger(podSelector),
		statusMonitor:      provider.Monitor.GetKubernetesMonitor(cfg),
		syncer:             provider.Syncer.GetKubernetesSyncer(podSelector),
		workingDir:         cfg.GetWorkingDir(),
		globalConfig:       cfg.GlobalConfig(),
		defaultRepo:        cfg.DefaultRepo(),
		kubectl:            NewCLI(cfg, d.Flags, defaultNamespace),
		insecureRegistries: cfg.GetInsecureRegistries(),
		skipRender:         cfg.SkipRender(),
		labels:             labels,
		// hydratedManifests refers to the DIR in the `skaffold apply DIR`. Used in both v1 and v2.
		hydratedManifests: cfg.HydratedManifests(),
		// hydrationDir refers to the path where the hydrated manifests are stored, this is introduced in v2.
		hydrationDir: hydrationDir,
	}, nil
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

func (k *Deployer) TrackBuildArtifacts(artifacts []graph.Artifact) {
	deployutil.AddTagsToPodSelector(artifacts, k.originalImages, k.podSelector)
	k.logger.RegisterArtifacts(artifacts)
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *Deployer) Deploy(ctx context.Context, out io.Writer, builds []graph.Artifact) ([]string, error) {
	var (
		manifests manifest.ManifestList
		err       error
	)
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kubectl",
	})

	// if any hydrated manifests are passed to `skaffold apply`, only deploy these
	// also, manually set the labels to ensure the runID is added
	switch {
	case len(k.hydratedManifests) > 0:
		_, endTrace := instrumentation.StartTrace(ctx, "Deploy_createManifestList")
		manifests, err = createManifestList(k.hydratedManifests)
		if err != nil {
			endTrace(instrumentation.TraceEndError(err))
			return nil, err
		}
		manifests, err = manifests.SetLabels(k.labels)
		endTrace()
	case k.skipRender:
		childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_readManifests")
		manifests, err = k.readManifests(childCtx, false)
		endTrace()
	default:
		childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_renderManifests")
		manifests, err = k.renderManifests(childCtx, out, builds, false)
		endTrace()
	}

	if err != nil {
		return nil, err
	}

	if len(manifests) == 0 {
		return nil, nil
	}
	_, endTrace := instrumentation.StartTrace(ctx, "Deploy_CollectNamespaces")
	namespaces, err := manifests.CollectNamespaces()
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not fetch deployed resource namespace. "+
			"This might cause port-forward and deploy health-check to fail: %w", err))
	}
	endTrace()

	childCtx, endTrace := instrumentation.StartTrace(ctx, "Deploy_WaitForDeletions")
	if err := k.kubectl.WaitForDeletions(childCtx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, err
	}
	endTrace()

	childCtx, endTrace = instrumentation.StartTrace(ctx, "Deploy_KubectlApply")
	if err := k.kubectl.Apply(childCtx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return nil, err
	}

	k.TrackBuildArtifacts(builds)
	endTrace()
	return namespaces, nil
}

func (k *Deployer) manifestFiles(manifests []string) ([]string, error) {
	var nonURLManifests, gcsManifests []string
	for _, manifest := range manifests {
		switch {
		case util.IsURL(manifest):
		case strings.HasPrefix(manifest, "gs://"):
			gcsManifests = append(gcsManifests, manifest)
		default:
			nonURLManifests = append(nonURLManifests, manifest)
		}
	}

	list, err := util.ExpandPathsGlob(k.workingDir, nonURLManifests)
	if err != nil {
		return nil, userErr(fmt.Errorf("expanding kubectl manifest paths: %w", err))
	}

	if len(gcsManifests) != 0 {
		// return tmp dir of the downloaded manifests
		tmpDir, err := manifest.DownloadFromGCS(gcsManifests)
		if err != nil {
			return nil, userErr(fmt.Errorf("downloading from GCS: %w", err))
		}
		k.gcsManifestDir = tmpDir
		l, err := util.ExpandPathsGlob(tmpDir, []string{"*"})
		if err != nil {
			return nil, userErr(fmt.Errorf("expanding kubectl manifest paths: %w", err))
		}
		list = append(list, l...)
	}

	var filteredManifests []string
	for _, f := range list {
		if !kubernetes.HasKubernetesFileExtension(f) {
			if !util.StrSliceContains(manifests, f) {
				logrus.Infof("refusing to deploy/delete non {json, yaml} file %s", f)
				logrus.Info("If you still wish to deploy this file, please specify it directly, outside a glob pattern.")
				continue
			}
		}
		filteredManifests = append(filteredManifests, f)
	}

	return filteredManifests, nil
}

// readManifests reads the manifests to deploy/delete.
func (k *Deployer) readManifests(ctx context.Context, offline bool) (manifest.ManifestList, error) {
	var manifests []string
	var err error

	// v1 kubectl deployer is used. No manifest hydration.
	if len(k.KubectlDeploy.Manifests) > 0 {
		logrus.Warnln("`deploy.kubectl.manfiests` (DEPRECATED) are given, skaffold will skip the `manifests` field. " +
			"If you expect skaffold to render the resources from the `manifests`, please delete the `deploy.kubectl.manfiests` field.")
		manifests, err = k.Dependencies()
		if err != nil {
			return nil, listManifestErr(fmt.Errorf("listing manifests: %w", err))
		}
	} else {
		// v2 kubectl deployer is used. The manifests are read from the hydrated directory.
		manifests, err = k.manifestFiles([]string{filepath.Join(k.hydrationDir, "*")})
		if err != nil {
			return nil, listManifestErr(fmt.Errorf("listing manifests: %w", err))
		}
	}

	// Clean the temporary directory that holds the manifests downloaded from GCS
	defer os.RemoveAll(k.gcsManifestDir)

	// Append URL manifests. URL manifests are excluded from `Dependencies`.
	hasURLManifest := false
	for _, manifest := range k.KubectlDeploy.Manifests {
		if util.IsURL(manifest) {
			manifests = append(manifests, manifest)
			hasURLManifest = true
		}
	}

	if len(manifests) == 0 {
		return manifest.ManifestList{}, nil
	}

	if !offline {
		return k.kubectl.ReadManifests(ctx, manifests)
	}

	// In case no URLs are provided, we can stay offline - no need to run "kubectl create" which
	// would try to connect to a cluster (https://github.com/kubernetes/kubernetes/issues/51475)
	if hasURLManifest {
		return nil, offlineModeErr()
	}
	return createManifestList(manifests)
}

func createManifestList(manifests []string) (manifest.ManifestList, error) {
	var manifestList manifest.ManifestList
	for _, manifestFilePath := range manifests {
		manifestFileContent, err := ioutil.ReadFile(manifestFilePath)
		if err != nil {
			return nil, readManifestErr(fmt.Errorf("reading manifest file %v: %w", manifestFilePath, err))
		}
		manifestList.Append(manifestFileContent)
	}
	return manifestList, nil
}

// readRemoteManifests will try to read manifests from the given kubernetes
// context in the specified namespace and for the specified type
func (k *Deployer) readRemoteManifest(ctx context.Context, name string) ([]byte, error) {
	var args []string
	ns := ""
	if parts := strings.Split(name, ":"); len(parts) > 1 {
		ns = parts[0]
		name = parts[1]
	}
	args = append(args, name, "-o", "yaml")

	var manifest bytes.Buffer
	err := k.kubectl.RunInNamespace(ctx, nil, &manifest, "get", ns, args...)
	if err != nil {
		return nil, readRemoteManifestErr(fmt.Errorf("getting remote manifests: %w", err))
	}

	return manifest.Bytes(), nil
}

func (k *Deployer) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool, filepath string) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kubectl",
	})

	childCtx, endTrace := instrumentation.StartTrace(ctx, "Render_renderManifests")
	manifests, err := k.renderManifests(childCtx, out, builds, offline)
	if err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return err
	}
	endTrace()

	_, endTrace = instrumentation.StartTrace(ctx, "Render_manifest.Write")
	defer endTrace()
	return manifest.Write(manifests.String(), filepath, out)
}

// renderManifests transforms the manifests' images with the actual image sha1 built from skaffold build.
func (k *Deployer) renderManifests(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestList, error) {
	if err := k.kubectl.CheckVersion(ctx); err != nil {
		output.Default.Fprintln(out, "kubectl client version:", k.kubectl.Version(ctx))
		output.Default.Fprintln(out, err)
	}

	debugHelpersRegistry, err := config.GetDebugHelpersRegistry(k.globalConfig)
	if err != nil {
		return nil, deployerr.DebugHelperRetrieveErr(fmt.Errorf("retrieving debug helpers registry: %w", err))
	}

	manifests, err := k.readManifests(ctx, offline)
	if err != nil {
		return nil, err
	}

	for _, m := range k.RemoteManifests {
		manifest, err := k.readRemoteManifest(ctx, m)
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, manifest)
	}

	if len(k.originalImages) == 0 {
		k.originalImages, err = manifests.GetImages()
		if err != nil {
			return nil, err
		}
	}

	if len(manifests) == 0 {
		return nil, nil
	}

	if len(builds) == 0 {
		for _, artifact := range k.originalImages {
			tag, err := deployutil.ApplyDefaultRepo(k.globalConfig, k.defaultRepo, artifact.Tag)
			if err != nil {
				return nil, err
			}
			builds = append(builds, graph.Artifact{
				ImageName: artifact.ImageName,
				Tag:       tag,
			})
		}
	}

	manifests, err = manifests.ReplaceImages(ctx, builds)
	if err != nil {
		return nil, err
	}

	if manifests, err = manifest.ApplyTransforms(manifests, builds, k.insecureRegistries, debugHelpersRegistry); err != nil {
		return nil, err
	}

	return manifests.SetLabels(k.labels)
}

// Cleanup deletes what was deployed by calling Deploy.
func (k *Deployer) Cleanup(ctx context.Context, out io.Writer) error {
	instrumentation.AddAttributesToCurrentSpanFromContext(ctx, map[string]string{
		"DeployerType": "kubectl",
	})
	manifests, err := k.readManifests(ctx, false)
	if err != nil {
		return err
	}

	// revert remote manifests
	// TODO(dgageot): That seems super dangerous and I don't understand
	// why we need to update resources just before we delete them.
	if len(k.RemoteManifests) > 0 {
		var rm manifest.ManifestList
		for _, m := range k.RemoteManifests {
			manifest, err := k.readRemoteManifest(ctx, m)
			if err != nil {
				return err
			}
			rm = append(rm, manifest)
		}

		upd, err := rm.ReplaceImages(ctx, k.originalImages)
		if err != nil {
			return err
		}

		if err := k.kubectl.Apply(ctx, out, upd); err != nil {
			return err
		}
	}

	if err := k.kubectl.Delete(ctx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		return err
	}

	return nil
}

// Dependencies lists all the files that describe what needs to be deployed.
func (k *Deployer) Dependencies() ([]string, error) {
	return k.manifestFiles(k.KubectlDeploy.Manifests)
}
