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
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

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

func getService(svc *v1.Service) (*v1.Service, error) {
	client, err := Client()
	if err != nil {
		return nil, errors.Wrap(err, "getting Kubernetes client")
	}

	return client.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
}
