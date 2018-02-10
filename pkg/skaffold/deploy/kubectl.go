/*
Copyright 2018 Google LLC

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
	"io"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

type KubectlDeployer struct {
	*config.DeployConfig
}

// NewKubectlDeployer returns a new KubectlDeployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewKubectlDeployer(cfg *config.DeployConfig) (*KubectlDeployer, error) {
	return &KubectlDeployer{cfg}, nil
}

// Run templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *KubectlDeployer) Run(b *build.BuildResult) (*Result, error) {
	params, err := JoinTagsToBuildResult(b, k.DeployConfig)
	if err != nil {
		return nil, errors.Wrap(err, "joining template keys to image tag")
	}

	// The manifests should all be relative to the path of the config
	manifests, err := util.ExpandPaths(".", k.DeployConfig.KubectlDeploy.Manifests)
	if err != nil {
		return nil, errors.Wrap(err, "expanding manifest paths")
	}
	logrus.Debugf("Expanded manifests %s", strings.Join(manifests, "\n"))
	for _, m := range manifests {
		if !strings.HasSuffix(m, ".yml") && !strings.HasSuffix(m, ".yaml") {
			logrus.Debugf("Refusing to deploy non yaml file %s", m)
			continue
		}
		logrus.Infof("Deploying %s", m)
		f, err := util.Fs.Open(m)
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
