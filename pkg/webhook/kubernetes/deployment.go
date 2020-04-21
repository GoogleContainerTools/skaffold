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

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path"
	"time"

	"github.com/google/go-github/github"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/labels"
)

const (
	initContainerName = "git-clone"
	emptyVol          = "empty-vol"
	emptyVolPath      = "/empty"
)

// CreateDeployment creates a deployment for this pull request
// The deployment has two containers:
// 		1. An init container to git clone the PR branch
// 		2. A container to run hugo server
// and one emptyDir volume to hold the git repository
func CreateDeployment(pr *github.PullRequestEvent, svc *v1.Service, externalIP string) (*appsv1.Deployment, error) {
	client, err := pkgkubernetes.Client()
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	deploymentLabels := svc.Spec.Selector
	_, name := labels.RetrieveLabel(pr.GetNumber())

	userRepo := fmt.Sprintf("https://github.com/%s.git", *pr.PullRequest.Head.Repo.FullName)
	// path to the docs directory, which we will run "hugo server -D" in
	repoPath := path.Join(emptyVolPath, *pr.PullRequest.Head.Repo.Name)

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
							Name:       initContainerName,
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
							Args:       []string{"deploy/docs/preview.sh", BaseURL(externalIP)},
							WorkingDir: repoPath,
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
	return client.AppsV1().Deployments(constants.Namespace).Create(d)
}

// WaitForDeploymentToStabilize waits till the Deployment has stabilized
func WaitForDeploymentToStabilize(d *appsv1.Deployment, ip string) error {
	client, err := pkgkubernetes.Client()
	if err != nil {
		return fmt.Errorf("getting Kubernetes client: %w", err)
	}

	if err := pkgkubernetes.WaitForDeploymentToStabilize(context.Background(), client, d.Namespace, d.Name, 5*time.Minute); err != nil {
		return fmt.Errorf("waiting for deployment to stabilize: %w", err)
	}

	// wait up to five minutes for the URL to return a valid endpoint
	url := BaseURL(ip)
	log.Printf("Waiting up to 2 minutes for %s to return an OK response...", url)
	return wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		resp, err := http.Get(url)
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK, nil
	})
}

// BaseURL returns the base url of the deployment
func BaseURL(ip string) string {
	return fmt.Sprintf("http://%s:%d", ip, constants.HugoPort)
}

// Logs returns the logs for both containers for the given deployment
func Logs(d *appsv1.Deployment) string {
	deploy := fmt.Sprintf("deployment/%s", d.Name)
	// get init container logs
	cmd := exec.Command("kubectl", "logs", deploy, "-c", initContainerName)
	initLogs, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error retrieving init container logs for %s: %v", d.Name, err)
	}
	// get deployment logs
	cmd = exec.Command("kubectl", "logs", deploy)
	logs, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error retrieving deployment logs for %s: %v", d.Name, err)
	}
	return fmt.Sprintf("Init container logs: \n %s \nContainer Logs: \n %s", initLogs, logs)
}
