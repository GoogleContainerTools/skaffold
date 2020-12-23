/*
Copyright 2020 The Skaffold Authors

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

package kubectl

const (
	KubectlVersion112 = `{"clientVersion":{"major":"1","minor":"12"}}`
	KubectlVersion118 = `{"clientVersion":{"major":"1","minor":"18"}}`
)

var TestKubeConfig = "kubeconfig"
var TestKubeContext = "kubecontext"
var TestNamespace = "testNamespace"
var TestNamespace2 = "testNamespace2"
var TestNamespace2FromEnvTemplate = "test{{.MYENV}}ace2" // needs `MYENV=Namesp` environment variable

const DeploymentWebYAML = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - name: leeroy-web
    image: leeroy-web`

const DeploymentWebYAMLv1 = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-web
spec:
  containers:
  - image: leeroy-web:v1
    name: leeroy-web`

const DeploymentAppYAML = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - name: leeroy-app
    image: leeroy-app`

const DeploymentAppYAMLv1 = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - image: leeroy-app:v1
    name: leeroy-app`

const DeploymentAppYAMLv2 = `apiVersion: v1
kind: Pod
metadata:
  name: leeroy-app
spec:
  containers:
  - image: leeroy-app:v2
    name: leeroy-app`
