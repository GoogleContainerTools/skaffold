/*
Copyright 2021 The Skaffold Authors

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

package runner

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/cloudrun"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	kptV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
)

// deployerCtx encapsulates a given skaffold run context along with additional deployer constructs.
type deployerCtx struct {
	*runcontext.RunContext
	deploy latest.DeployConfig
}

func (d *deployerCtx) GetKubeContext() string {
	// if the kubeContext is not overridden by CLI flag or env. variable then use the value provided in config.
	if d.RunContext.IsDefaultKubeContext() && d.deploy.KubeContext != "" {
		return d.deploy.KubeContext
	}
	return d.RunContext.GetKubeContext()
}

func (d *deployerCtx) StatusCheck() *bool {
	// runcontext StatusCheck method returns the value set by the cli flag `--status-check`
	// which overrides the value set in the individual configs.
	if cliValue := d.RunContext.StatusCheck(); cliValue != nil {
		return cliValue
	}
	return d.deploy.StatusCheck
}

// JsonParseType returns the JsonParseType field from the underlying deployConfig struct
func (d *deployerCtx) JSONParseConfig() latest.JSONParseConfig {
	return d.deploy.Logs.JSONParse
}

// GetDeployer creates a deployer from a given RunContext and deploy pipeline definitions.
func GetDeployer(ctx context.Context, runCtx *runcontext.RunContext, labeller *label.DefaultLabeller, hydrationDir string, usingLegacyHelmDeploy bool) (deploy.Deployer, error) {
	pipelines := runCtx.Pipelines

	if runCtx.Opts.Apply {
		helmNamespaces := make(map[string]bool)
		nonHelmDeployFound := false
		cloudRunDeployFound := false

		for _, d := range pipelines.Deployers() {
			if d.DockerDeploy != nil || d.KptDeploy != nil || d.KubectlDeploy != nil || d.KustomizeDeploy != nil {
				nonHelmDeployFound = true
			}

			if d.CloudRunDeploy != nil {
				cloudRunDeployFound = true
			}

			if d.LegacyHelmDeploy != nil {
				for _, release := range d.LegacyHelmDeploy.Releases {
					if release.Namespace != "" {
						helmNamespaces[release.Namespace] = true
					}
				}
			}
		}
		if cloudRunDeployFound {
			if nonHelmDeployFound || len(helmNamespaces) > 0 {
				// Cloud Run doesn't support multiple deployers in the config.
				return nil, errors.New("skaffold apply called with both Cloud Run and Kubernetes deployers. Mixing deployment targets is not allowed" +
					" when using the Cloud Run deployer")
			}
			return getCloudRunDeployer(runCtx, labeller)
		}
		if len(helmNamespaces) > 1 || (nonHelmDeployFound && len(helmNamespaces) == 1) {
			return nil, errors.New("skaffold apply called with conflicting namespaces set via skaffold.yaml. This is likely due to the use of the 'deploy.helm.releases.*.namespace' field which is not supported in apply.  Remove the 'deploy.helm.releases.*.namespace' field(s) and run skaffold apply again")
		}

		if len(helmNamespaces) == 1 && !nonHelmDeployFound {
			if runCtx.Opts.Namespace == "" {
				// if skaffold --namespace flag not set, use the helm namespace value
				for k := range helmNamespaces {
					// map only has 1 (k,v) from length check in if condition
					runCtx.Opts.Namespace = k
				}
			}
		}

		return getDefaultDeployer(runCtx, labeller, hydrationDir)
	}

	var deployers []deploy.Deployer
	localDeploy := false
	remoteDeploy := false
	for _, pl := range pipelines.All() {
		d := pl.Deploy
		r := pl.Render
		dCtx := &deployerCtx{runCtx, d}

		if d.DockerDeploy != nil {
			localDeploy = true
			d, err := docker.NewDeployer(ctx, runCtx, labeller, d.DockerDeploy, runCtx.PortForwardResources())
			if err != nil {
				return nil, err
			}
			// Override the cluster on the runcontext.
			// This is used to determine whether we should push images, and we want to avoid that unless explicitly asked for.
			// Safe to do because we explicitly disallow simultaneous remote and local deployments.
			runCtx.Cluster = config.Cluster{
				Local:      true,
				PushImages: false,
				LoadImages: false,
			}
			deployers = append(deployers, d)
		}

		if d.LegacyHelmDeploy != nil {
			// copy relevant render config to legacy helm deployer
			if r.Helm != nil {
				d.LegacyHelmDeploy.Releases = r.Helm.Releases
				d.LegacyHelmDeploy.Flags = r.Helm.Flags
			}

			h, err := helm.NewDeployer(ctx, dCtx, labeller, d.LegacyHelmDeploy, runCtx.Artifacts())
			if err != nil {
				return nil, err
			}
			deployers = append(deployers, h)
		}

		if d.KubectlDeploy != nil {
			deployer, err := kubectl.NewDeployer(dCtx, labeller, d.KubectlDeploy, hydrationDir)
			if err != nil {
				return nil, err
			}
			deployers = append(deployers, deployer)
		}

		if d.KptDeploy != nil {
			if d.KptDeploy.Dir == "" {
				log.Entry(context.TODO()).Infof("manifests are deployed from render path %v\n", hydrationDir)
				d.KptDeploy.Dir = hydrationDir
			}
			deployer, err := kptV2.NewDeployer(dCtx, labeller, d.KptDeploy, runCtx.Opts)
			if err != nil {
				return nil, err
			}
			deployers = append(deployers, deployer)
		}

		if d.KustomizeDeploy != nil {
			deployer, err := kustomize.NewDeployer(dCtx, labeller, d.KustomizeDeploy)
			if err != nil {
				return nil, err
			}
			deployers = append(deployers, deployer)
		}
		if d.CloudRunDeploy != nil {
			deployer, err := cloudrun.NewDeployer(labeller, d.CloudRunDeploy)
			if err != nil {
				return nil, err
			}
			deployers = append(deployers, deployer)
		}
	}

	if localDeploy && remoteDeploy {
		return nil, errors.New("docker deployment not supported alongside cluster deployments")
	}

	return deploy.NewDeployerMux(deployers, runCtx.IterativeStatusCheck()), nil
}

/*
The "default deployer" is used in `skaffold apply`, which uses a `kubectl` deployer to actuate resources
on a cluster regardless of provided deployer configuration in the skaffold.yaml.
The default deployer will honor a select set of deploy configuration from an existing skaffold.yaml:
	- deploy.StatusCheckDeadlineSeconds
	- deploy.Logs.Prefix
	- deploy.Kubectl.Flags
	- deploy.Kubectl.DefaultNamespace
	- deploy.Kustomize.Flags
	- deploy.Kustomize.DefaultNamespace
For a multi-config project, we do not currently support resolving conflicts between differing sets of this deploy configuration.
Therefore, in this function we do implicit validation of the provided configuration, and fail if any conflict cannot be resolved.
*/
func getDefaultDeployer(runCtx *runcontext.RunContext, labeller *label.DefaultLabeller, hydrationDir string) (deploy.Deployer, error) {
	deployCfgs := runCtx.DeployConfigs()

	var kFlags *latest.KubectlFlags
	var logPrefix string
	var defaultNamespace *string
	var kubeContext string
	statusCheckTimeout := -1
	var statusCheck *bool
	for _, d := range deployCfgs {
		if d.KubeContext != "" {
			if kubeContext != "" && kubeContext != d.KubeContext {
				return nil, errors.New("cannot resolve active Kubernetes context - multiple contexts configured in skaffold.yaml")
			}
			kubeContext = d.KubeContext
		}
		if d.StatusCheck != nil {
			if statusCheck == nil {
				statusCheck = d.StatusCheck
			} else if statusCheck != d.StatusCheck {
				// if we get conflicting values for status check from different skaffold configs, we turn status check off
				statusCheck = util.BoolPtr(false)
			}
		}
		if d.StatusCheckDeadlineSeconds != 0 && d.StatusCheckDeadlineSeconds != int(status.DefaultStatusCheckDeadline.Seconds()) {
			if statusCheckTimeout != -1 && statusCheckTimeout != d.StatusCheckDeadlineSeconds {
				return nil, fmt.Errorf("found multiple status check timeouts in skaffold.yaml (not supported in `skaffold apply`): %d, %d", statusCheckTimeout, d.StatusCheckDeadlineSeconds)
			}
			statusCheckTimeout = d.StatusCheckDeadlineSeconds
		}
		if d.Logs.Prefix != "" {
			if logPrefix != "" && logPrefix != d.Logs.Prefix {
				return nil, fmt.Errorf("found multiple log prefixes in skaffold.yaml (not supported in `skaffold apply`): %s, %s", logPrefix, d.Logs.Prefix)
			}
			logPrefix = d.Logs.Prefix
		}
		var currentDefaultNamespace *string
		var currentKubectlFlags latest.KubectlFlags
		if d.KubectlDeploy != nil {
			currentDefaultNamespace = d.KubectlDeploy.DefaultNamespace
			currentKubectlFlags = d.KubectlDeploy.Flags
		}
		if d.KustomizeDeploy != nil {
			currentDefaultNamespace = d.KustomizeDeploy.DefaultNamespace
			currentKubectlFlags = d.KustomizeDeploy.Flags
		}
		if kFlags == nil {
			kFlags = &currentKubectlFlags
		}
		if err := validateKubectlFlags(kFlags, currentKubectlFlags); err != nil {
			return nil, err
		}
		if currentDefaultNamespace != nil {
			if defaultNamespace != nil && *defaultNamespace != *currentDefaultNamespace {
				return nil, fmt.Errorf("found multiple namespaces in skaffold.yaml (not supported in `skaffold apply`): %s, %s", *defaultNamespace, *currentDefaultNamespace)
			}
			defaultNamespace = currentDefaultNamespace
		}
	}
	if kFlags == nil {
		kFlags = &latest.KubectlFlags{}
	}
	k := &latest.KubectlDeploy{
		Flags:            *kFlags,
		DefaultNamespace: defaultNamespace,
	}
	dCtx := &deployerCtx{runCtx, latest.DeployConfig{StatusCheck: statusCheck, KubeContext: kubeContext, DeployType: latest.DeployType{KubectlDeploy: k}}}
	defaultDeployer, err := kubectl.NewDeployer(dCtx, labeller, k, hydrationDir)
	if err != nil {
		return nil, fmt.Errorf("instantiating default kubectl deployer: %w", err)
	}
	return defaultDeployer, nil
}

