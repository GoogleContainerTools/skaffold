/*
Copyright 2019 The Tekton Authors.

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

package v1alpha1

import (
	b64 "encoding/base64"
	"encoding/json"
	"flag"
	"strconv"
	"strings"

	"github.com/tektoncd/pipeline/pkg/names"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
)

var (
	kubeconfigWriterImage = flag.String("kubeconfig-writer-image", "override-with-kubeconfig-writer:latest", "The container image containing our kubeconfig writer binary.")
)

// ClusterResource represents a cluster configuration (kubeconfig)
// that can be accessed by tasks in the pipeline
type ClusterResource struct {
	Name string               `json:"name"`
	Type PipelineResourceType `json:"type"`
	// URL must be a host string
	URL      string `json:"url"`
	Revision string `json:"revision"`
	// Server requires Basic authentication
	Username string `json:"username"`
	Password string `json:"password"`
	// Server requires Bearer authentication. This client will not attempt to use
	// refresh tokens for an OAuth2 flow.
	// Token overrides userame and password
	Token string `json:"token"`
	// Server should be accessed without verifying the TLS certificate. For testing only.
	Insecure bool
	// CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
	// CAData takes precedence over CAFile
	CAData []byte `json:"cadata"`
	//Secrets holds a struct to indicate a field name and corresponding secret name to populate it
	Secrets []SecretParam `json:"secrets"`
}

// NewClusterResource create a new k8s cluster resource to pass to a pipeline task
func NewClusterResource(r *PipelineResource) (*ClusterResource, error) {
	if r.Spec.Type != PipelineResourceTypeCluster {
		return nil, xerrors.Errorf("ClusterResource: Cannot create a Cluster resource from a %s Pipeline Resource", r.Spec.Type)
	}
	clusterResource := ClusterResource{
		Type: r.Spec.Type,
	}
	for _, param := range r.Spec.Params {
		switch {
		case strings.EqualFold(param.Name, "Name"):
			clusterResource.Name = param.Value
		case strings.EqualFold(param.Name, "URL"):
			clusterResource.URL = param.Value
		case strings.EqualFold(param.Name, "Revision"):
			clusterResource.Revision = param.Value
		case strings.EqualFold(param.Name, "Username"):
			clusterResource.Username = param.Value
		case strings.EqualFold(param.Name, "Password"):
			clusterResource.Password = param.Value
		case strings.EqualFold(param.Name, "Token"):
			clusterResource.Token = param.Value
		case strings.EqualFold(param.Name, "Insecure"):
			b, _ := strconv.ParseBool(param.Value)
			clusterResource.Insecure = b
		case strings.EqualFold(param.Name, "CAData"):
			if param.Value != "" {
				sDec, _ := b64.StdEncoding.DecodeString(param.Value)
				clusterResource.CAData = sDec
			}
		}
	}
	clusterResource.Secrets = r.Spec.SecretParams

	if len(clusterResource.CAData) == 0 {
		clusterResource.Insecure = true
		for _, secret := range clusterResource.Secrets {
			if strings.EqualFold(secret.FieldName, "CAData") {
				clusterResource.Insecure = false
				break
			}
		}
	}

	return &clusterResource, nil
}

// GetName returns the name of the resource
func (s ClusterResource) GetName() string {
	return s.Name
}

// GetType returns the type of the resource, in this case "cluster"
func (s ClusterResource) GetType() PipelineResourceType {
	return PipelineResourceTypeCluster
}

// GetURL returns the url to be used with this resource
func (s *ClusterResource) GetURL() string {
	return s.URL
}

// Replacements is used for template replacement on a ClusterResource inside of a Taskrun.
func (s *ClusterResource) Replacements() map[string]string {
	return map[string]string{
		"name":     s.Name,
		"type":     string(s.Type),
		"url":      s.URL,
		"revision": s.Revision,
		"username": s.Username,
		"password": s.Password,
		"token":    s.Token,
		"insecure": strconv.FormatBool(s.Insecure),
		"cadata":   string(s.CAData),
	}
}

func (s ClusterResource) String() string {
	json, _ := json.Marshal(s)
	return string(json)
}

func (s *ClusterResource) GetUploadContainerSpec() ([]corev1.Container, error) {
	return nil, nil
}

func (s *ClusterResource) SetDestinationDirectory(path string) {
}
func (s *ClusterResource) GetDownloadContainerSpec() ([]corev1.Container, error) {
	var envVars []corev1.EnvVar
	for _, sec := range s.Secrets {
		ev := corev1.EnvVar{
			Name: strings.ToUpper(sec.FieldName),
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: sec.SecretName,
					},
					Key: sec.SecretKey,
				},
			},
		}
		envVars = append(envVars, ev)
	}

	clusterContainer := corev1.Container{
		Name:    names.SimpleNameGenerator.RestrictLengthWithRandomSuffix("kubeconfig"),
		Image:   *kubeconfigWriterImage,
		Command: []string{"/ko-app/kubeconfigwriter"},
		Args: []string{
			"-clusterConfig", s.String(),
		},
		Env: envVars,
	}

	return []corev1.Container{clusterContainer}, nil
}
