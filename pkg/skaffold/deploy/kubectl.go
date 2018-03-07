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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"

	"github.com/sirupsen/logrus"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/build"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

// Slightly modified from kubectl run --dry-run
var deploymentTemplate = template.Must(template.New("deployment").Parse(`apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    run: skaffold
  name: skaffold
spec:
  replicas: 1
  selector:
    matchLabels:
      run: skaffold
  strategy: {}
  template:
    metadata:
      labels:
        run: skaffold
    spec:
      containers:
      - image: IMAGE
        name: app
{{if .Ports}}
        ports:
{{range .Ports}}
        - containerPort: {{.}}
{{end}}
{{end}}
`))

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
func (k *KubectlDeployer) Run(out io.Writer, b *build.BuildResult) (*Result, error) {
	if len(k.DeployConfig.KubectlDeploy.Manifests) == 0 {
		if len(b.Builds) != 1 {
			return nil, errors.New("must specify manifest if using more than one image")
		}
		yaml, err := generateManifest(b.Builds[0])
		if err != nil {
			return nil, errors.Wrap(err, "generating manifest")
		}
		params := map[string]build.Build{"IMAGE": b.Builds[0]}

		if err := deployManifestFile(strings.NewReader(yaml), params); err != nil {
			return nil, errors.Wrap(err, "deploying manifest")
		}
		return &Result{}, nil
	}

	for _, m := range k.DeployConfig.KubectlDeploy.Manifests {
		logrus.Debugf("Deploying path: %s parameters: %s", m.Paths, m.Parameters)
		if err := deployManifest(out, b.Builds, m); err != nil {
			return nil, errors.Wrap(err, "deploying manifests")
		}
	}

	return &Result{}, nil
}

func generateManifest(b build.Build) (string, error) {
	logrus.Info("No manifests specified. Generating a deployment.")
	dockerfilePath := filepath.Join(b.Artifact.Workspace, b.Artifact.DockerfilePath)
	r, err := os.Open(dockerfilePath)
	if err != nil {
		return "", errors.Wrap(err, "reading dockerfile")
	}
	ports, err := docker.PortsFromDockerfile(r)
	if err != nil {
		logrus.Warnf("Unable to determine port from Dockerfile: %s.", err)
	}
	var out bytes.Buffer
	if err := deploymentTemplate.Execute(&out, struct{ Ports []string }{Ports: ports}); err != nil {
		return "", err
	}
	return out.String(), nil
}

func deployManifest(out io.Writer, b []build.Build, manifest config.Manifest) error {
	params, err := JoinTagsToBuildResult(b, manifest.Parameters)
	if err != nil {
		return errors.Wrap(err, "joining template keys to image tag")
	}
	manifests, err := util.ExpandPathsGlob(manifest.Paths)
	if err != nil {
		return errors.Wrap(err, "expanding manifest paths")
	}
	logrus.Debugf("Expanded manifests %s", strings.Join(manifests, "\n"))
	for _, fname := range manifests {
		if !util.IsSupportedKubernetesFormat(fname) {
			if !util.StrSliceContains(manifest.Paths, fname) {
				logrus.Infof("Refusing to deploy non {json, yaml} file %s", fname)
				logrus.Info("If you still wish to deploy this file, please specify it directly, outside a glob pattern.")
				continue
			}
		}
		fmt.Fprintf(out, "Deploying %s...\n", fname)
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