func validateKubectlFlags(flags *latest.KubectlFlags, additional latest.KubectlFlags) error {
	errStr := "conflicting sets of kubectl deploy flags not supported in `skaffold apply` (flag: %s)"
	if additional.DisableValidation != flags.DisableValidation {
		return fmt.Errorf(errStr, strconv.FormatBool(additional.DisableValidation))
	}
	for _, flag := range additional.Apply {
		if !stringslice.Contains(flags.Apply, flag) {
			return fmt.Errorf(errStr, flag)
		}
	}
	for _, flag := range additional.Delete {
		if !stringslice.Contains(flags.Delete, flag) {
			return fmt.Errorf(errStr, flag)
		}
	}
	for _, flag := range additional.Global {
		if !stringslice.Contains(flags.Global, flag) {
			return fmt.Errorf(errStr, flag)
		}
	}
	return nil
}

/* The Cloud Run deployer for apply. Used when Cloud Run is specified. */
func getCloudRunDeployer(runCtx *runcontext.RunContext, labeller *label.DefaultLabeller) (*cloudrun.Deployer, error) {
	var region string
	var defaultProject string
	for _, d := range runCtx.DeployConfigs() {
		if d.CloudRunDeploy != nil {
			crDeploy := d.CloudRunDeploy
			if region != "" && region != crDeploy.Region {
				return nil, fmt.Errorf("expected all Cloud Run deploys to be in the same region, found deploys to %s and %s", region, crDeploy.Region)
			}
			region = crDeploy.Region
			if defaultProject != "" && defaultProject != crDeploy.DefaultProjectID {
				return nil, fmt.Errorf("expected all Cloud Run deploys to use the same default project, found deploys to projects %s and %s", defaultProject, crDeploy.DefaultProjectID)
			}
		}
	}
	return cloudrun.NewDeployer(labeller, &latest.CloudRunDeploy{Region: region, DefaultProjectID: defaultProject})
}
