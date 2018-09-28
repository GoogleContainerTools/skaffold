/*
Copyright 2018 The Skaffold Authors

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
	"bufio"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	latest "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha4"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// KubectlDeployer deploys workflows using kubectl CLI.
type KubectlDeployer struct {
	*latest.KubectlDeploy

	workingDir string
	kubectl    kubectl.CLI
}

// NewKubectlDeployer returns a new KubectlDeployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewKubectlDeployer(workingDir string, cfg *latest.KubectlDeploy, kubeContext string, namespace string) *KubectlDeployer {
	return &KubectlDeployer{
		KubectlDeploy: cfg,
		workingDir:    workingDir,
		kubectl: kubectl.CLI{
			Namespace:   namespace,
			KubeContext: kubeContext,
			Flags:       cfg.Flags,
		},
	}
}

func (k *KubectlDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "kubectl",
	}
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *KubectlDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]Artifact, error) {
	color.Default.Fprintln(out, "kubectl client version:", k.kubectl.Version())

	manifests, err := k.readManifests(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "reading manifests")
	}

	if len(manifests) == 0 {
		return nil, nil
	}

	manifests, err = manifests.ReplaceImages(builds)
	if err != nil {
		return nil, errors.Wrap(err, "replacing images in manifests")
	}

	updated, err := k.kubectl.Apply(ctx, out, manifests)
	if err != nil {
		return nil, errors.Wrap(err, "apply")
	}

	return parseManifestsForDeploys(updated)
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

func parseManifestsForDeploys(manifests kubectl.ManifestList) ([]Artifact, error) {
	results := []Artifact{}
	for _, manifest := range manifests {
		b := bufio.NewReader(bytes.NewReader(manifest))
		results = append(results, parseReleaseInfo("", b)...)
	}
	return results, nil
}

// readManifests reads the manifests to deploy/delete.
func (k *KubectlDeployer) readManifests(ctx context.Context) (kubectl.ManifestList, error) {
	files, err := k.manifestFiles(k.Manifests)
	if err != nil {
		return nil, errors.Wrap(err, "expanding user manifest list")
	}

	var manifests kubectl.ManifestList
	for _, manifest := range files {
		buf, err := ioutil.ReadFile(manifest)
		if err != nil {
			return nil, errors.Wrap(err, "reading manifest")
		}

		manifests.Append(buf)
	}

	for _, m := range k.RemoteManifests {
		manifest, err := k.readRemoteManifest(ctx, m)
		if err != nil {
			return nil, errors.Wrap(err, "get remote manifests")
		}

		manifests = append(manifests, manifest)
	}

	logrus.Debugln("manifests", manifests.String())

	return manifests, nil
}

func (k *KubectlDeployer) readRemoteManifest(ctx context.Context, name string) ([]byte, error) {
	var args []string
	if parts := strings.Split(name, ":"); len(parts) > 1 {
		args = append(args, "--namespace", parts[0])
		name = parts[1]
	}
	args = append(args, name, "-o", "yaml")

	var manifest bytes.Buffer
	err := k.kubectl.Run(ctx, nil, &manifest, "get", nil, args...)
	if err != nil {
		return nil, errors.Wrap(err, "getting manifest")
	}

	return manifest.Bytes(), nil
}
