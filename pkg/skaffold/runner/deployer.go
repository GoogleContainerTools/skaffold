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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	kptV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/v2/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext/v2"
	latestV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util/stringslice"
)

// deployerCtx encapsulates a given skaffold run context along with additional deployer constructs.
type deployerCtx struct {
	*v2.RunContext
	deploy latestV2.DeployConfig
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
func (d *deployerCtx) JSONParseConfig() latestV2.JSONParseConfig {
	return d.deploy.Logs.JSONParse
}

// GetDeployer creates a deployer from a given RunContext and deploy pipeline definitions.
func GetDeployer(ctx context.Context, runCtx *v2.RunContext, labeller *label.DefaultLabeller, hydrationDir string) (deploy.Deployer, error) {
	if runCtx.Opts.Apply {
		return getDefaultDeployer(runCtx, labeller, hydrationDir)
	}

	var deployers []deploy.Deployer

	// TODO(nkubala)[v2-merge]: Dirty workaround due to the missing helm strategy in kpt.
	// This should be moved to renderer.generate instead, and replaced with LegacyHelmDeployer
	pipelines := runCtx.GetPipelines()
	for _, p := range pipelines {
		if p.Render.Generate.Helm != nil {
			dCtx := &deployerCtx{runCtx, p.Deploy}
			h, err := helm.NewDeployer(ctx, dCtx, labeller, &latestV2.LegacyHelmDeploy{
				Flags: p.Render.Generate.Helm.Flags,
			}, runCtx.Artifacts())
			if err != nil {
				return nil, err
			}
			if p.Render.Generate.Helm.Releases != nil {
				h.Releases = append(h.Releases, *p.Render.Generate.Helm.Releases...)
			}
			deployers = append(deployers, h)
		}
	}
	deployerCfg := runCtx.Deployers()
	localDeploy := false
	remoteDeploy := false
	for _, d := range deployerCfg {
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

		// TODO(nkubala)[v2-merge]: add d.LegacyHelmDeploy (or something similar)
		if d.LegacyHelmDeploy != nil {
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

		if d.KptV2Deploy != nil {
			if d.KptV2Deploy.Dir == "" {
				log.Entry(context.TODO()).Infof("manifests are deployed from render path %v\n", hydrationDir)
				d.KptV2Deploy.Dir = hydrationDir
			}
			deployer, err := kptV2.NewDeployer(dCtx, labeller, d.KptV2Deploy, runCtx.Opts)
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
func getDefaultDeployer(runCtx *v2.RunContext, labeller *label.DefaultLabeller, hydrationDir string) (deploy.Deployer, error) {
	deployCfgs := runCtx.DeployConfigs()

	var kFlags *latestV2.KubectlFlags
	var logPrefix string
	var defaultNamespace *string
	var kubeContext string
	statusCheckTimeout := -1

	for _, d := range deployCfgs {
		if d.KubeContext != "" {
			if kubeContext != "" && kubeContext != d.KubeContext {
				return nil, errors.New("cannot resolve active Kubernetes context - multiple contexts configured in skaffold.yaml")
			}
			kubeContext = d.KubeContext
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
		var currentKubectlFlags latestV2.KubectlFlags
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
		kFlags = &latestV2.KubectlFlags{}
	}
	k := &latestV2.KubectlDeploy{
		Flags:            *kFlags,
		DefaultNamespace: defaultNamespace,
	}
	defaultDeployer, err := kubectl.NewDeployer(runCtx, labeller, k, hydrationDir)
	if err != nil {
		return nil, fmt.Errorf("instantiating default kubectl deployer: %w", err)
	}
	return defaultDeployer, nil
}

func validateKubectlFlags(flags *latestV2.KubectlFlags, additional latestV2.KubectlFlags) error {
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
