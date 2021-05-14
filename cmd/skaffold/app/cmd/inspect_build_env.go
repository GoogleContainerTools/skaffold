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

package cmd

import (
	"context"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect"
	buildEnv "github.com/GoogleContainerTools/skaffold/pkg/skaffold/inspect/buildEnv"
)

var buildEnvFlags = struct {
	profile     string

	// Common
	timeout string
	concurrency int

	// Google Cloud Build
	projectID   string
	diskSizeGb  int64
	machineType string

	// Cluster (kaniko)
	pullSecretPath string
	pullSecretName string
	pullSecretMountPath string
	namespace string

	dockerConfigPath string
	dockerConfigSecretName string

	serviceAccount string
	runAsUser int64

	cpuRequest string
	memoryRequest string
	ephemeralStorageRequest string
	resourceStorageRequest string

	cpuLimit string
	memoryLimit string
	ephemeralStorageLimit string
	resourceStorageLimit string

	randomPullSecret bool
	randomDockerConigSecret bool
}{}

func cmdBuildEnv() *cobra.Command {
	return NewCmd("build-env").
		WithDescription("Interact with skaffold build environment definitions.").
		WithPersistentFlagAdder(cmdBuildEnvFlags).
		WithCommands(cmdBuildEnvList(), cmdBuildEnvAdd())
}

func cmdBuildEnvList() *cobra.Command {
	return NewCmd("list").
		WithExample("Get list of target build environments with activated profiles p1 and p2", "inspect build-env list -p p1,p2 --format json").
		WithDescription("Print the list of active build environments.").
		WithFlagAdder(cmdBuildEnvListFlags).
		NoArgs(listBuildEnv)
}

func cmdBuildEnvAdd() *cobra.Command {
	return NewCmd("add").
		WithDescription("Add a new build environment to the default pipeline or to a new or existing profile.").
		WithPersistentFlagAdder(cmdBuildEnvAddFlags).
		WithCommands(cmdBuildEnvAddGcb(), cmdBuildEnvAddCluster())
}

func cmdBuildEnvAddGcb() *cobra.Command {
	return NewCmd("googleCloudBuild").
		WithDescription("Add a new GoogleCloudBuild build environment definition").
		WithLongDescription(`Add a new GoogleCloudBuild build environment definition.
Without the '--profile' flag the new environment definition is added to the default pipeline. With the '--profile' flag it will create a new profile with this build env definition. 
In these respective scenarios, it will fail if the build env definition for the default pipeline or the named profile already exists. To override an existing definition use 'skaffold inspect build-env modify' command instead. 
Use the '--module' filter to specify the individual module to target. Otherwise, it'll be applied to all modules defined in the target file. Also, with the '--profile' flag if the target config imports other configs as dependencies, then the new profile will be recursively created in all the imported configs also.`).
		WithExample("Add a new profile named 'gcb' targeting the builder 'googleCloudBuild' against the GCP project ID '1234'.", "inspect build-env add googleCloudBuild --profile gcb --projectID 1234 -f skaffold.yaml").
		WithFlagAdder(cmdBuildEnvAddGcbFlags).
		NoArgs(addGcbBuildEnv)
}

func cmdBuildEnvAddCluster() *cobra.Command {
	return NewCmd("cluster").
		WithDescription("Add a new Cluster build environment definition").
			WithLongDescription(`Add a new Cluster build environment definition.
Without the '--profile' flag the new environment definition is added to the default pipeline. With the '--profile' flag it will create a new profile with this build env definition. 
In these respective scenarios, it will fail if the build env definition for the default pipeline or the named profile already exists. To override an existing definition use 'skaffold inspect build-env modify' command instead. 
Use the '--module' filter to specify the individual module to target. Otherwise, it'll be applied to all modules defined in the target file. Also, with the '--profile' flag if the target config imports other configs as dependencies, then the new profile will be recursively created in all the imported configs also.`).
		WithExample("Add a new profile named 'cluster' targeting the builder 'kaniko' using the Kubernetes secret 'my-secret'", "inspect build-env add cluster --profile cluster --pullSecretName my-secret -f skaffold.yaml").
		WithFlagAdder(cmdBuildEnvAddClusterFlags).
		NoArgs(addClusterBuildEnv)
}

