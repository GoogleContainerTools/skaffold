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
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

type KustomizeDeployer struct {
	*v1alpha2.KustomizeDeploy

	kubeContext string
	namespace   string
}

func NewKustomizeDeployer(cfg *v1alpha2.KustomizeDeploy, kubeContext string, namespace string) *KustomizeDeployer {
	return &KustomizeDeployer{
		KustomizeDeploy: cfg,
		kubeContext:     kubeContext,
		namespace:       namespace,
	}
}

func (k *KustomizeDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "kustomize",
	}
}

func (k *KustomizeDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]Artifact, error) {
	manifests, err := buildManifests(k.KustomizePath)
	if err != nil {
		return nil, errors.Wrap(err, "kustomize")
	}
	manifestList, err := newManifestList(manifests)
	if err != nil {
		return nil, errors.Wrap(err, "getting manifest list")
	}
	manifestList, err = manifestList.replaceImages(builds)
	if err != nil {
		return nil, errors.Wrap(err, "replacing images")
	}
	if err := k.kubectl(manifestList.reader(), out, "apply", k.Flags.Apply, "-f", "-"); err != nil {
		return nil, errors.Wrap(err, "running kubectl")
	}
	return parseManifestsForDeploys(manifestList)
}

func newManifestList(r io.Reader) (manifestList, error) {
	var manifests manifestList
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "reading manifests")
	}

	parts := bytes.Split(buf, []byte("\n---"))
	for _, part := range parts {
		manifests = append(manifests, part)
	}

	return manifests, nil
}

func (k *KustomizeDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	manifests, err := buildManifests(k.KustomizePath)
	if err != nil {
		return errors.Wrap(err, "kustomize")
	}
	if err := k.kubectl(manifests, out, "delete", k.Flags.Delete, "-f", "-"); err != nil {
		return errors.Wrap(err, "kubectl delete")
	}
	return nil
}

func (k *KustomizeDeployer) Dependencies() ([]string, error) {
	// TODO(r2d4): parse kustomization yaml and add base and patches as dependencies
	return []string{k.KustomizePath}, nil
}

func buildManifests(kustomization string) (io.Reader, error) {
	cmd := exec.Command("kustomize", "build", kustomization)
	out, err := util.DefaultExecCommand.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "running kustomize build")
	}
	return bytes.NewReader(out), nil
}

// TODO(dgageot): this code is already in KubectlDeployer
func (k *KustomizeDeployer) kubectl(in io.Reader, out io.Writer, command string, commandFlags []string, arg ...string) error {
	args := []string{"--context", k.kubeContext}
	if k.namespace != "" {
		args = append(args, "--namespace", k.namespace)
	}
	args = append(args, k.Flags.Global...)
	args = append(args, command)
	args = append(args, commandFlags...)
	args = append(args, arg...)

	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(cmd)
}
