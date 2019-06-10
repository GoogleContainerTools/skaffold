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

package deploy

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// KubectlDeployer deploys workflows using kubectl CLI.
type KubectlDeployer struct {
	*latest.KubectlDeploy

	workingDir         string
	kubectl            kubectl.CLI
	defaultRepo        string
	insecureRegistries map[string]bool
}

// NewKubectlDeployer returns a new KubectlDeployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewKubectlDeployer(runCtx *runcontext.RunContext) *KubectlDeployer {
	return &KubectlDeployer{
		KubectlDeploy: runCtx.Cfg.Deploy.KubectlDeploy,
		workingDir:    runCtx.WorkingDir,
		kubectl: kubectl.CLI{
			Namespace:   runCtx.Opts.Namespace,
			KubeContext: runCtx.KubeContext,
			Flags:       runCtx.Cfg.Deploy.KubectlDeploy.Flags,
			ForceDeploy: runCtx.Opts.ForceDeploy(),
		},
		defaultRepo:        runCtx.DefaultRepo,
		insecureRegistries: runCtx.InsecureRegistries,
	}
}

func (k *KubectlDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "kubectl",
	}
}

type ManifestTransform func(l kubectl.ManifestList, builds []build.Artifact, insecureRegistries map[string]bool) (kubectl.ManifestList, error)

// Transforms are applied to manifests
var manifestTransforms []ManifestTransform

// AddManifestTransform adds a transform to be applied when deploying.
func AddManifestTransform(newTransform ManifestTransform) {
	manifestTransforms = append(manifestTransforms, newTransform)
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *KubectlDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact, labellers []Labeller) error {
	color.Default.Fprintln(out, "kubectl client version:", k.kubectl.Version(ctx))
	if err := k.kubectl.CheckVersion(ctx); err != nil {
		color.Default.Fprintln(out, err)
	}

	event.DeployInProgress()

	manifests, err := k.readManifests(ctx)
	if err != nil {
		event.DeployFailed(err)
		return errors.Wrap(err, "reading manifests")
	}

	if len(manifests) == 0 {
		return nil
	}

	manifests, err = manifests.ReplaceImages(builds, k.defaultRepo)
	if err != nil {
		event.DeployFailed(err)
		return errors.Wrap(err, "replacing images in manifests")
	}

	manifests, err = manifests.SetLabels(merge(labellers...))
	if err != nil {
		event.DeployFailed(err)
		return errors.Wrap(err, "setting labels in manifests")
	}

	for _, transform := range manifestTransforms {
		manifests, err = transform(manifests, builds, k.insecureRegistries)
		if err != nil {
			return errors.Wrap(err, "unable to transform manifests")
		}
	}

	err = k.kubectl.Apply(ctx, out, manifests)
	if err != nil {
		event.DeployFailed(err)
		return errors.Wrap(err, "kubectl error")
	}

	event.DeployComplete()
	return err
}

// Cleanup deletes what was deployed by calling Deploy.
func (k *KubectlDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	manifests, err := k.readManifests(ctx)
	if err != nil {
		return errors.Wrap(err, "reading manifests")
	}

	if err := k.kubectl.Delete(ctx, out, manifests); err != nil {
		return errors.Wrap(err, "delete")
	}

	return nil
}

func (k *KubectlDeployer) Dependencies() ([]string, error) {
	return k.manifestFiles(k.KubectlDeploy.Manifests)
}

func (k *KubectlDeployer) manifestFiles(manifests []string) ([]string, error) {
	list, err := util.ExpandPathsGlob(k.workingDir, manifests)
	if err != nil {
		return nil, errors.Wrap(err, "expanding kubectl manifest paths")
	}

	var filteredManifests []string
	for _, f := range list {
		if !util.IsSupportedKubernetesFormat(f) {
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
func (k *KubectlDeployer) readManifests(ctx context.Context) (kubectl.ManifestList, error) {
	manifests, err := k.Dependencies()
	if err != nil {
		return nil, errors.Wrap(err, "listing manifests")
	}

	if len(manifests) == 0 {
		return kubectl.ManifestList{}, nil
	}

	return k.kubectl.ReadManifests(ctx, manifests)
}
