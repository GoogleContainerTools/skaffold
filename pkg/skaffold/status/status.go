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

package status

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
)

const DefaultTemplate = `
--- Skaffold Status ---{{if .ConfigInfo.Profiles}}
Applied Profiles: {{range $prof := .ConfigInfo.Profiles}}{{$prof}} {{end}}{{end}}

-- Builder
Type: {{.BuilderInfo.Name}}{{if .BuilderInfo.ProjectID}}
Project ID: {{.BuilderInfo.ProjectID}}{{end}}{{if .BuilderInfo.Namespace}}
Namespace: {{.BuilderInfo.Namespace}}{{end}}{{if .BuilderInfo.KubeContext}}
KubeContext: {{.BuilderInfo.KubeContext}}{{end}}{{if .BuilderInfo.GCSBucket}}
GCS Bucket: {{.BuilderInfo.GCSBucket}}{{end}}{{if .BuilderInfo.PullSecretName}}
Pull Secret Name: {{.BuilderInfo.PullSecretName}}{{end}}{{if .BuilderInfo.PullSecret}}
Pull Secret: {{.BuilderInfo.PullSecret}}{{end}}

-- Tagger
Type: {{.TaggerInfo.Name}}{{if .TaggerInfo.Tag}}
Tag: {{.TaggerInfo.Tag}}{{end}}{{if .TaggerInfo.Template}}
Template: {{.TaggerInfo.Template}}{{end}}

-- Deployer
Type: {{.DeployerInfo.Name}}{{if .DeployerInfo.KustomizePath}}
Kustomize Path: {{.DeployerInfo.KustomizePath}}{{end}}{{if .DeployerInfo.Namespace}}
Namespace: {{.DeployerInfo.Namespace}}{{end}}{{if .DeployerInfo.WorkingDir}}
Working Dir: {{.DeployerInfo.WorkingDir}}{{end}}{{if .DeployerInfo.KubeContext}}
KubeContext: {{.DeployerInfo.KubeContext}}{{end}}{{if .DeployerInfo.RemoteManifestPaths}}
RemoteManifests:
{{range $path := .DeployerInfo.RemoteManifestPaths}}  - {{$path}}
{{end}}{{end}}{{if .DeployerInfo.ManifestPaths}}
Manifests:
{{range $path := .DeployerInfo.ManifestPaths}}  - {{$path}}
{{end}}{{end}}{{if .DeployerInfo.Releases}}
Helm Releases:
{{range $release := .DeployerInfo.Releases}}{{if $release.Version}}  - Version: {{$.release.Version}}
{{end}}{{if $release.ChartPath}}  - Chart Path: {{$release.ChartPath}}
{{end}}{{if $release.ValuesFilePath}} - Values File Path: {{$release.ValuesFilePath}}
{{end}}{{if $release.Values}}  - Values:
{{range $key, $value := $release.Values}}   - {{$key}}: {{$value}}{{end}}
{{end}}{{if $release.SetValues}}  - Set Values:
{{range $key, $value := $release.SetValues}}   - {{$key}}: {{$value}}{{end}}
{{end}}{{if $release.SetValueTemplates}}  - Set Value Templates:
{{range $key, $value := $release.SetValueTemplates}}   - {{$key}}: {{$value}}{{end}}
{{end}}{{if $release.Wait}}  - Wait: {{$release.Wait}}
{{end}}{{if $release.Packaged}}  - App Packaging: {{$release.Packaged}}
{{end}}{{end}}{{end}}
`

type Status struct {
	ConfigInfo
	BuilderInfo
	TaggerInfo
	DeployerInfo
}

type ConfigInfo struct {
	Profiles []string
}

type BuilderInfo struct {
	Name string // required

	// GCB
	ProjectID   string
	Namespace   string
	KubeContext string

	// Kaniko
	GCSBucket      string
	PullSecretName string
	PullSecret     string
}

type TaggerInfo struct {
	Name string // required
	Tag  string

	// Env Template
	Template string
}

type DeployerInfo struct {
	Name string // required

	// helm
	Namespace string
	Releases  []v1alpha2.HelmRelease

	// kubectl
	WorkingDir          string
	KubeContext         string
	ManifestPaths       []string
	RemoteManifestPaths []string

	// kustomize
	KustomizePath string
}
