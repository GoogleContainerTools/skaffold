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
	"os/exec"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// for testing
var warner Warner = &logrusWarner{}

// KubectlDeployer deploys workflows using kubectl CLI.
type KubectlDeployer struct {
	*v1alpha2.KubectlDeploy

	workingDir  string
	kubeContext string
}

// NewKubectlDeployer returns a new KubectlDeployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewKubectlDeployer(workingDir string, cfg *v1alpha2.KubectlDeploy, kubeContext string) *KubectlDeployer {
	return &KubectlDeployer{
		KubectlDeploy: cfg,
		workingDir:    workingDir,
		kubeContext:   kubeContext,
	}
}

func (k *KubectlDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "kubectl",
	}
}

func (k *KubectlDeployer) DeployInfo() status.DeployerInfo {
	files, err := k.manifestFiles(k.Manifests)
	if err != nil {
		logrus.Warn(err.Error())
	}

	return status.DeployerInfo{
		Name:                "Kubectl",
		WorkingDir:          k.workingDir,
		KubeContext:         k.kubeContext,
		ManifestPaths:       files,
		RemoteManifestPaths: k.RemoteManifests,
	}
}

func (k *KubectlDeployer) processManifestsForDeploy(builds []build.Artifact) (manifestList, error) {
	manifests, err := k.readManifests()
	if err != nil {
		return nil, errors.Wrap(err, "reading manifests")
	}

	manifests, err = manifests.replaceImages(builds)
	if err != nil {
		return nil, errors.Wrap(err, "replacing images in manifests")
	}
	return manifests, nil
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *KubectlDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]Artifact, error) {
	manifests, err := k.processManifestsForDeploy(builds)
	if err != nil {
		return nil, err
	}

	err = kubectl(manifests.reader(), out, k.kubeContext, k.Flags.Global, "apply", k.Flags.Apply, "-f", "-")
	if err != nil {
		return nil, errors.Wrap(err, "deploying manifests")
	}

	return parseManifestsForDeploys(manifests)
}

// Cleanup deletes what was deployed by calling Deploy.
func (k *KubectlDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	manifests, err := k.readManifests()
	if err != nil {
		return errors.Wrap(err, "reading manifests")
	}

	if err := kubectl(manifests.reader(), out, k.kubeContext, k.Flags.Global, "delete", k.Flags.Delete, "--grace-period=1", "--ignore-not-found=true", "-f", "-"); err != nil {
		return errors.Wrap(err, "deleting manifests")
	}

	return nil
}

func (k *KubectlDeployer) Dependencies() ([]string, error) {
	return k.manifestFiles(k.KubectlDeploy.Manifests)
}

func kubectl(in io.Reader, out io.Writer, kubeContext string, globalFlags []string, command string, commandFlags []string, arg ...string) error {
	args := []string{"--context", kubeContext}
	args = append(args, globalFlags...)
	args = append(args, command)
	args = append(args, commandFlags...)
	args = append(args, arg...)

	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(cmd)
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

func parseManifestsForDeploys(manifests manifestList) ([]Artifact, error) {
	results := []Artifact{}
	for _, manifest := range manifests {
		b := bufio.NewReader(bytes.NewReader(manifest))
		results = append(results, parseReleaseInfo("", b)...)
	}
	return results, nil
}

// readManifests reads the manifests to deploy/delete.
func (k *KubectlDeployer) readManifests() (manifestList, error) {
	files, err := k.manifestFiles(k.Manifests)
	if err != nil {
		return nil, errors.Wrap(err, "expanding user manifest list")
	}
	var manifests manifestList

	for _, manifest := range files {
		buf, err := ioutil.ReadFile(manifest)
		if err != nil {
			return nil, errors.Wrap(err, "reading manifest")
		}

		parts := bytes.Split(buf, []byte("\n---"))
		for _, part := range parts {
			manifests = append(manifests, part)
		}
	}

	for _, m := range k.RemoteManifests {
		manifest, err := k.readRemoteManifest(m)
		if err != nil {
			return nil, errors.Wrap(err, "get remote manifests")
		}

		manifests = append(manifests, manifest)
	}

	logrus.Debugln("manifests", manifests.String())

	return manifests, nil
}

func (k *KubectlDeployer) readRemoteManifest(name string) ([]byte, error) {
	var args []string
	if parts := strings.Split(name, ":"); len(parts) > 1 {
		args = append(args, "--namespace", parts[0])
		name = parts[1]
	}
	args = append(args, name, "-o", "yaml")

	var manifest bytes.Buffer
	err := kubectl(nil, &manifest, k.kubeContext, k.Flags.Global, "get", nil, args...)
	if err != nil {
		return nil, errors.Wrap(err, "getting manifest")
	}

	return manifest.Bytes(), nil
}

type replacement struct {
	tag   string
	found bool
}

type manifestList [][]byte

func (l *manifestList) String() string {
	var str string
	for i, manifest := range *l {
		if i != 0 {
			str += "\n---\n"
		}
		str += string(bytes.TrimSpace(manifest))
	}
	return str
}

func (l *manifestList) reader() io.Reader {
	return strings.NewReader(l.String())
}

func (l *manifestList) replaceImages(builds []build.Artifact) (manifestList, error) {
	replacements := map[string]*replacement{}
	for _, build := range builds {
		replacements[build.ImageName] = &replacement{
			tag: build.Tag,
		}
	}

	var updatedManifests manifestList

	for _, manifest := range *l {
		m := make(map[interface{}]interface{})
		if err := yaml.Unmarshal(manifest, &m); err != nil {
			return nil, errors.Wrap(err, "reading kubernetes YAML")
		}

		if len(m) == 0 {
			continue
		}

		recursiveReplaceImage(m, replacements)

		updatedManifest, err := yaml.Marshal(m)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling yaml")
		}

		updatedManifests = append(updatedManifests, updatedManifest)
	}

	for name, replacement := range replacements {
		if !replacement.found {
			warner.Warnf("image [%s] is not used by the deployment", name)
		}
	}

	logrus.Debugln("manifests with tagged images", updatedManifests.String())

	return updatedManifests, nil
}

func recursiveReplaceImage(i interface{}, replacements map[string]*replacement) {
	switch t := i.(type) {
	case []interface{}:
		for _, v := range t {
			recursiveReplaceImage(v, replacements)
		}
	case map[interface{}]interface{}:
		for k, v := range t {
			if k.(string) != "image" {
				recursiveReplaceImage(v, replacements)
				continue
			}

			image := v.(string)
			parsed, err := docker.ParseReference(image)
			if err != nil {
				warner.Warnf("Couldn't parse image: %s", v)
				continue
			}

			if img, present := replacements[parsed.BaseName]; present {
				if parsed.FullyQualified {
					if img.tag == image {
						img.found = true
					}
				} else {
					t[k] = img.tag
					img.found = true
				}
			}
		}
	}
}