func listBuildEnv(ctx context.Context, out io.Writer) error {
	return buildEnv.PrintBuildEnvsList(ctx, out, printBuildEnvsListOptions())
}

func addGcbBuildEnv(ctx context.Context, out io.Writer) error {
	return buildEnv.AddGcbBuildEnv(ctx, out, addGcbBuildEnvOptions())
}

func addClusterBuildEnv(ctx context.Context, out io.Writer) error {
	return buildEnv.AddClusterBuildEnv(ctx, out, addClusterBuildEnvOptions())
}

func cmdBuildEnvAddFlags(f *pflag.FlagSet) {
	f.StringVarP(&buildEnvFlags.profile, "profile", "p", "", `Profile name to add the new build env definition in. If the profile name doesn't exist then the profile will be created in all the target configs. If this flag is not specified then the build env is added to the default pipeline of the target configs.`)
}

func cmdBuildEnvAddGcbFlags(f *pflag.FlagSet) {
	f.StringVar(&buildEnvFlags.projectID, "projectId", "", `ID of the Cloud Platform Project.`)
	f.Int64Var(&buildEnvFlags.diskSizeGb, "diskSizeGb", 0, `Disk size of the VM that runs the build`)
	f.StringVar(&buildEnvFlags.machineType, "machineType", "", `Type of VM that runs the build`)
	f.StringVar(&buildEnvFlags.timeout, "timeout", "", `Build timeout (in seconds)`)
	f.IntVar(&buildEnvFlags.concurrency, "concurrency", -1, `number of artifacts to build concurrently. 0 means "no-limit"`)
}

func cmdBuildEnvAddClusterFlags(f *pflag.FlagSet) {
	f.StringVar(&buildEnvFlags.timeout, "timeout", "", `Build timeout (in seconds)`)
	f.IntVar(&buildEnvFlags.concurrency, "concurrency", -1, `number of artifacts to build concurrently. 0 means "no-limit"`)

	f.StringVar(&buildEnvFlags.pullSecretPath, "pullSecretPath", "", "Path to the Google Cloud service account secret key file.")
	f.StringVar(&buildEnvFlags.pullSecretName, "pullSecretName", "", "Name of the Kubernetes secret for pulling base images and pushing the final image.")
	f.StringVar(&buildEnvFlags.pullSecretMountPath, "pullSecretMountPath", "", "Path the pull secret will be mounted at within the running container.")
	f.StringVar(&buildEnvFlags.namespace, "namespace", "", "Kubernetes namespace.")

	f.StringVar(&buildEnvFlags.dockerConfigPath, "dockerConfigPath", "", "Path to the docker config.json.")
	f.StringVar(&buildEnvFlags.dockerConfigSecretName, "dockerConfigSecretName", "", "Kubernetes secret that contains the config.json Docker configuration.")

	f.StringVar(&buildEnvFlags.serviceAccount, "serviceAccount", "", "Kubernetes service account to use for the pod.")
	f.Int64Var(&buildEnvFlags.runAsUser, "runAsUser", -1, "Defines the UID to request for running the container.")

	f.StringVar(&buildEnvFlags.cpuRequest, "cpuRequest", "", "Number of cores to be used.")
	f.StringVar(&buildEnvFlags.memoryRequest, "memoryRequest", "", "Amount of memory to allocate to the pod.")
	f.StringVar(&buildEnvFlags.ephemeralStorageRequest, "ephemeralStorageRequest", "", "Amount of Ephemeral storage to allocate to the pod.")
	f.StringVar(&buildEnvFlags.resourceStorageRequest, "resourceStorageRequest", "", "Amount of resource storage to allocate to the pod.")

	f.StringVar(&buildEnvFlags.cpuLimit, "cpuLimit", "", "Limit on number of cores to be used.")
	f.StringVar(&buildEnvFlags.memoryLimit, "memoryLimit", "", "Limit on amount of memory to allocate to the pod.")
	f.StringVar(&buildEnvFlags.ephemeralStorageLimit, "ephemeralStorageLimit", "", "Limit on amount of ephemeral storage to allocate to the pod.")
	f.StringVar(&buildEnvFlags.resourceStorageLimit, "resourceStorageLimit", "", "Limit on amount of resource storage to allocation to the pod.")

	f.BoolVar(&buildEnvFlags.randomPullSecret, "randomPullSecret", false, "Adds a random UUID postfix to the default name of the pull secret to facilitate parallel builds.")
	f.BoolVar(&buildEnvFlags.randomDockerConigSecret, "randomDockerConigSecret", false, "Adds a random UUID postfix to the default name of the docker secret to facilitate parallel builds.")
}

func cmdBuildEnvFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
}

func cmdBuildEnvListFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.profiles, "profile", "p", nil, `Profile names to activate`)
}

func printBuildEnvsListOptions() inspect.Options {
	return inspect.Options{
		Filename:  inspectFlags.fileName,
		OutFormat: inspectFlags.outFormat,
		Modules:   inspectFlags.modules,
		BuildEnvOptions: inspect.BuildEnvOptions{
			Profiles: inspectFlags.profiles,
		},
	}
}
func addGcbBuildEnvOptions() inspect.Options {
	return inspect.Options{
		Filename:  inspectFlags.fileName,
		OutFormat: inspectFlags.outFormat,
		Modules:   inspectFlags.modules,
		BuildEnvOptions: inspect.BuildEnvOptions{
			Profile:     buildEnvFlags.profile,
			ProjectID:   buildEnvFlags.projectID,
			DiskSizeGb:  buildEnvFlags.diskSizeGb,
			MachineType: buildEnvFlags.machineType,
			Timeout:     buildEnvFlags.timeout,
			Concurrency: buildEnvFlags.concurrency,
		},
	}
}
func addClusterBuildEnvOptions() inspect.Options {
	return inspect.Options{
		Filename:  inspectFlags.fileName,
		OutFormat: inspectFlags.outFormat,
		Modules:   inspectFlags.modules,
		BuildEnvOptions: inspect.BuildEnvOptions{
			PullSecretPath: buildEnvFlags.pullSecretPath,
			PullSecretName: buildEnvFlags.pullSecretName,
			PullSecretMountPath: buildEnvFlags.pullSecretMountPath,
			Namespace: buildEnvFlags.namespace,
			DockerConfigPath: buildEnvFlags.dockerConfigPath,
			DockerConfigSecretName: buildEnvFlags.dockerConfigSecretName,
			ServiceAccount: buildEnvFlags.serviceAccount,
			RunAsUser: buildEnvFlags.runAsUser,
			CpuRequest: buildEnvFlags.cpuRequest,
			MemoryRequest: buildEnvFlags.memoryRequest,
			EphemeralStorageRequest: buildEnvFlags.ephemeralStorageRequest,
			ResourceStorageRequest: buildEnvFlags.resourceStorageRequest,
			CpuLimit: buildEnvFlags.cpuLimit,
			MemoryLimit: buildEnvFlags.memoryLimit,
			EphemeralStorageLimit: buildEnvFlags.ephemeralStorageLimit,
			ResourceStorageLimit: buildEnvFlags.resourceStorageLimit,
			RandomPullSecret: buildEnvFlags.randomPullSecret,
			RandomDockerConigSecret: buildEnvFlags.randomDockerConigSecret,
			Timeout:     buildEnvFlags.timeout,
			Concurrency: buildEnvFlags.concurrency,
		},
	}
}
