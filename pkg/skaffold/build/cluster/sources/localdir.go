/*
Copyright 2019 The Skaffold Authors

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
	"path/filepath"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sources"
)

const (
	initContainer = "kaniko-init-container"
)

// LocalDir refers to kaniko using a local directory as a buildcontext
// skaffold copies the buildcontext into the local directory via kubectl cp
type LocalDir struct {
	artifact       *latest.KanikoArtifact
	clusterDetails *latest.ClusterDetails
	kubectl        *kubectl.CLI
	tarPath        string
}

// Setup for LocalDir creates a tarball of the buildcontext and stores it in /tmp
func (g *LocalDir) Setup(ctx context.Context, out io.Writer, artifact *latest.Artifact, initialTag string, dependencies []string) (string, error) {
	g.tarPath = filepath.Join(os.TempDir(), fmt.Sprintf("context-%s.tar.gz", initialTag))
	color.Default.Fprintln(out, "Storing build context at", g.tarPath)

	f, err := os.Create(g.tarPath)
	if err != nil {
		return "", errors.Wrap(err, "creating temporary buildcontext tarball")
	}
	defer f.Close()

	err = sources.TarGz(ctx, f, artifact, dependencies)

	context := fmt.Sprintf("dir://%s", constants.DefaultKanikoEmptyDirMountPath)
	return context, err
}

// Pod returns the pod template to ModifyPod
func (g *LocalDir) Pod(args []string) *v1.Pod {
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
		Name:         initContainer,
		Image:        g.artifact.BuildContext.LocalDir.InitImage,
		Command:      []string{"sh", "-c", "while [ ! -f /tmp/complete ]; do sleep 1; done"},
		VolumeMounts: []v1.VolumeMount{vm},
		Resources:    resourceRequirements(g.clusterDetails.Resources),
	}

	p := podTemplate(g.clusterDetails, g.artifact, args, version.Get().Version)
	p.Spec.InitContainers = []v1.Container{ic}
	p.Spec.Containers[0].VolumeMounts = append(p.Spec.Containers[0].VolumeMounts, vm)
	p.Spec.Volumes = append(p.Spec.Volumes, v)
	return p
}

// ModifyPod first copies over the buildcontext tarball into the init container tmp dir via kubectl cp
// Via kubectl exec, we extract the tarball to the empty dir
// Then, via kubectl exec, create the /tmp/complete file via kubectl exec to complete the init container
func (g *LocalDir) ModifyPod(ctx context.Context, p *v1.Pod) error {
	client, err := kubernetes.Client()
	if err != nil {
		return errors.Wrap(err, "getting kubernetes client")
	}

	if err := kubernetes.WaitForPodInitialized(ctx, client.CoreV1().Pods(p.Namespace), p.Name); err != nil {
		return errors.Wrap(err, "waiting for pod to initialize")
	}

	f, err := os.Open(g.tarPath)
	if err != nil {
		return errors.Wrap(err, "opening context tar")
	}
	defer f.Close()

	// Copy the context to the empty dir and extract it
	err = g.kubectl.Run(ctx, f, nil, "exec", "-i", p.Name, "-c", initContainer, "-n", p.Namespace, "--", "tar", "-xzf", "-", "-C", constants.DefaultKanikoEmptyDirMountPath)
	if err != nil {
		return errors.Wrap(err, "copying and extracting buildcontext to empty dir")
	}
	// Generate a file to successfully terminate the init container
	return g.kubectl.Run(ctx, nil, nil, "exec", p.Name, "-c", initContainer, "-n", p.Namespace, "--", "touch", "/tmp/complete")
}

// Cleanup deletes the buildcontext tarball stored on the local filesystem
func (g *LocalDir) Cleanup(ctx context.Context) error {
	return os.Remove(g.tarPath)
}
