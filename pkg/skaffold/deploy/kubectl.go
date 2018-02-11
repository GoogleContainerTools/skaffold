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
	for _, m := range k.DeployConfig.KubectlDeploy.Manifests {
		logrus.Debugf("Deploying path: %s parameters: %s", m.Path, m.Parameters)
		if err := deployManifest(b.Builds, m); err != nil {
			return nil, errors.Wrap(err, "deploying manifests")
		}
	}

	return &Result{}, nil
}

func deployManifest(b []build.Build, manifest config.Manifest) error {
	params, err := JoinTagsToBuildResult(b, manifest.Parameters)
	if err != nil {
		return errors.Wrap(err, "joining template keys to image tag")
	}
	manifests, err := util.ExpandPathsGlob(manifest.Path)
	if err != nil {
		return errors.Wrap(err, "expanding manifest paths")
	}
	logrus.Debugf("Expanded manifests %s", strings.Join(manifests, "\n"))
	for _, fname := range manifests {
		if !strings.HasSuffix(fname, ".yml") && !strings.HasSuffix(fname, ".yaml") {
			logrus.Debugf("Refusing to deploy non yaml file %s", fname)
			continue
		}
		logrus.Infof("Deploying %s", fname)
		f, err := util.Fs.Open(fname)
		if err != nil {
			return errors.Wrap(err, "opening manifest")
		}
		if err := deployManifestFile(f, params); err != nil {
			return errors.Wrapf(err, "deploying manifest %s", fname)
		}
	}
	return nil
}

func deployManifestFile(r io.Reader, params map[string]build.Build) error {
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
