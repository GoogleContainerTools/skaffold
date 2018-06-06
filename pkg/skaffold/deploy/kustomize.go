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
	*v1alpha2.DeployConfig
	kubeContext string
}

func NewKustomizeDeployer(cfg *v1alpha2.DeployConfig, kubeContext string) *KustomizeDeployer {
	return &KustomizeDeployer{
		DeployConfig: cfg,
		kubeContext:  kubeContext,
	}
}

func (k *KustomizeDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Build) error {
	manifests, err := buildManifests(constants.DefaultKustomizationPath)
	if err != nil {
		return errors.Wrap(err, "kustomize")
	}
	manifestList, err := newManifestList(manifests)
	if err != nil {
		return errors.Wrap(err, "getting manifest list")
	}
	manifestList, err = manifestList.replaceImages(builds)
	if err != nil {
		return errors.Wrap(err, "replacing images")
	}
	if err := kubectl(manifestList.reader(), out, k.kubeContext, "apply", "-f", "-"); err != nil {
		return errors.Wrap(err, "running kubectl")
	}
	return nil
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
	manifests, err := buildManifests(constants.DefaultKustomizationPath)
	if err != nil {
		return errors.Wrap(err, "kustomize")
	}
	if err := kubectl(manifests, out, k.kubeContext, "delete", "-f", "-"); err != nil {
		return errors.Wrap(err, "kubectl delete")
	}
	return nil
}

func (k *KustomizeDeployer) Dependencies() ([]string, error) {
	// TODO(r2d4): parse kustomization yaml and add base and patches as dependencies
	return []string{constants.DefaultKustomizationPath}, nil
}

func buildManifests(kustomization string) (io.Reader, error) {
	cmd := exec.Command("kustomize", "build", kustomization)
	out, err := util.DefaultExecCommand.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "running kustomize build")
	}
	return bytes.NewReader(out), nil
}
