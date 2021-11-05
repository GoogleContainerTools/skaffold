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

package cmd

import (
	"fmt"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/flags"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
)

var (
	fromBuildOutputFile flags.BuildOutputFileFlag
)

// Nillable is used to reset objects that implement pflag's `Value` and `SliceValue`.
// Some flags, like `--default-repo`, use nil to indicate that they are unset, which
// is different from the empty string.
type Nillable interface {
	SetNil() error
}

// Flag defines a Skaffold CLI flag which contains a list of
// subcommands the flag belongs to in `DefinedOn` field.
// See https://pkg.go.dev/github.com/spf13/pflag#Flag
type Flag struct {
	Name                 string
	Shorthand            string
	Usage                string
	Value                interface{}
	DefValue             interface{}
	DefValuePerCommand   map[string]interface{}
	DeprecatedPerCommand map[string]interface{}
	NoOptDefVal          string
	FlagAddMethod        string
	Deprecated           string
	DefinedOn            []string
	Hidden               bool
	IsEnum               bool
}

// flagRegistry is a list of all Skaffold CLI flags.
// When adding a new flag to the registry, please specify the
// command/commands to which the flag belongs in `DefinedOn` field.
// If the flag is a global flag, or belongs to all the subcommands,
// specify "all"
// FlagAddMethod is method which defines a flag value with specified
// name, default value, and usage string. e.g. `StringVar`, `BoolVar`
var flagRegistry = []Flag{
	{
		Name:          "filename",
		Shorthand:     "f",
		Usage:         "Path or URL to the Skaffold config file",
		Value:         &opts.ConfigurationFile,
		DefValue:      "skaffold.yaml",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"all"},
	},
	{
		Name:          "module",
		Shorthand:     "m",
		Usage:         "Filter Skaffold configs to only the provided named modules",
		Value:         &opts.ConfigurationFilter,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"all"},
	},
	{
		Name:          "user",
		Shorthand:     "u",
		Value:         &opts.User,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		Hidden:        true,
		DefinedOn:     []string{"all"},
	},
	{
		Name:          "profile",
		Shorthand:     "p",
		Usage:         "Activate profiles by name (prefixed with `-` to disable a profile)",
		Value:         &opts.Profiles,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render", "build", "delete", "diagnose", "apply", "test"},
	},
	{
		Name:          "namespace",
		Shorthand:     "n",
		Usage:         "Run deployments in the specified namespace",
		Value:         &opts.Namespace,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render", "build", "delete", "apply"},
	},
	{
		Name:          "default-repo",
		Shorthand:     "d",
		Usage:         "Default repository value (overrides global config)",
		Value:         &opts.DefaultRepo,
		DefValue:      nil,
		FlagAddMethod: "Var",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render", "build", "delete"},
	},
	{
		Name:          "cache-artifacts",
		Usage:         "Set to false to disable default caching of artifacts",
		Value:         &opts.CacheArtifacts,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "render"},
		IsEnum:        true,
	},
	{
		Name:          "cache-file",
		Usage:         "Specify the location of the cache file (default $HOME/.skaffold/cache)",
		Value:         &opts.CacheFile,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"dev", "build", "run", "debug"},
	},
	{
		Name:          "remote-cache-dir",
		Usage:         "Specify the location of the git repositories cache (default $HOME/.skaffold/repos)",
		Value:         &opts.RepoCacheDir,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"all"},
	},
	{
		Name:          "insecure-registry",
		Usage:         "Target registries for built images which are not secure",
		Value:         &opts.InsecureRegistries,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"dev", "build", "run", "debug"},
	},
	{
		Name:     "enable-rpc",
		Usage:    "Enable gRPC for exposing Skaffold events",
		Value:    &opts.EnableRPC,
		DefValue: false,
		DefValuePerCommand: map[string]interface{}{
			"dev":   true,
			"debug": true,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy", "render", "apply", "test"},
		IsEnum:        true,
	},
	{
		Name:          "event-log-file",
		Usage:         "Save Skaffold events to the provided file after skaffold has finished executing, requires --enable-rpc=true",
		Hidden:        true,
		Value:         &opts.EventLogFile,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy", "render", "test", "apply"},
	},
	{
		Name:          "rpc-port",
		Usage:         "tcp port to expose event API",
		Value:         &opts.RPCPort,
		DefValue:      constants.DefaultRPCPort,
		FlagAddMethod: "IntVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy", "test"},
	},
	{
		Name:          "rpc-http-port",
		Usage:         "tcp port to expose event REST API over HTTP",
		Value:         &opts.RPCHTTPPort,
		DefValue:      constants.DefaultRPCHTTPPort,
		FlagAddMethod: "IntVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy", "test"},
	},
	{
		Name:          "label",
		Shorthand:     "l",
		Usage:         "Add custom labels to deployed objects. Set multiple times for multiple labels",
		Value:         &opts.CustomLabels,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render"},
	},
	{
		Name:          "toot",
		Usage:         "Emit a terminal beep after the deploy is complete",
		Value:         &opts.Notification,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy"},
		IsEnum:        true,
	},
	{
		Name:     "tail",
		Usage:    "Stream logs from deployed objects",
		Value:    &opts.Tail,
		DefValue: false,
		DefValuePerCommand: map[string]interface{}{
			"dev":   true,
			"debug": true,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "apply"},
		IsEnum:        true,
	},
	{
		Name:          "force",
		Usage:         "Recreate Kubernetes resources if necessary for deployment, warning: might cause downtime!",
		Value:         &opts.Force,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"deploy", "dev", "run", "debug", "apply"},
		IsEnum:        true,
	},
	{
		Name:          "skip-tests",
		Usage:         "Whether to skip the tests after building",
		Value:         &opts.SkipTests,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug", "build"},
		IsEnum:        true,
	},
	{
		Name:          "cleanup",
		Usage:         "Delete deployments after dev or debug mode is interrupted",
		Value:         &opts.Cleanup,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug"},
		IsEnum:        true,
	},
	{
		Name:          "no-prune",
		Usage:         "Skip removing images and containers built by Skaffold",
		Value:         &opts.NoPrune,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug"},
		IsEnum:        true,
	},
	{
		Name:          "no-prune-children",
		Usage:         "Skip removing layers reused by Skaffold",
		Value:         &opts.NoPruneChildren,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug"},
		IsEnum:        true,
	},
	{
		Name:     "port-forward",
		Usage:    "Port-forward exposes service ports and container ports within pods and other resources (off, user, services, debug, pods)",
		Value:    &opts.PortForward,
		DefValue: []string{"off"},
		DefValuePerCommand: map[string]interface{}{
			"debug": []string{"user", "debug"},
			"dev":   []string{"user"},
		},
		NoOptDefVal:   "true", // uses the settings from when --port-forward was boolean
		FlagAddMethod: "Var",
		DefinedOn:     []string{"dev", "run", "deploy", "debug"},
		IsEnum:        true,
	},
	{
		Name:          "status-check",
		Usage:         "Wait for deployed resources to stabilize",
		Value:         &opts.StatusCheck,
		DefValue:      true,
		FlagAddMethod: "Var",
		DefinedOn:     []string{"dev", "debug", "deploy", "run", "apply"},
		IsEnum:        true,
		NoOptDefVal:   "true",
	},
	{
		Name:          "iterative-status-check",
		Usage:         "Run `status-check` iteratively after each deploy step, instead of all-together at the end of all deploys (default).",
		Value:         &opts.IterativeStatusCheck,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug", "deploy", "run", "apply"},
		IsEnum:        true,
	},
	{
		Name:          "render-only",
		Usage:         "Print rendered Kubernetes manifests instead of deploying them",
		Value:         &opts.RenderOnly,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run"},
		Deprecated:    "please use the `skaffold render` command instead.",
		IsEnum:        true,
	},
	{
		Name:          "render-output",
		Usage:         "Writes '--render-only' output to the specified file",
		Value:         &opts.RenderOutput,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"run"},
		Deprecated:    "please use the `skaffold render` command instead.",
	},
	{
		Name:          "config",
		Shorthand:     "c",
		Usage:         "File for global configurations (defaults to $HOME/.skaffold/config)",
		Value:         &opts.GlobalConfig,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"run", "dev", "debug", "build", "deploy", "delete", "diagnose", "apply", "test"},
	},
	{
		Name:          "kube-context",
		Usage:         "Deploy to this Kubernetes context",
		Value:         &opts.KubeContext,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"build", "debug", "delete", "deploy", "dev", "run", "filter", "apply"},
	},
	{
		Name:          "kubeconfig",
		Usage:         "Path to the kubeconfig file to use for CLI requests.",
		Value:         &opts.KubeConfig,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"build", "debug", "delete", "deploy", "dev", "run", "filter", "apply"},
	},
	{
		Name:          "tag",
		Shorthand:     "t",
		Usage:         "The optional custom tag to use for images which overrides the current Tagger configuration",
		Value:         &opts.CustomTag,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"build", "debug", "dev", "run", "deploy"},
	},
	{
		Name:          "minikube-profile",
		Usage:         "forces skaffold use the given minikube-profile and forces building against the docker daemon inside that minikube profile",
		Value:         &opts.MinikubeProfile,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"build", "debug", "dev", "run"},
		// this is a temporary solution until we figure out an automated way to detect the
		// minikube profile see
		// https://github.com/GoogleContainerTools/skaffold/issues/3668
		Hidden: true,
	},
	{
		Name:          "profile-auto-activation",
		Usage:         "Set to false to disable profile auto activation",
		Value:         &opts.ProfileAutoActivation,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render", "build", "delete", "diagnose", "test"},
		IsEnum:        true,
	},
	{
		Name:          "propagate-profiles",
		Usage:         "Setting '--propagate-profiles=false' disables propagating profiles set by the '--profile' flag across config dependencies. This mean that only profiles defined directly in the target 'skaffold.yaml' file are activated.",
		Value:         &opts.PropagateProfiles,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render", "build", "delete", "diagnose", "test"},
		IsEnum:        true,
	},
	{
		Name:          "trigger",
		Usage:         "How is change detection triggered? (polling, notify, or manual)",
		Value:         &opts.Trigger,
		DefValue:      "notify",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"dev", "debug"},
		IsEnum:        true,
	},
	{
		Name:     "auto-build",
		Usage:    "When set to false, builds wait for API request instead of running automatically",
		Value:    &opts.AutoBuild,
		DefValue: true,
		DefValuePerCommand: map[string]interface{}{
			"dev":   true,
			"debug": false,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug"},
		IsEnum:        true,
	},
	{
		Name:     "auto-sync",
		Usage:    "When set to false, syncs wait for API request instead of running automatically",
		Value:    &opts.AutoSync,
		DefValue: true,
		DefValuePerCommand: map[string]interface{}{
			"dev":   true,
			"debug": false,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug"},
		IsEnum:        true,
	},
	{
		Name:     "auto-deploy",
		Usage:    "When set to false, deploys wait for API request instead of running automatically",
		Value:    &opts.AutoDeploy,
		DefValue: true,
		DefValuePerCommand: map[string]interface{}{
			"dev":   true,
			"debug": false,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug"},
		IsEnum:        true,
	},
	{
		Name:          "watch-image",
		Shorthand:     "w",
		Usage:         "Choose which artifacts to watch. Artifacts with image names that contain the expression will be watched only. Default is to watch sources for all artifacts",
		Value:         &opts.TargetImages,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"dev", "debug"},
	},
	{
		Name:          "watch-poll-interval",
		Shorthand:     "i",
		Usage:         "Interval (in ms) between two checks for file changes",
		Value:         &opts.WatchPollInterval,
		DefValue:      1000,
		FlagAddMethod: "IntVar",
		DefinedOn:     []string{"dev", "debug"},
	},
	{
		Name:     "add-skaffold-labels",
		Usage:    "Add Skaffold-specific labels to rendered manifest. Custom labels will still be applied. Helpful for GitOps model where Skaffold is not the deployer.",
		Value:    &opts.AddSkaffoldLabels,
		DefValue: true,
		DefValuePerCommand: map[string]interface{}{
			"render": false,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug", "render", "run"},
		IsEnum:        true,
		Deprecated:    "Adding Skaffold-specific labels in `render` is deprecated.",
	},
	{
		Name:          "mute-logs",
		Usage:         "mute logs for specified stages in pipeline (build, deploy, status-check, none, all)",
		Value:         &opts.Muted.Phases,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"dev", "run", "debug", "build", "deploy"},
		IsEnum:        true,
	},
	{
		Name:          "wait-for-deletions",
		Usage:         "Wait for pending deletions to complete before a deployment",
		Value:         &opts.WaitForDeletions.Enabled,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"deploy", "dev", "run", "debug"},
		IsEnum:        true,
	},
	{
		Name:          "wait-for-deletions-max",
		Usage:         "Max duration to wait for pending deletions",
		Value:         &opts.WaitForDeletions.Max,
		DefValue:      60 * time.Second,
		FlagAddMethod: "DurationVar",
		DefinedOn:     []string{"deploy", "dev", "run", "debug"},
	},
	{
		Name:          "wait-for-deletions-delay",
		Usage:         "Delay between two checks for pending deletions",
		Value:         &opts.WaitForDeletions.Delay,
		DefValue:      2 * time.Second,
		FlagAddMethod: "DurationVar",
		DefinedOn:     []string{"deploy", "dev", "run", "debug"},
	},
	{
		Name:          "build-image",
		Shorthand:     "b",
		Usage:         "Only build artifacts with image names that contain the given substring. Default is to build sources for all artifacts",
		Value:         &opts.TargetImages,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"build", "run"},
	},
	{
		Name:          "detect-minikube",
		Usage:         "Use heuristics to detect a minikube cluster",
		Value:         &opts.DetectMinikube,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"build", "debug", "delete", "deploy", "dev", "run"},
		IsEnum:        true,
	},
	{
		Name:          "build-artifacts",
		Shorthand:     "a",
		Usage:         "File containing build result from a previous 'skaffold build --file-output'",
		Value:         &fromBuildOutputFile,
		DefValue:      "",
		FlagAddMethod: "Var",
		DefinedOn:     []string{"test", "deploy"},
	},
	{
		Name:          "auto-create-config",
		Usage:         "If true, skaffold will try to create a config for the user's run if it doesn't find one",
		Value:         &opts.AutoCreateConfig,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"debug", "dev", "run"},
		IsEnum:        true,
	},
	{
		Name:          "assume-yes",
		Usage:         "If true, skaffold will skip yes/no confirmation from the user and default to yes",
		Value:         &opts.AssumeYes,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"all"},
		IsEnum:        true,
	},
	{
		Name:          "build-concurrency",
		Usage:         "Number of concurrently running builds. Set to 0 to run all builds in parallel. Doesn't violate build order among dependencies.",
		Value:         &opts.BuildConcurrency,
		DefValue:      -1,
		FlagAddMethod: "IntVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy"},
	},
	{
		Name:          "digest-source",
		Usage:         "Set to 'remote' to skip builds and resolve the digest of images by tag from the remote registry. Set to 'local' to build images locally and use digests from built images. Set to 'tag' to use tags directly from the build. Set to 'none' to use tags directly from the Kubernetes manifests.",
		Value:         &opts.DigestSource,
		DefValue:      "remote",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"dev", "render", "run"},
		DeprecatedPerCommand: map[string]interface{}{
			"dev": true,
			"run": true,
		},
		IsEnum: true,
	},
	{
		Name: "hydration-dir",
		Usage: fmt.Sprintf("the directory to where the (kpt) hydration takes place. "+
			"Default to a hidden directory %s.", constants.DefaultHydrationDir),
		Value:         &opts.HydrationDir,
		DefValue:      constants.DefaultHydrationDir,
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"dev", "render", "run", "debug", "deploy"},
	},
	{
		Name:          "inventory-namespace",
		Usage:         "The namespace for the ResourceGroup resource that contains the inventory",
		Value:         &opts.InventoryNamespace,
		Hidden:        true,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"all"},
	},
	{
		Name: "inventory-id",
		Usage: "Inventory identifier, should be generated by previous kpt live. " +
			"Do not use custom value",
		Value:         &opts.InventoryID,
		Hidden:        true,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"all"},
	},
	{
		Name: "inventory-name",
		Usage: "Inventory name in the form of 'inventory-XXX', should be generated by previous kpt live." +
			" Do not use custom value",
		Value:         &opts.InventoryName,
		Hidden:        true,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"all"},
	},
}

