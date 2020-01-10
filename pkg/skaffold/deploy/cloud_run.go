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

package deploy

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/gcp"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"

	"google.golang.org/api/run/v1"
)

// CloudRunDeployer deploys workflows using CloudRun CLI.
type CloudRunDeployer struct {
	*latest.CloudRunDeploy

	envVars []*run.EnvVar
}

type Build build.Artifact

// NewCloudRunDeployer returns a new CloudRunDeployer for a DeployConfig filled
// with the needed configuration for `CloudRun apply`
func NewCloudRunDeployer(runCtx *runcontext.RunContext) *CloudRunDeployer {
	envVars := []*run.EnvVar{}
	for name, value := range runCtx.Cfg.Deploy.CloudRunDeploy.Env {
		envVars = append(envVars, &run.EnvVar{Name: name, Value: value})
	}
	return &CloudRunDeployer{
		CloudRunDeploy: runCtx.Cfg.Deploy.CloudRunDeploy,
		envVars:        envVars,
	}
}

func (c *CloudRunDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "CloudRun",
	}
}

func executeTemplate(nameTemplate string) (string, error) {
	tmpl, err := util.ParseEnvTemplate(nameTemplate)
	if err != nil {
		return "", errors.Wrap(err, "parsing template")
	}

	return util.ExecuteEnvTemplate(tmpl, nil)
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `CloudRun apply` on those manifests
func (c *CloudRunDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact, labellers []Labeller) *Result {
	event.DeployInProgress()

	runService, err := run.NewService(ctx, gcp.ClientOptions()...)

	if err != nil {
		event.DeployFailed(err)
		return NewDeployErrorResult(err)
	}

	for _, build := range builds {
		err := c.deployArtifact(runService, Build(build))
		if err != nil {
			event.DeployFailed(err)
			return NewDeployErrorResult(err)
		}
	}

	event.DeployComplete()
	return NewDeploySuccessResult(nil)
}

func (build Build) serviceName() string {
	parts := strings.Split(build.ImageName, "/")
	return parts[len(parts)-1]
}

func (build Build) containerImageName() string {
	return strings.Split(build.Tag, "@sha25")[0]
}

func (c *CloudRunDeployer) deployArtifact(runService *run.APIService, build Build) error {
	projectid, err := gcp.ExtractProjectID(build.Tag)
	if err != nil {
		return err
	}

	serviceName, err := executeTemplate(c.Name)
	if err != nil {
		return err
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", projectid, c.Region)
	name := fmt.Sprintf("%s/services/%s", parent, serviceName)

	envVars := []*run.EnvVar{}
	for _, env := range c.envVars {
		envValue, err := executeTemplate(env.Value)
		if err != nil {
			return err
		}
		envVars = append(envVars, &run.EnvVar{
			Name:  env.Name,
			Value: envValue,
		})
	}

	newService := run.Service{
		ApiVersion: "serving.knative.dev/v1",
		Kind:       "Service",
		Metadata: &run.ObjectMeta{
			Name: serviceName,
		},
		Spec: &run.ServiceSpec{
			Template: &run.RevisionTemplate{
				Spec: &run.RevisionSpec{
					Containers: []*run.Container{
						&run.Container{
							Image: build.containerImageName(),
							Env:   envVars,
						},
					},
				},
			},
		}}

	_, err = runService.Projects.Locations.Services.Get(name).Do()
	isExists := err == nil

	if !isExists {
		_, err = runService.Projects.Locations.Services.Create(parent, &newService).Do()
	} else {
		_, err = runService.Projects.Locations.Services.ReplaceService(name, &newService).Do()
	}

	if err != nil {
		return err
	}

	_, err = runService.Projects.Locations.Services.SetIamPolicy(name, &run.SetIamPolicyRequest{
		Policy: &run.Policy{
			Bindings: []*run.Binding{
				&run.Binding{
					Members: []string{"allUsers"},
					Role:    "roles/run.invoker",
				},
			},
		},
	}).Do()

	if err != nil {
		return err
	}

	return nil
}

func (c *CloudRunDeployer) Dependencies() ([]string, error) {
	return nil, nil
}

func (c *CloudRunDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	return nil
}

func (c *CloudRunDeployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, filepath string) error {
	return nil
}
