package deploy

import (
	"fmt"
	"os/exec"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type HelmDeployer struct {
	*config.DeployConfig
}

func NewHelmDeployer(cfg *config.DeployConfig) (*HelmDeployer, error) {
	return &HelmDeployer{cfg}, nil
}

func (h *HelmDeployer) Run(b *build.BuildResult) (*Result, error) {
	for _, r := range h.HelmDeploy.Releases {
		if err := deployRelease(r, b); err != nil {
			return nil, errors.Wrapf(err, "deploying %s", r.Name)
		}
	}
	return nil, nil
}

func deployRelease(r config.HelmRelease, b *build.BuildResult) error {
	isInstalled := true
	getCmd := exec.Command("helm", "get", r.Name)
	if out, stderr, err := util.RunCommand(getCmd, nil); err != nil {
		logrus.Debugf("Error getting release %s: %s stdout: %s stderr: %s", r.Name, err, string(out), string(stderr))
		logrus.Infof("Helm release %s not installed. Installing...", r.Name)
		isInstalled = false
	}
	fmt.Println("here")
	params, err := JoinTagsToBuildResult(b.Builds, r.Values)
	if err != nil {
		return errors.Wrap(err, "matching build results to chart values")
	}

	var setOpts []string
	for k, v := range params {
		setOpts = append(setOpts, "--set")
		setOpts = append(setOpts, fmt.Sprintf("%s=%s", k, v.Tag))
	}

	var args []string
	if !isInstalled {
		args = []string{"install", "--name", r.Name, r.ChartPath}
	} else {
		args = []string{"upgrade", r.Name, r.ChartPath}
	}

	args = append(args, setOpts...)
	out, stderr, err := util.RunCommand(exec.Command("helm", args...), nil)
	if err != nil {
		return errors.Wrapf(err, "helm updater stdout: %s, stderr: %s", string(out), string(stderr))
	}
	logrus.Infof("Helm: %s", string(out))
	return nil
}