func methodNameByType(v reflect.Value) string {
	t := v.Type().Kind()
	switch t {
	case reflect.Bool:
		return "BoolVar"
	case reflect.String:
		return "StringVar"
	case reflect.Slice:
		return "StringSliceVar"
	case reflect.Struct:
		return "Var"
	case reflect.Ptr:
		return methodNameByType(reflect.Indirect(v))
	}
	return ""
}

func (fl *Flag) flag(cmdName string) *pflag.Flag {
	methodName := fl.FlagAddMethod
	if methodName == "" {
		methodName = methodNameByType(reflect.ValueOf(fl.Value))
	}
	isVar := methodName == "Var"
	// pflags' Var*() methods do not take a default value but instead
	// assume the value is already set to its default value.  So we
	// explicitly set the default value here to ensure help text is correct.
	if isVar {
		setDefaultValues(fl.Value, fl, cmdName)
	}

	inputs := []interface{}{fl.Value, fl.Name}
	if !isVar {
		if d, found := fl.DefValuePerCommand[cmdName]; found {
			inputs = append(inputs, d)
		} else {
			inputs = append(inputs, fl.DefValue)
		}
	}
	inputs = append(inputs, fl.Usage)

	fs := pflag.NewFlagSet(fl.Name, pflag.ContinueOnError)
	reflect.ValueOf(fs).MethodByName(methodName).Call(reflectValueOf(inputs))

	f := fs.Lookup(fl.Name)
	if len(fl.NoOptDefVal) > 0 {
		// f.NoOptDefVal may be set depending on value type
		f.NoOptDefVal = fl.NoOptDefVal
	}
	f.Shorthand = fl.Shorthand
	f.Hidden = fl.Hidden || (fl.Deprecated != "")
	f.Deprecated = fl.Deprecated

	// Deprecations can be applied per command
	if _, found := fl.DeprecatedPerCommand[cmdName]; found {
		f.Deprecated = fl.Deprecated
	}
	return f
}

