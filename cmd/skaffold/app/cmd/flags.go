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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

// Flag defines a Skaffold CLI flag which contains a list of
// subcommands the flag belongs to in `DefinedOn` field.
type Flag struct {
	Name               string
	Shorthand          string
	Usage              string
	Value              interface{}
	DefValue           interface{}
	DefValuePerCommand map[string]interface{}
	FlagAddMethod      string
	DefinedOn          []string
	Hidden             bool

	pflag *pflag.Flag
}

// flagRegistry is a list of all Skaffold CLI flags.
// When adding a new flag to the registry, please specify the
// command/commands to which the flag belongs in `DefinedOn` field.
// If the flag is a global flag, or belongs to all the subcommands,
/// specify "all"
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
		Name:          "profile",
		Shorthand:     "p",
		Usage:         "Activate profiles by name (prefixed with `-` to disable a profile)",
		Value:         &opts.Profiles,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render", "build", "delete", "diagnose"},
	},
	{
		Name:          "namespace",
		Shorthand:     "n",
		Usage:         "Run deployments in the specified namespace",
		Value:         &opts.Namespace,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render", "build", "delete"},
	},
	{
		Name:          "default-repo",
		Shorthand:     "d",
		Usage:         "Default repository value (overrides global config)",
		Value:         &opts.DefaultRepo,
		DefValue:      "",
		FlagAddMethod: "Var",
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render", "build", "delete"},
	},
	{
		Name:          "cache-artifacts",
		Usage:         "Set to false to disable default caching of artifacts",
		Value:         &opts.CacheArtifacts,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "build", "run", "debug"},
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
		Name:          "insecure-registry",
		Usage:         "Target registries for built images which are not secure",
		Value:         &opts.InsecureRegistries,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"dev", "build", "run", "debug"},
	},
	{
		Name:     "enable-rpc",
		Usage:    "Enable gRPC for exposing Skaffold events (true by default for `skaffold dev`)",
		Value:    &opts.EnableRPC,
		DefValue: false,
		DefValuePerCommand: map[string]interface{}{
			"dev": true,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy"},
	},
	{
		Name:          "rpc-port",
		Usage:         "tcp port to expose event API",
		Value:         &opts.RPCPort,
		DefValue:      constants.DefaultRPCPort,
		FlagAddMethod: "IntVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy"},
	},
	{
		Name:          "rpc-http-port",
		Usage:         "tcp port to expose event REST API over HTTP",
		Value:         &opts.RPCHTTPPort,
		DefValue:      constants.DefaultRPCHTTPPort,
		FlagAddMethod: "IntVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy"},
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
	},
	{
		Name:     "tail",
		Usage:    "Stream logs from deployed objects (true by default for `skaffold dev` and `skaffold debug`)",
		Value:    &opts.Tail,
		DefValue: false,
		DefValuePerCommand: map[string]interface{}{
			"dev":   true,
			"debug": true,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug", "deploy"},
	},
	{
		Name:          "force",
		Usage:         "Recreate Kubernetes resources if necessary for deployment, warning: might cause downtime!",
		Value:         &opts.Force,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"deploy", "dev", "run", "debug"},
	},
	{
		Name:          "skip-tests",
		Usage:         "Whether to skip the tests after building",
		Value:         &opts.SkipTests,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug", "build"},
	},
	{
		Name:          "cleanup",
		Usage:         "Delete deployments after dev or debug mode is interrupted",
		Value:         &opts.Cleanup,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug"},
	},
	{
		Name:          "no-prune",
		Usage:         "Skip removing images and containers built by Skaffold",
		Value:         &opts.NoPrune,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug"},
	},
	{
		Name:          "no-prune-children",
		Usage:         "Skip removing layers reused by Skaffold",
		Value:         &opts.NoPruneChildren,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug"},
	},
	{
		Name:          "port-forward",
		Usage:         "Port-forward exposed container ports within pods",
		Value:         &opts.PortForward.Enabled,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug", "deploy", "run"},
	},
	{
		Name:          "status-check",
		Usage:         "Wait for deployed resources to stabilize",
		Value:         &opts.StatusCheck,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug", "deploy", "run"},
	},
	{
		Name:          "render-only",
		Usage:         "Print rendered Kubernetes manifests instead of deploying them",
		Value:         &opts.RenderOnly,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run"},
	},
	{
		Name:          "render-output",
		Usage:         "Writes '--render-only' output to the specified file",
		Value:         &opts.RenderOutput,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"run"},
	},
	{
		Name:          "config",
		Shorthand:     "c",
		Usage:         "File for global configurations (defaults to $HOME/.skaffold/config)",
		Value:         &opts.GlobalConfig,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"run", "dev", "debug", "build", "deploy", "delete", "diagnose"},
	},
	{
		Name:          "kube-context",
		Usage:         "Deploy to this Kubernetes context",
		Value:         &opts.KubeContext,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"build", "debug", "delete", "deploy", "dev", "run"},
	},
	{
		Name:          "kubeconfig",
		Usage:         "Path to the kubeconfig file to use for CLI requests.",
		Value:         &opts.KubeConfig,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"build", "debug", "delete", "deploy", "dev", "run"},
	},
	{
		Name:          "tag",
		Shorthand:     "t",
		Usage:         "The optional custom tag to use for images which overrides the current Tagger configuration",
		Value:         &opts.CustomTag,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"build", "debug", "dev", "run"},
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
		DefinedOn:     []string{"dev", "run", "debug", "deploy", "render", "build", "delete", "diagnose"},
	},
	{
		Name:          "trigger",
		Usage:         "How is change detection triggered? (polling, notify, or manual)",
		Value:         &opts.Trigger,
		DefValue:      "notify",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"dev", "debug"},
	},
	{
		Name:     "auto-build",
		Usage:    "When set to false, builds wait for API request instead of running automatically",
		Hidden:   true,
		Value:    &opts.AutoBuild,
		DefValue: true,
		DefValuePerCommand: map[string]interface{}{
			"dev":   true,
			"debug": false,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug"},
	},
	{
		Name:     "auto-sync",
		Usage:    "When set to false, syncs wait for API request instead of running automatically",
		Hidden:   true,
		Value:    &opts.AutoSync,
		DefValue: true,
		DefValuePerCommand: map[string]interface{}{
			"dev":   true,
			"debug": false,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug"},
	},
	{
		Name:     "auto-deploy",
		Usage:    "When set to false, deploys wait for API request instead of running automatically",
		Hidden:   true,
		Value:    &opts.AutoDeploy,
		DefValue: true,
		DefValuePerCommand: map[string]interface{}{
			"dev":   true,
			"debug": false,
		},
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug"},
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
		Name:          "add-skaffold-labels",
		Usage:         "Add Skaffold-specific labels to rendered manifest. If false, custom labels are still applied. Helpful for GitOps model where Skaffold is not the deployer.",
		Value:         &opts.AddSkaffoldLabels,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"render"},
	},
	{
		Name:          "suppress-logs",
		Usage:         "Suppress logs for specified stages in pipeline (build, deploy, status-check, none, all)",
		Value:         &opts.SuppressLogs,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"dev", "run", "debug", "build", "deploy"},
	},
	{
		Name:          "wait-for-deletions",
		Usage:         "Wait for pending deletions to complete before a deployment",
		Value:         &opts.WaitForDeletions.Enabled,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"deploy", "dev", "run", "debug"},
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
}

