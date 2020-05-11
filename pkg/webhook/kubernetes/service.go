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
	"fmt"
	"time"

	"github.com/google/go-github/github"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/labels"
)

// CreateService creates a service for the deployment to bind to
// and returns the external IP of the service
func CreateService(pr *github.PullRequestEvent) (*v1.Service, error) {
	client, err := kubernetes.Client()
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	l := labels.GenerateLabelsFromPR(pr.GetNumber())
	key, val := labels.RetrieveLabel(pr.GetNumber())
	selector := map[string]string{key: val}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   serviceName(pr.GetNumber()),
			Labels: l,
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeLoadBalancer,
			Ports: []v1.ServicePort{
				{
					Port: constants.HugoPort,
				},
			},
			Selector: selector,
		},
	}
	return client.CoreV1().Services(constants.Namespace).Create(svc)
}

// GetExternalIP polls the service until an external IP is available and returns it
func GetExternalIP(s *v1.Service) (string, error) {
	var ip string
	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (bool, error) {
		svc, err := getService(s)
		if err != nil {
			return false, nil
		}
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			ip = svc.Status.LoadBalancer.Ingress[0].IP
			return true, nil
		}
		return false, nil
	})
	return ip, err
}

func serviceName(prNumber int) string {
	return fmt.Sprintf("docs-controller-svc-%d", prNumber)
}

func getService(svc *v1.Service) (*v1.Service, error) {
	client, err := kubernetes.Client()
	if err != nil {
		return nil, fmt.Errorf("getting Kubernetes client: %w", err)
	}

	return client.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
}
