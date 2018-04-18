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
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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
	*v1alpha2.DeployConfig
	kubeContext string
}

// NewKubectlDeployer returns a new KubectlDeployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewKubectlDeployer(cfg *v1alpha2.DeployConfig, kubeContext string) *KubectlDeployer {
	return &KubectlDeployer{
		DeployConfig: cfg,
		kubeContext:  kubeContext,
	}
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *KubectlDeployer) Deploy(ctx context.Context, out io.Writer, b *build.BuildResult) (*Result, error) {
	if len(k.DeployConfig.KubectlDeploy.Manifests) > 0 {
		manifests, err := util.ExpandPathsGlob(k.DeployConfig.KubectlDeploy.Manifests)
		if err != nil {
			logrus.Error("expanding kubectl manifest paths", err)
		} else {
			for _, m := range manifests {
				logrus.Debugf("Deploying path: %s", m)
				if err := k.deployManifest(out, b.Builds, m); err != nil {
					return nil, errors.Wrap(err, "deploying manifests")
				}
			}
		}
	}

	if len(k.DeployConfig.KubectlDeploy.RemoteManifests) > 0 {
		for _, m := range k.DeployConfig.KubectlDeploy.RemoteManifests {
			namespace := "default" //TODO get namespace from kubectl context
			name := m
			if strings.Contains(m, ":") {
				namespace = strings.Split(m, ":")[0]
				name = strings.Split(m, ":")[1]
			}
			manifest, err := k.getRemoteManifest(out, namespace, name, b.Builds)
			if err != nil {
				return nil, errors.Wrap(err, "get remote manifests")
			}
			logrus.Debugf("Deploying Remote Manifest: %s", m)
			err = k.apply(manifest)
			if err != nil {
				return nil, errors.Wrap(err, "deploying remote manifests")
			}

		}
	}

	if len(k.DeployConfig.KubectlDeploy.Manifests) == 0 && len(k.DeployConfig.KubectlDeploy.RemoteManifests) == 0 {
		if len(b.Builds) != 1 {
			return nil, errors.New("must specify manifest if using more than one image")
		}
		yaml, err := generateManifest(b.Builds[0])
		if err != nil {
			return nil, errors.Wrap(err, "generating manifest")
		}

		if err := k.deployManifestFile(strings.NewReader(yaml), []build.Build{{ImageName: "IMAGE", Tag: b.Builds[0].Tag}}); err != nil {
			return nil, errors.Wrap(err, "deploying manifest")
		}
	}

	return &Result{}, nil
}

func (k *KubectlDeployer) getRemoteManifest(out io.Writer, namespace, name string, b []build.Build) (string, error) {
	var manifestContents bytes.Buffer
	cmd := exec.Command("kubectl", "--context", k.kubeContext, "--namespace", namespace, "get", name, "-o", "yaml")
	cmd.Stdout = &manifestContents
	cmd.Stderr = out
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, "getting manifest")
	}
	manifest, err := replaceParameters(manifestContents.Bytes(), b)
	if err != nil {
		return "", errors.Wrap(err, "replacing image in manifest")
	}
	return manifest, nil
}

func (k *KubectlDeployer) apply(manifest string) error {
	cmd := exec.Command("kubectl", "--context", k.kubeContext, "apply", "-f", "-")
	stdin := strings.NewReader(manifest)
	output, outerr, err := util.RunCommand(cmd, stdin)
	if err != nil {
		return errors.Wrapf(err, "running kubectl apply: stdout: %s stderr: %s err: %s", output, outerr, err)
	}
	return nil

}

func generateManifest(b build.Build) (string, error) {
	logrus.Info("No manifests specified. Generating a deployment.")
	dockerfilePath := filepath.Join(b.Artifact.Workspace, b.Artifact.DockerArtifact.DockerfilePath)
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

func (k *KubectlDeployer) deployManifest(out io.Writer, b []build.Build, manifest string) error {
	if !util.IsSupportedKubernetesFormat(manifest) {
		if !util.StrSliceContains(k.KubectlDeploy.Manifests, manifest) {
			logrus.Infof("Refusing to deploy non {json, yaml} file %s", manifest)
			logrus.Info("If you still wish to deploy this file, please specify it directly, outside a glob pattern.")
			return nil
		}
	}
	fmt.Fprintf(out, "Deploying %s...\n", manifest)
	f, err := util.Fs.Open(manifest)
	if err != nil {
		return errors.Wrap(err, "opening manifest")
	}

	if err := k.deployManifestFile(f, b); err != nil {
		return errors.Wrapf(err, "deploying manifest %s", manifest)
	}
	return nil
}

func (k *KubectlDeployer) deployManifestFile(r io.Reader, b []build.Build) error {
	var manifestContents bytes.Buffer
	if _, err := manifestContents.ReadFrom(r); err != nil {
		return errors.Wrap(err, "reading manifest")
	}

	manifest, err := replaceParameters(manifestContents.Bytes(), b)
	if err != nil {
		return errors.Wrap(err, "replacing image in manifest")
	}
	return k.apply(manifest)
}

type replacement struct {
	tag   string
	found bool
}

func replaceParameters(contents []byte, b []build.Build) (string, error) {
	var manifests []string

	replacements := map[string]*replacement{}
	for _, build := range b {
		replacements[build.ImageName] = &replacement{
			tag: build.Tag,
		}
	}

	parts := bytes.Split(contents, []byte("\n---"))
	for _, part := range parts {
		m := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(part, &m); err != nil {
			return "", errors.Wrap(err, "reading kubernetes YAML")
		}

		replaced := recursiveReplace(m, replacements)
		replacedMap := replaced.(map[string]interface{})

		out, err := yaml.Marshal(replacedMap)
		if err != nil {
			return "", errors.Wrap(err, "marshalling yaml")
		}

		manifests = append(manifests, string(out))
	}

	for name, replacement := range replacements {
		if !replacement.found {
			logrus.Warnf("image [%s] is not used by the deployment", name)
		}
	}

	manifest := strings.Join(manifests, "---\n")
	logrus.Debugln("Applying manifest:", manifest)

	return manifest, nil
}

func recursiveReplace(i interface{}, replacements map[string]*replacement) interface{} {
	switch t := i.(type) {
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range t {
			if k.(string) == "image" {
				name := v.(string)
				if strings.Contains(name, ":") {
					name = strings.Split(name, ":")[0]
				}
				if img, present := replacements[name]; present {
					v = img.tag
					img.found = true
				}
			}
			m[k.(string)] = recursiveReplace(v, replacements)
		}
		return m
	case []interface{}:
		for i, v := range t {
			t[i] = recursiveReplace(v, replacements)
		}
	}
	return i
}
