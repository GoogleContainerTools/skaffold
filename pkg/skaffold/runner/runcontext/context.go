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

package runcontext

import (
	"fmt"
	"os"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	runnerutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

const (
	emptyNamespace = ""
)

type RunContext struct {
	Opts               config.SkaffoldOptions
	Pipelines          Pipelines
	KubeContext        string
	Namespaces         []string
	WorkingDir         string
	InsecureRegistries map[string]bool
	Cluster            config.Cluster
}

// Pipelines encapsulates multiple config pipelines
type Pipelines struct {
	pipelines            []latest.Pipeline
	pipelinesByImageName map[string]latest.Pipeline
}

// All returns all config pipelines.
func (ps Pipelines) All() []latest.Pipeline {
	return ps.pipelines
}

// Head returns the first `latest.Pipeline`.
func (ps Pipelines) Head() latest.Pipeline {
	return ps.pipelines[0] // there always exists atleast one pipeline.
}

// Select returns the first `latest.Pipeline` that matches the given artifact `imageName`.
func (ps Pipelines) Select(imageName string) (latest.Pipeline, bool) {
	p, found := ps.pipelinesByImageName[imageName]
	return p, found
}

func (ps Pipelines) PortForwardResources() []*latest.PortForwardResource {
	var pf []*latest.PortForwardResource
	for _, p := range ps.pipelines {
		pf = append(pf, p.PortForward...)
	}
	return pf
}

func (ps Pipelines) Artifacts() []*latest.Artifact {
	var artifacts []*latest.Artifact
	for _, p := range ps.pipelines {
		artifacts = append(artifacts, p.Build.Artifacts...)
	}
	return artifacts
}

func (ps Pipelines) Deployers() []latest.DeployType {
	var deployers []latest.DeployType
	for _, p := range ps.pipelines {
		deployers = append(deployers, p.Deploy.DeployType)
	}
	return deployers
}

func (ps Pipelines) TestCases() []*latest.TestCase {
	var tests []*latest.TestCase
	for _, p := range ps.pipelines {
		tests = append(tests, p.Test...)
	}
	return tests
}

func (ps Pipelines) StatusCheckDeadlineSeconds() int {
	c := 0
	// set the group status check deadline to maximum of any individually specified value
	for _, p := range ps.pipelines {
		if p.Deploy.StatusCheckDeadlineSeconds > c {
			c = p.Deploy.StatusCheckDeadlineSeconds
		}
	}
	return c
}
func NewPipelines(pipelines []latest.Pipeline) Pipelines {
	m := make(map[string]latest.Pipeline)
	for _, p := range pipelines {
		for _, a := range p.Build.Artifacts {
			m[a.ImageName] = p
		}
	}
	return Pipelines{pipelines: pipelines, pipelinesByImageName: m}
}

func (rc *RunContext) PipelineForImage(imageName string) (latest.Pipeline, bool) {
	return rc.Pipelines.Select(imageName)
}

func (rc *RunContext) PortForwardResources() []*latest.PortForwardResource {
	return rc.Pipelines.PortForwardResources()
}

func (rc *RunContext) Artifacts() []*latest.Artifact { return rc.Pipelines.Artifacts() }

func (rc *RunContext) Deployers() []latest.DeployType { return rc.Pipelines.Deployers() }

func (rc *RunContext) TestCases() []*latest.TestCase { return rc.Pipelines.TestCases() }

func (rc *RunContext) StatusCheckDeadlineSeconds() int {
	return rc.Pipelines.StatusCheckDeadlineSeconds()
}

func (rc *RunContext) DefaultPipeline() latest.Pipeline          { return rc.Pipelines.Head() }
func (rc *RunContext) GetKubeContext() string                    { return rc.KubeContext }
func (rc *RunContext) GetNamespaces() []string                   { return rc.Namespaces }
func (rc *RunContext) GetPipelines() []latest.Pipeline           { return rc.Pipelines.All() }
func (rc *RunContext) GetInsecureRegistries() map[string]bool    { return rc.InsecureRegistries }
func (rc *RunContext) GetWorkingDir() string                     { return rc.WorkingDir }
func (rc *RunContext) GetCluster() config.Cluster                { return rc.Cluster }
func (rc *RunContext) AddSkaffoldLabels() bool                   { return rc.Opts.AddSkaffoldLabels }
func (rc *RunContext) AutoBuild() bool                           { return rc.Opts.AutoBuild }
func (rc *RunContext) AutoDeploy() bool                          { return rc.Opts.AutoDeploy }
func (rc *RunContext) AutoSync() bool                            { return rc.Opts.AutoSync }
func (rc *RunContext) CacheArtifacts() bool                      { return rc.Opts.CacheArtifacts }
func (rc *RunContext) CacheFile() string                         { return rc.Opts.CacheFile }
func (rc *RunContext) ConfigurationFile() string                 { return rc.Opts.ConfigurationFile }
func (rc *RunContext) CustomLabels() []string                    { return rc.Opts.CustomLabels }
func (rc *RunContext) CustomTag() string                         { return rc.Opts.CustomTag }
func (rc *RunContext) DefaultRepo() *string                      { return rc.Opts.DefaultRepo.Value() }
func (rc *RunContext) Mode() config.RunMode                      { return rc.Opts.Mode() }
func (rc *RunContext) DigestSource() string                      { return rc.Opts.DigestSource }
func (rc *RunContext) DryRun() bool                              { return rc.Opts.DryRun }
func (rc *RunContext) ForceDeploy() bool                         { return rc.Opts.Force }
func (rc *RunContext) GetKubeConfig() string                     { return rc.Opts.KubeConfig }
func (rc *RunContext) GetKubeNamespace() string                  { return rc.Opts.Namespace }
func (rc *RunContext) GlobalConfig() string                      { return rc.Opts.GlobalConfig }
func (rc *RunContext) MinikubeProfile() string                   { return rc.Opts.MinikubeProfile }
func (rc *RunContext) Muted() config.Muted                       { return rc.Opts.Muted }
func (rc *RunContext) NoPruneChildren() bool                     { return rc.Opts.NoPruneChildren }
func (rc *RunContext) Notification() bool                        { return rc.Opts.Notification }
func (rc *RunContext) PortForward() bool                         { return rc.Opts.PortForward.Enabled }
func (rc *RunContext) Prune() bool                               { return rc.Opts.Prune() }
func (rc *RunContext) RenderOnly() bool                          { return rc.Opts.RenderOnly }
func (rc *RunContext) RenderOutput() string                      { return rc.Opts.RenderOutput }
func (rc *RunContext) SkipRender() bool                          { return rc.Opts.SkipRender }
func (rc *RunContext) SkipTests() bool                           { return rc.Opts.SkipTests }
func (rc *RunContext) StatusCheck() bool                         { return rc.Opts.StatusCheck }
func (rc *RunContext) Tail() bool                                { return rc.Opts.Tail }
func (rc *RunContext) Trigger() string                           { return rc.Opts.Trigger }
func (rc *RunContext) WaitForDeletions() config.WaitForDeletions { return rc.Opts.WaitForDeletions }
func (rc *RunContext) WatchPollInterval() int                    { return rc.Opts.WatchPollInterval }

func GetRunContext(opts config.SkaffoldOptions, pipelines []latest.Pipeline) (*RunContext, error) {
	kubeConfig, err := kubectx.CurrentConfig()
	if err != nil {
		return nil, fmt.Errorf("getting current cluster context: %w", err)
	}
	kubeContext := kubeConfig.CurrentContext
	logrus.Infof("Using kubectl context: %s", kubeContext)

	// TODO(dgageot): this should be the folder containing skaffold.yaml. Should also be moved elsewhere.
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("finding current directory: %w", err)
	}

	namespaces, err := runnerutil.GetAllPodNamespaces(opts.Namespace, pipelines)
	if err != nil {
		return nil, fmt.Errorf("getting namespace list: %w", err)
	}

	// combine all provided lists of insecure registries into a map
	cfgRegistries, err := config.GetInsecureRegistries(opts.GlobalConfig)
	if err != nil {
		logrus.Warnf("error retrieving insecure registries from global config: push/pull issues may exist...")
	}
	var regList []string
	regList = append(regList, opts.InsecureRegistries...)
	for _, cfg := range pipelines {
		regList = append(regList, cfg.Build.InsecureRegistries...)
	}
	regList = append(regList, cfgRegistries...)
	insecureRegistries := make(map[string]bool, len(regList))
	for _, r := range regList {
		insecureRegistries[r] = true
	}
	ps := NewPipelines(pipelines)

	// TODO(https://github.com/GoogleContainerTools/skaffold/issues/3668):
	// remove minikubeProfile from here and instead detect it by matching the
	// kubecontext API Server to minikube profiles
	cluster, err := config.GetCluster(opts.GlobalConfig, opts.MinikubeProfile, opts.DetectMinikube)
	if err != nil {
		return nil, fmt.Errorf("getting cluster: %w", err)
	}

	return &RunContext{
		Opts:               opts,
		Pipelines:          ps,
		WorkingDir:         cwd,
		KubeContext:        kubeContext,
		Namespaces:         namespaces,
		InsecureRegistries: insecureRegistries,
		Cluster:            cluster,
	}, nil
}

func (rc *RunContext) UpdateNamespaces(ns []string) {
	if len(ns) == 0 {
		return
	}

	nsMap := map[string]bool{}
	for _, ns := range append(ns, rc.Namespaces...) {
		if ns == emptyNamespace {
			continue
		}
		nsMap[ns] = true
	}

	// Update RunContext Namespace
	updated := make([]string, 0, len(nsMap))
	for k := range nsMap {
		updated = append(updated, k)
	}
	sort.Strings(updated)
	rc.Namespaces = updated
}