func (fl *Flag) flag() *pflag.Flag {
	if fl.pflag != nil {
		return fl.pflag
	}

	inputs := []interface{}{fl.Value, fl.Name}
	if fl.FlagAddMethod != "Var" {
		inputs = append(inputs, fl.DefValue)
	}
	inputs = append(inputs, fl.Usage)

	fs := pflag.NewFlagSet(fl.Name, pflag.ContinueOnError)

	reflect.ValueOf(fs).MethodByName(fl.FlagAddMethod).Call(reflectValueOf(inputs))
	f := fs.Lookup(fl.Name)
	f.Shorthand = fl.Shorthand
	f.Hidden = fl.Hidden

	fl.pflag = f
	return f
}

func reflectValueOf(values []interface{}) []reflect.Value {
	var results []reflect.Value
	for _, v := range values {
		results = append(results, reflect.ValueOf(v))
	}
	return results
}

// AddFlags adds to the command the common flags that are annotated with the command name.
func AddFlags(cmd *cobra.Command) {
	var flagsForCommand []*Flag

	for i := range flagRegistry {
		fl := &flagRegistry[i]
		if !hasCmdAnnotation(cmd.Use, fl.DefinedOn) {
			continue
		}

		cmd.Flags().AddFlag(fl.flag())

		flagsForCommand = append(flagsForCommand, fl)
	}

	// Apply command-specific default values to flags.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Update default values.
		for _, fl := range flagsForCommand {
			if defValue, present := fl.DefValuePerCommand[cmd.Use]; present {
				if flag := cmd.Flag(fl.Name); !flag.Changed {
					flag.Value.Set(fmt.Sprintf("%v", defValue))
				}
			}
		}

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
