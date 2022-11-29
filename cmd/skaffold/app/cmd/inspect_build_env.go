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

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	buildEnv "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect/buildEnv"
)

var buildEnvFlags = struct {
	// Common
	timeout     string
	concurrency int

	// Local
	push             config.BoolOrUndefined
	tryImportMissing config.BoolOrUndefined
	useDockerCLI     config.BoolOrUndefined
	useBuildkit      config.BoolOrUndefined

	// Google Cloud Build
	projectID          string
	diskSizeGb         int64
	machineType        string
	logging            string
	logStreamingOption string
	workerPool         string

	// Cluster (kaniko)
	pullSecretPath      string
	pullSecretName      string
	pullSecretMountPath string
	namespace           string

	dockerConfigPath       string
	dockerConfigSecretName string

	serviceAccount string
	runAsUser      int64

	randomPullSecret         bool
	randomDockerConfigSecret bool
}{}

func cmdBuildEnv() *cobra.Command {
	return NewCmd("build-env").
		WithDescription("Interact with skaffold build environment definitions.").
		WithPersistentFlagAdder(cmdBuildEnvFlags).
		WithCommands(cmdBuildEnvList(), cmdBuildEnvAdd(), cmdBuildEnvModify())
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
		WithCommands(cmdBuildEnvAddLocal(), cmdBuildEnvAddGcb(), cmdBuildEnvAddCluster())
}

func cmdBuildEnvModify() *cobra.Command {
	return NewCmd("modify").
		WithDescription("Modify the build environment for the default pipeline or an existing profile.").
		WithPersistentFlagAdder(cmdBuildEnvModifyFlags).
		WithCommands(cmdBuildEnvModifyGcb())
}

func cmdBuildEnvAddGcb() *cobra.Command {
	return NewCmd("googleCloudBuild").
		WithDescription("Add a new 'googleCloudBuild' build environment definition").
		WithLongDescription(`Add a new 'googleCloudBuild' build environment definition.
Without the '--profile' flag the new environment definition is added to the default pipeline. With the '--profile' flag it will create a new profile with this build env definition.
In these respective scenarios, it will fail if the build env definition for the default pipeline or the named profile already exists. To override an existing 'googleCloudBuild' definition use 'skaffold inspect build-env modify' command instead.
Use the '--module' filter to specify the individual module to target. Otherwise, it'll be applied to all modules defined in the target file. Also, with the '--profile' flag if the target config imports other configs as dependencies, then the new profile will be recursively created in all the imported configs also.`).
		WithExample("Add a new profile named 'gcb' targeting the builder 'googleCloudBuild' against the GCP project ID '1234'.", "inspect build-env add googleCloudBuild --profile gcb --projectID 1234 -f skaffold.yaml").
		WithFlagAdder(cmdBuildEnvGcbFlags).
		NoArgs(addGcbBuildEnv)
}

func cmdBuildEnvAddLocal() *cobra.Command {
	return NewCmd("local").
		WithDescription("Add a new 'local' build environment definition").
		WithLongDescription(`Add a new 'local' build environment definition.
Without the '--profile' flag the new environment definition is added to the default pipeline. With the '--profile' flag it will create a new profile with this build env definition.
In these respective scenarios, it will fail if the build env definition for the default pipeline or the named profile already exists. To override an existing 'local' build env definition use 'skaffold inspect build-env modify' command instead.
Use the '--module' filter to specify the individual module to target. Otherwise, it'll be applied to all modules defined in the target file. Also, with the '--profile' flag if the target config imports other configs as dependencies, then the new profile will be recursively created in all the imported configs also.`).
		WithExample("Add a new profile named 'local' targeting the local build environment with option to push images and using buildkit", "inspect build-env add local --profile local --push=true --useBuildkit=true -f skaffold.yaml").
		WithFlagAdder(cmdBuildEnvLocalFlags).
		NoArgs(addLocalBuildEnv)
}

func cmdBuildEnvAddCluster() *cobra.Command {
	return NewCmd("cluster").
		WithDescription("Add a new 'cluster' build environment definition").
		WithLongDescription(`Add a new 'cluster' build environment definition.
Without the '--profile' flag the new environment definition is added to the default pipeline. With the '--profile' flag it will create a new profile with this build env definition.
In these respective scenarios, it will fail if the build env definition for the default pipeline or the named profile already exists. To override an existing 'cluster' build env definition use 'skaffold inspect build-env modify' command instead.
Use the '--module' filter to specify the individual module to target. Otherwise, it'll be applied to all modules defined in the target file. Also, with the '--profile' flag if the target config imports other configs as dependencies, then the new profile will be recursively created in all the imported configs also.`).
		WithExample("Add a new profile named 'cluster' targeting the builder 'kaniko' using the Kubernetes secret 'my-secret'", "inspect build-env add cluster --profile cluster --pullSecretName my-secret -f skaffold.yaml").
		WithFlagAdder(cmdBuildEnvAddClusterFlags).
		NoArgs(addClusterBuildEnv)
}

