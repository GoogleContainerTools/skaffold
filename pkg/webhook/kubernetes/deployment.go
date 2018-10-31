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

package kubernetes

import (
	"context"
	"fmt"
	"path"
	"time"

	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/labels"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	emptyVol     = "empty-vol"
	emptyVolPath = "/empty"
)

// CreateDeployment creates a deployment for this pull request
// The deployment has two containers:
// 		1. An init container to git clone the PR branch
// 		2. A container to run hugo server
// and one emptyDir volume to hold the git repository
func CreateDeployment(pr *github.PullRequestEvent, svc *v1.Service, externalIP string) (*appsv1.Deployment, error) {
	clientset, err := pkgkubernetes.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "getting clientset")
	}

	deploymentLabels := svc.Spec.Selector
	_, name := labels.RetrieveLabel(pr.GetNumber())

	userRepo := fmt.Sprintf("https://github.com/%s.git", *pr.PullRequest.Head.Repo.FullName)
	// path to the docs directory, which we will run "hugo server -D" in
	docsPath := path.Join(emptyVolPath, *pr.PullRequest.Head.Repo.Name, "docs")

	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: deploymentLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: deploymentLabels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: deploymentLabels,
				},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						{
							Name:       "git-clone",
							Image:      constants.DeploymentImage,
							Args:       []string{"git", "clone", userRepo, "--branch", pr.PullRequest.Head.GetRef()},
							WorkingDir: emptyVolPath,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      emptyVol,
									MountPath: emptyVolPath,
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:       "server",
							Image:      constants.DeploymentImage,
							Args:       []string{"hugo", "server", "--bind=0.0.0.0", "-D", "--baseURL", baseURL(externalIP)},
							WorkingDir: docsPath,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      emptyVol,
									MountPath: emptyVolPath,
								},
							},
							Ports: []v1.ContainerPort{
								{
									ContainerPort: constants.HugoPort,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: emptyVol,
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
	return clientset.AppsV1().Deployments(constants.Namespace).Create(d)
}

// WaitForDeploymentToStabilize waits till the Deployment has stabilized
func WaitForDeploymentToStabilize(d *appsv1.Deployment) error {
	client, err := pkgkubernetes.GetClientset()
	if err != nil {
		return errors.Wrap(err, "getting clientset")
	}
	return pkgkubernetes.WaitForDeploymentToStabilize(context.Background(), client, d.Namespace, d.Name, 5*time.Minute)
}

func baseURL(ip string) string {
	return fmt.Sprintf("http://%s:%d", ip, constants.HugoPort)
}
