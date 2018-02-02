package deploy

import (
	"bytes"
	"io"
	"os/exec"
	"strings"

	"github.com/spf13/afero"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

var fs = afero.NewOsFs()

type KubectlDeployer struct {
	*config.DeployConfig
}

func NewKubectlDeployer(cfg *config.DeployConfig) (*KubectlDeployer, error) {
	return &KubectlDeployer{cfg}, nil
}

func (k *KubectlDeployer) Run(b *build.BuildResult) (*Result, error) {
	params, err := JoinTagsToBuildResult(b, k.DeployConfig)
	if err != nil {
		return nil, errors.Wrap(err, "joining template keys to image tag")
	}

	for _, m := range k.DeployConfig.KubectlDeploy.Manifests {
		f, err := fs.Open(m)
		if err != nil {
			return nil, errors.Wrap(err, "opening manifest")
		}
		if err := deployManifest(f, params); err != nil {
			return nil, errors.Wrapf(err, "deploying manifest %s", m)
		}
	}

	return &Result{}, nil
}

func deployManifest(r io.Reader, params map[string]build.Build) error {
	var manifestContents bytes.Buffer
	if _, err := manifestContents.ReadFrom(r); err != nil {
		return errors.Wrap(err, "reading manifest")
	}
	manifest := manifestContents.String()
	for old, new := range params {
		manifest = strings.Replace(manifest, old, new.Tag, -1)
	}
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	stdin := strings.NewReader(manifest)
	out, outerr, err := util.RunCommand(cmd, stdin)
	if err != nil {
		return errors.Wrapf(err, "running kubectl apply: stdout: %s stderr: %s err: %s", out, outerr, err)
	}
	return nil
}
