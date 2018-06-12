package deploy

import (
	"bytes"
	"context"
	"io"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

type ComposeDeployer struct {
	*v1alpha2.DeployConfig
	kubeContext string
}

func NewComposeDeployer(cfg *v1alpha2.DeployConfig, kubeContext string) *ComposeDeployer {
	return &ComposeDeployer{
		DeployConfig: cfg,
		kubeContext:  kubeContext,
	}
}

func (c *ComposeDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Build) error {
	manifests, err := buildKomposeManifests()
	if err != nil {
		return errors.Wrap(err, "generating kompose manifests")
	}
	if err := applyManifests(manifests, out, c.kubeContext, builds); err != nil {
		return errors.Wrap(err, "applying manifests")
	}
	return nil
}

func buildKomposeManifests() (io.Reader, error) {
	cmd := exec.Command("kompose", "convert", "--stdout")
	out, err := util.DefaultExecCommand.RunCmdOut(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "running kustomize build")
	}
	return bytes.NewReader(out), nil
}

func (c *ComposeDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	manifests, err := buildKomposeManifests()
	if err != nil {
		return errors.Wrap(err, "generating kompose manifests")
	}
	if err := kubectl(manifests, out, c.kubeContext, "delete", "-f", "-"); err != nil {
		return errors.Wrap(err, "kubectl delete")
	}
	return nil
}

func (c *ComposeDeployer) Dependencies() ([]string, error) {
	return []string{}, nil
}