func cmdBuildEnvModifyGcb() *cobra.Command {
	return NewCmd("googleCloudBuild").
		WithDescription("Modify an existing GoogleCloudBuild build environment definition").
		WithLongDescription(`Modify an existing GoogleCloudBuild build environment definition.
Without the '--profile' flag, the default pipeline's build environment definition is modified. With the '--profile' flag it'll modify the build environment definition for all profile pipelines that match the given profile name.
In these respective scenarios, it will fail if the build env definition for the default pipeline or the named profile is not GoogleCloudBuild. To create a new build environment definition use 'skaffold inspect build-env add' command instead.
Use the '--module' filter to specify the individual module to target. Otherwise, it'll be applied to all modules defined in the target file. Also, with the '--profile' flag if the target config imports other configs as dependencies, then it'll modify profiles that match the given name in all imported configs`).
		WithExample("Modify an existing profile named 'gcb' targeting the builder 'googleCloudBuild' against the GCP project ID '1234'.", "inspect build-env modify googleCloudBuild --profile gcb --projectID 1234 -f skaffold.yaml").
		WithFlagAdder(cmdBuildEnvGcbFlags).
		NoArgs(modifyGcbBuildEnv)
}

func listBuildEnv(ctx context.Context, out io.Writer) error {
	return buildEnv.PrintBuildEnvsList(ctx, out, printBuildEnvsListOptions())
}

func addLocalBuildEnv(ctx context.Context, out io.Writer) error {
	return buildEnv.AddLocalBuildEnv(ctx, out, localBuildEnvOptions())
}

func addGcbBuildEnv(ctx context.Context, out io.Writer) error {
	return buildEnv.AddGcbBuildEnv(ctx, out, gcbBuildEnvOptions())
}

func modifyGcbBuildEnv(ctx context.Context, out io.Writer) error {
	return buildEnv.ModifyGcbBuildEnv(ctx, out, gcbBuildEnvOptions())
}

func addClusterBuildEnv(ctx context.Context, out io.Writer) error {
	return buildEnv.AddClusterBuildEnv(ctx, out, addClusterBuildEnvOptions())
}

func cmdBuildEnvAddFlags(f *pflag.FlagSet) {
	f.StringVarP(&inspectFlags.profile, "profile", "p", "", `Profile name to add the new build env definition in. If the profile name doesn't exist then the profile will be created in all the target configs. If this flag is not specified then the build env is added to the default pipeline of the target configs.`)
}

func cmdBuildEnvLocalFlags(f *pflag.FlagSet) {
	var flags []*pflag.Flag
	flags = append(flags, f.VarPF(&buildEnvFlags.push, "push", "", `Set to true to push images to a registry`))
	flags = append(flags, f.VarPF(&buildEnvFlags.tryImportMissing, "tryImportMissing", "", `Set to true to to attempt importing artifacts from Docker (either a local or remote registry) if not in the build cache`))
	flags = append(flags, f.VarPF(&buildEnvFlags.useDockerCLI, "useDockerCLI", "", `Set to true to use 'docker' command-line interface instead of Docker Engine APIs`))
	flags = append(flags, f.VarPF(&buildEnvFlags.useBuildkit, "useBuildkit", "", `Set to true to use BuildKit to build Docker images`))
	f.IntVar(&buildEnvFlags.concurrency, "concurrency", -1, `number of artifacts to build concurrently. 0 means "no-limit"`)

	// support *bool flags without a value to be interpreted as `true`; like `--push` instead of `--push=true`
	for _, f := range flags {
		f.NoOptDefVal = "true"
	}
}

func cmdBuildEnvModifyFlags(f *pflag.FlagSet) {
	f.StringVarP(&inspectFlags.profile, "profile", "p", "", `If specified then the modify actions are performed on all pipelines from profiles that match flag value. It must match at least one existing profile name. If unspecified then only the default pipeline in the target 'skaffold.yaml' file are modified`)
	f.BoolVar(&inspectFlags.strict, "strict", true, "If set to 'false' then do not fail on mismatch of build env type or missing definitions. If set to 'true' then all default pipelines or those matching the `--profile` flag need to match the build env type exactly. Defaults to 'true'")
}

