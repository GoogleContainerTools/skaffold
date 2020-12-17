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
	"strings"

	"github.com/segmentio/textio"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	deployerr "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/error"
	deployutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// Deployer deploys workflows using kubectl CLI.
type Deployer struct {
	*latest.KubectlDeploy

	originalImages     []build.Artifact
	workingDir         string
	globalConfig       string
	gcsManifestDir     string
	defaultRepo        *string
	kubectl            CLI
	insecureRegistries map[string]bool
	labels             map[string]string
	skipRender         bool
}

// NewDeployer returns a new Deployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewDeployer(cfg Config, labels map[string]string, d *latest.KubectlDeploy) (*Deployer, error) {
	defaultNamespace := ""
	if d.DefaultNamespace != nil {
		var err error
		defaultNamespace, err = util.ExpandEnvTemplate(*d.DefaultNamespace, nil)
		if err != nil {
			return nil, err
		}
	}

	return &Deployer{
		KubectlDeploy:      d,
		workingDir:         cfg.GetWorkingDir(),
		globalConfig:       cfg.GlobalConfig(),
		defaultRepo:        cfg.DefaultRepo(),
		kubectl:            NewCLI(cfg, d.Flags, defaultNamespace),
		insecureRegistries: cfg.GetInsecureRegistries(),
		skipRender:         cfg.SkipRender(),
		labels:             labels,
	}, nil
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *Deployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]string, error) {
	var (
		manifests manifest.ManifestList
		err       error
	)
	if k.skipRender {
		manifests, err = k.readManifests(ctx, false)
	} else {
		manifests, err = k.renderManifests(ctx, out, builds, false)
	}
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

	if err := k.kubectl.WaitForDeletions(ctx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		return nil, err
	}

	if err := k.kubectl.Apply(ctx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		return nil, err
	}

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
	// Get file manifests
	manifests, err := k.Dependencies()
	// Clean the temporary directory that holds the manifests downloaded from GCS
	defer os.RemoveAll(k.gcsManifestDir)

	if err != nil {
		return nil, listManifestErr(fmt.Errorf("listing manifests: %w", err))
	}

	// Append URL manifests
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

func (k *Deployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, offline bool, filepath string) error {
	manifests, err := k.renderManifests(ctx, out, builds, offline)
	if err != nil {
		return err
	}

	return manifest.Write(manifests.String(), filepath, out)
}

func (k *Deployer) renderManifests(ctx context.Context, out io.Writer, builds []build.Artifact, offline bool) (manifest.ManifestList, error) {
	if err := k.kubectl.CheckVersion(ctx); err != nil {
		color.Default.Fprintln(out, "kubectl client version:", k.kubectl.Version(ctx))
		color.Default.Fprintln(out, err)
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
			builds = append(builds, build.Artifact{
				ImageName: artifact.ImageName,
				Tag:       tag,
			})
		}
	}

	manifests, err = manifests.ReplaceImages(builds)
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

		upd, err := rm.ReplaceImages(k.originalImages)
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