func ResetFlagDefaults(cmd *cobra.Command, flags []*Flag) {
	// Update default values.
	for _, fl := range flags {
		flag := cmd.Flag(fl.Name)
		if !flag.Changed {
			setDefaultValues(flag.Value, fl, cmd.Name())
		}
		if fl.IsEnum {
			instrumentation.AddFlag(flag)
		}
	}
}

// setDefaultValues sets the default value (or values) for the given flag definition.
// This function handles pflag's SliceValue and Value interfaces.
func setDefaultValues(v interface{}, fl *Flag, cmdName string) {
	d, found := fl.DefValuePerCommand[cmdName]
	if !found {
		d = fl.DefValue
	}
	if nv, ok := v.(Nillable); ok && d == nil {
		nv.SetNil()
	} else if sv, ok := v.(pflag.SliceValue); ok {
		sv.Replace(asStringSlice(d))
	} else if val, ok := v.(pflag.Value); ok {
		val.Set(fmt.Sprintf("%v", d))
	} else {
		logrus.Fatalf("%s --%s: unhandled value type: %v (%T)", cmdName, fl.Name, v, v)
	}
}

// AddFlags adds to the command the common flags that are annotated with the command name.
func AddFlags(cmd *cobra.Command) {
	var flagsForCommand []*Flag

	for i := range flagRegistry {
		fl := &flagRegistry[i]
		if !hasCmdAnnotation(cmd.Use, fl.DefinedOn) {
			continue
		}

		cmd.Flags().AddFlag(fl.flag(cmd.Use))

		flagsForCommand = append(flagsForCommand, fl)
	}

	// Apply command-specific default values to flags.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ResetFlagDefaults(cmd, flagsForCommand)
		// Since PersistentPreRunE replaces the parent's PersistentPreRunE,
		// make sure we call it, if it is set.
		if parent := cmd.Parent(); parent != nil {
			if preRun := parent.PersistentPreRunE; preRun != nil {
				if err := preRun(cmd, args); err != nil {
					return err
				}
			} else if preRun := parent.PersistentPreRun; preRun != nil {
				preRun(cmd, args)
			}
		}

		return nil
	}
}

func hasCmdAnnotation(cmdName string, annotations []string) bool {
	for _, a := range annotations {
		if cmdName == a || a == "all" {
			return true
		}
	}
	return false
}

func reflectValueOf(values []interface{}) []reflect.Value {
	var results []reflect.Value
	for _, v := range values {
		results = append(results, reflect.ValueOf(v))
	}
	return results
}

func asStringSlice(v interface{}) []string {
	vt := reflect.TypeOf(v)
	if vt == reflect.TypeOf([]string{}) {
		return v.([]string)
	}
	switch vt.Kind() {
	case reflect.Array, reflect.Slice:
		value := reflect.ValueOf(v)
		var slice []string
		for i := 0; i < value.Len(); i++ {
			slice = append(slice, fmt.Sprintf("%v", value.Index(i)))
		}
		return slice
	default:
		return []string{fmt.Sprintf("%v", v)}
	}
}