func cmdBuildEnvGcbFlags(f *pflag.FlagSet) {
	f.StringVar(&buildEnvFlags.projectID, "projectId", "", `ID of the Cloud Platform Project.`)
	f.Int64Var(&buildEnvFlags.diskSizeGb, "diskSizeGb", 0, `Disk size of the VM that runs the build`)
	f.StringVar(&buildEnvFlags.machineType, "machineType", "", `Type of VM that runs the build`)
	f.StringVar(&buildEnvFlags.timeout, "timeout", "", `Build timeout (in seconds)`)
	f.IntVar(&buildEnvFlags.concurrency, "concurrency", -1, `number of artifacts to build concurrently. 0 means "no-limit"`)
	f.StringVar(&buildEnvFlags.logging, "logging", "", `Specifies the logging mode for GCB`)
	f.StringVar(&buildEnvFlags.logStreamingOption, "logStreamingOption", "", `Specifies the log streaming specifies behavior when writing build logs to Google Cloud Storage for GCB`)
	f.StringVar(&buildEnvFlags.workerPool, "workerPool", "", `Configures a pool of workers to run the build`)
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

	f.BoolVar(&buildEnvFlags.randomPullSecret, "randomPullSecret", false, "Adds a random UUID postfix to the default name of the pull secret to facilitate parallel builds.")
	f.BoolVar(&buildEnvFlags.randomDockerConfigSecret, "randomDockerConfigSecret", false, "Adds a random UUID postfix to the default name of the docker secret to facilitate parallel builds.")
}

func cmdBuildEnvFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.modules, "module", "m", nil, "Names of modules to filter target action by.")
}

func cmdBuildEnvListFlags(f *pflag.FlagSet) {
	f.StringSliceVarP(&inspectFlags.profiles, "profile", "p", nil, `Profile names to activate`)
	f.BoolVar(&inspectFlags.propagateProfiles, "propagate-profiles", true, `Setting '--propagate-profiles=false' disables propagating profiles set by the '--profile' flag across config dependencies. This mean that only profiles defined directly in the target 'skaffold.yaml' file are activated.`)
}

func printBuildEnvsListOptions() inspect.Options {
	return inspect.Options{
		Filename:          inspectFlags.filename,
		RepoCacheDir:      inspectFlags.repoCacheDir,
		OutFormat:         inspectFlags.outFormat,
		Modules:           inspectFlags.modules,
		Profiles:          inspectFlags.profiles,
		PropagateProfiles: inspectFlags.propagateProfiles,
	}
}

func localBuildEnvOptions() inspect.Options {
	return inspect.Options{
		Filename:     inspectFlags.filename,
		RepoCacheDir: inspectFlags.repoCacheDir,
		OutFormat:    inspectFlags.outFormat,
		Modules:      inspectFlags.modules,
		Profile:      inspectFlags.profile,
		BuildEnvOptions: inspect.BuildEnvOptions{
			Push:             buildEnvFlags.push.Value(),
			TryImportMissing: buildEnvFlags.tryImportMissing.Value(),
			UseDockerCLI:     buildEnvFlags.useDockerCLI.Value(),
			UseBuildkit:      buildEnvFlags.useBuildkit.Value(),
			Concurrency:      buildEnvFlags.concurrency,
		},
	}
}

func gcbBuildEnvOptions() inspect.Options {
	return inspect.Options{
		Filename:  inspectFlags.filename,
		OutFormat: inspectFlags.outFormat,
		Modules:   inspectFlags.modules,
		Profile:   inspectFlags.profile,
		BuildEnvOptions: inspect.BuildEnvOptions{
			ProjectID:          buildEnvFlags.projectID,
			DiskSizeGb:         buildEnvFlags.diskSizeGb,
			MachineType:        buildEnvFlags.machineType,
			Timeout:            buildEnvFlags.timeout,
			Concurrency:        buildEnvFlags.concurrency,
			Logging:            buildEnvFlags.logging,
			LogStreamingOption: buildEnvFlags.logStreamingOption,
			WorkerPool:         buildEnvFlags.workerPool,
		},
		Strict: inspectFlags.strict,
	}
}

func addClusterBuildEnvOptions() inspect.Options {
	return inspect.Options{
		Filename:     inspectFlags.filename,
		RepoCacheDir: inspectFlags.repoCacheDir,
		OutFormat:    inspectFlags.outFormat,
		Modules:      inspectFlags.modules,
		Profile:      inspectFlags.profile,
		BuildEnvOptions: inspect.BuildEnvOptions{
			PullSecretPath:           buildEnvFlags.pullSecretPath,
			PullSecretName:           buildEnvFlags.pullSecretName,
			PullSecretMountPath:      buildEnvFlags.pullSecretMountPath,
			Namespace:                buildEnvFlags.namespace,
			DockerConfigPath:         buildEnvFlags.dockerConfigPath,
			DockerConfigSecretName:   buildEnvFlags.dockerConfigSecretName,
			ServiceAccount:           buildEnvFlags.serviceAccount,
			RunAsUser:                buildEnvFlags.runAsUser,
			RandomPullSecret:         buildEnvFlags.randomPullSecret,
			RandomDockerConfigSecret: buildEnvFlags.randomDockerConfigSecret,
			Timeout:                  buildEnvFlags.timeout,
			Concurrency:              buildEnvFlags.concurrency,
		},
	}
}
