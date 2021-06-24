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
	"errors"
	"fmt"
	"strconv"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/helm"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kpt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kustomize"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	v1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// deployerCtx encapsulates a given skaffold run context along with additional deployer constructs.
type deployerCtx struct {
	*runcontext.RunContext
	deploy v1.DeployConfig
}

func (d *deployerCtx) GetKubeContext() string {
	if d.deploy.KubeContext != "" {
		return d.deploy.KubeContext
	}
	return d.RunContext.GetKubeContext()
}

// GetDeployer creates a deployer from a given RunContext and deploy pipeline definitions.
func GetDeployer(runCtx *runcontext.RunContext, provider deploy.ComponentProvider, labels map[string]string) (deploy.Deployer, kubernetes.ImageListMux, error) {
	var podSelectors kubernetes.ImageListMux
	if runCtx.Opts.Apply {
		return getDefaultDeployer(runCtx, provider, labels)
	}

	deployerCfg := runCtx.Deployers()

	var deployers deploy.DeployerMux
	for _, d := range deployerCfg {
		dCtx := &deployerCtx{runCtx, d}
		if d.HelmDeploy != nil {
			h, podSelector, err := helm.NewDeployer(dCtx, labels, provider, d.HelmDeploy)
			if err != nil {
				return nil, nil, err
			}
			podSelectors = append(podSelectors, podSelector)
			deployers = append(deployers, h)
		}

		if d.KptDeploy != nil {
			deployer, podSelector := kpt.NewDeployer(dCtx, labels, provider, d.KptDeploy)
			podSelectors = append(podSelectors, podSelector)
			deployers = append(deployers, deployer)
		}

		if d.KubectlDeploy != nil {
			deployer, podSelector, err := kubectl.NewDeployer(dCtx, labels, provider, d.KubectlDeploy)
			if err != nil {
				return nil, nil, err
			}
			podSelectors = append(podSelectors, podSelector)
			deployers = append(deployers, deployer)
		}

		if d.KustomizeDeploy != nil {
			deployer, podSelector, err := kustomize.NewDeployer(dCtx, labels, provider, d.KustomizeDeploy)
			if err != nil {
				return nil, nil, err
			}
			podSelectors = append(podSelectors, podSelector)
			deployers = append(deployers, deployer)
		}
	}

	return deployers, podSelectors, nil
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
func getDefaultDeployer(runCtx *runcontext.RunContext, provider deploy.ComponentProvider, labels map[string]string) (deploy.Deployer, kubernetes.ImageListMux, error) {
	deployCfgs := runCtx.DeployConfigs()

	var kFlags *v1.KubectlFlags
	var logPrefix string
	var defaultNamespace *string
	var kubeContext string
	statusCheckTimeout := -1

	for _, d := range deployCfgs {
		if d.KubeContext != "" {
			if kubeContext != "" && kubeContext != d.KubeContext {
				return nil, nil, errors.New("cannot resolve active Kubernetes context - multiple contexts configured in skaffold.yaml")
			}
			kubeContext = d.KubeContext
		}
		if d.StatusCheckDeadlineSeconds != 0 && d.StatusCheckDeadlineSeconds != int(status.DefaultStatusCheckDeadline.Seconds()) {
			if statusCheckTimeout != -1 && statusCheckTimeout != d.StatusCheckDeadlineSeconds {
				return nil, nil, fmt.Errorf("found multiple status check timeouts in skaffold.yaml (not supported in `skaffold apply`): %d, %d", statusCheckTimeout, d.StatusCheckDeadlineSeconds)
			}
			statusCheckTimeout = d.StatusCheckDeadlineSeconds
		}
		if d.Logs.Prefix != "" {
			if logPrefix != "" && logPrefix != d.Logs.Prefix {
				return nil, nil, fmt.Errorf("found multiple log prefixes in skaffold.yaml (not supported in `skaffold apply`): %s, %s", logPrefix, d.Logs.Prefix)
			}
			logPrefix = d.Logs.Prefix
		}
		var currentDefaultNamespace *string
		var currentKubectlFlags v1.KubectlFlags
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
			return nil, nil, err
		}
		if currentDefaultNamespace != nil {
			if defaultNamespace != nil && *defaultNamespace != *currentDefaultNamespace {
				return nil, nil, fmt.Errorf("found multiple namespaces in skaffold.yaml (not supported in `skaffold apply`): %s, %s", *defaultNamespace, *currentDefaultNamespace)
			}
			defaultNamespace = currentDefaultNamespace
		}
	}
	if kFlags == nil {
		kFlags = &v1.KubectlFlags{}
	}
	k := &v1.KubectlDeploy{
		Flags:            *kFlags,
		DefaultNamespace: defaultNamespace,
	}
	defaultDeployer, podSelector, err := kubectl.NewDeployer(runCtx, labels, provider, k)
	if err != nil {
		return nil, nil, fmt.Errorf("instantiating default kubectl deployer: %w", err)
	}
	return defaultDeployer, kubernetes.ImageListMux{podSelector}, nil
}

func validateKubectlFlags(flags *v1.KubectlFlags, additional v1.KubectlFlags) error {
	errStr := "conflicting sets of kubectl deploy flags not supported in `skaffold apply` (flag: %s)"
	if additional.DisableValidation != flags.DisableValidation {
		return fmt.Errorf(errStr, strconv.FormatBool(additional.DisableValidation))
	}
	for _, flag := range additional.Apply {
		if !util.StrSliceContains(flags.Apply, flag) {
			return fmt.Errorf(errStr, flag)
		}
	}
	for _, flag := range additional.Delete {
		if !util.StrSliceContains(flags.Delete, flag) {
			return fmt.Errorf(errStr, flag)
		}
	}
	for _, flag := range additional.Global {
		if !util.StrSliceContains(flags.Global, flag) {
			return fmt.Errorf(errStr, flag)
		}
	}
	return nil
}
