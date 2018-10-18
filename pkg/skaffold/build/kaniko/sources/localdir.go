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

package sources

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const (
	initContainer = "kaniko-init-container"
)

// LocalDir refers to kaniko using a local directory as a buildcontext
// skaffold copies the buildcontext into the local directory via kubectl cp
type LocalDir struct {
	cfg     *latest.KanikoBuild
	tarPath string
}

// Setup for LocalDir creates a tarball of the buildcontext and stores it in /tmp
func (g *LocalDir) Setup(ctx context.Context, out io.Writer, artifact *latest.Artifact, initialTag string) (string, error) {
	g.tarPath = filepath.Join("/tmp", fmt.Sprintf("context-%s.tar.gz", initialTag))
	color.Default.Fprintln(out, "Storing build context at", g.tarPath)

	f, err := os.Create(g.tarPath)
	if err != nil {
		return "", errors.Wrap(err, "creating temporary buildcontext tarball")
	}
	defer f.Close()

	err = docker.CreateDockerTarGzContext(ctx, f, artifact.Workspace, artifact.DockerArtifact)

	context := fmt.Sprintf("dir://%s", constants.DefaultKanikoEmptyDirMountPath)
	return context, err
}

// Pod returns the pod template to ModifyPod
func (g *LocalDir) Pod(args []string) *v1.Pod {
	p := podTemplate(g.cfg, args)
	// Include the emptyDir volume and volume source in both containers
	v := v1.Volume{
		Name: constants.DefaultKanikoEmptyDirName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	}
	vm := v1.VolumeMount{
		Name:      constants.DefaultKanikoEmptyDirName,
		MountPath: constants.DefaultKanikoEmptyDirMountPath,
	}
	// Generate the init container, which will run until the /tmp/complete file is created
	ic := v1.Container{
		Name:  initContainer,
		Image: constants.DefaultAlpineImage,
		Args: []string{"sh", "-c", `while true; do
	sleep 1; if [ -f /tmp/complete ]; then break; fi
done`},
		VolumeMounts: []v1.VolumeMount{vm},
	}

	p.Spec.InitContainers = []v1.Container{ic}
	p.Spec.Containers[0].VolumeMounts = append(p.Spec.Containers[0].VolumeMounts, vm)
	p.Spec.Volumes = append(p.Spec.Volumes, v)
	return p
}

// ModifyPod first copies over the buildcontext tarball into the init container tmp dir via kubectl cp
// Via kubectl exec, we extract the tarball to the empty dir
// Then, via kubectl exec, create the /tmp/complete file via kubectl exec to complete the init container
func (g *LocalDir) ModifyPod(ctx context.Context, p *v1.Pod) error {
	client, err := kubernetes.GetClientset()
	if err != nil {
		return errors.Wrap(err, "getting clientset")
	}
	if err := kubernetes.WaitForPodInitialized(ctx, client.CoreV1().Pods(p.Namespace), p.Name); err != nil {
		return errors.Wrap(err, "waiting for pod to initialize")
	}
	// Copy over the buildcontext tarball into the init container
	copy := exec.CommandContext(ctx, "kubectl", "cp", g.tarPath, fmt.Sprintf("%s:/%s", p.Name, g.tarPath), "-c", initContainer, "-n", p.Namespace)
	if err := util.RunCmd(copy); err != nil {
		return errors.Wrap(err, "copying buildcontext into init container")
	}
	// Next, extract the buildcontext to the empty dir
	extract := exec.CommandContext(ctx, "kubectl", "exec", p.Name, "-c", initContainer, "-n", p.Namespace, "--", "tar", "-xzf", g.tarPath, "-C", constants.DefaultKanikoEmptyDirMountPath)
	if err := util.RunCmd(extract); err != nil {
		return errors.Wrap(err, "extracting buildcontext to empty dir")
	}
	// Generate a file to successfully terminate the init container
	file := exec.CommandContext(ctx, "kubectl", "exec", p.Name, "-c", initContainer, "-n", p.Namespace, "--", "touch", "/tmp/complete")
	return util.RunCmd(file)
}

// Cleanup deletes the buidcontext tarball stored on the local filesystem
func (g *LocalDir) Cleanup(ctx context.Context) error {
	return os.Remove(g.tarPath)
}
