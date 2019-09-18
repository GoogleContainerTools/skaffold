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
	"reflect"

	"github.com/spf13/pflag"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

// Flag defines a Skaffold CLI flag which contains a list of
// subcommands the flag belongs to in `DefinedOn` field.
type Flag struct {
	Name          string
	Shorthand     string
	Usage         string
	Value         interface{}
	DefValue      interface{}
	FlagAddMethod string
	DefinedOn     []string
}

// FlagRegistry is a list of all Skaffold CLI flags.
// When adding a new flag to the registry, please specify the
// command/commands to which the flag belongs in `DefinedOn` field.
// If the flag is a global flag, or belongs to all the subcommands,
/// specify "all"
// FlagAddMethod is method which defines a flag value with specified
// name, default value, and usage string. e.g. `StringVar`, `BoolVar`
var FlagRegistry = []Flag{
	{
		Name:          "filename",
		Shorthand:     "f",
		Usage:         "Filename or URL to the pipeline file",
		Value:         &opts.ConfigurationFile,
		DefValue:      "skaffold.yaml",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"all"},
	},
	{
		Name:          "profile",
		Shorthand:     "p",
		Usage:         "Activate profiles by name",
		Value:         &opts.Profiles,
		DefValue:      []string{},
		FlagAddMethod: "StringSliceVar",
		DefinedOn:     []string{"all"},
	},
	{
		Name:          "namespace",
		Shorthand:     "n",
		Usage:         "Run deployments in the specified namespace",
		Value:         &opts.Namespace,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"all"},
	},
	{
		Name:          "default-repo",
		Shorthand:     "d",
		Usage:         "Default repository value (overrides global config)",
		Value:         &opts.DefaultRepo,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"all"},
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
		Name:          "enable-rpc",
		Usage:         "Enable gRPC for exposing Skaffold events (true by default for `skaffold dev`)",
		Value:         &opts.EnableRPC,
		DefValue:      false,
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
		DefinedOn:     []string{"dev", "run", "debug", "deploy"},
	},
	{
		Name:          "toot",
		Usage:         "Emit a terminal beep after the deploy is complete",
		Value:         &opts.Notification,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "build", "run", "debug", "deploy"},
	},
	// We need opts.Tail and opts.TailDev since cobra, overwrites the default value
	// when registering the flag twice.
	{
		Name:          "tail",
		Usage:         "Stream logs from deployed objects (default false)",
		Value:         &opts.Tail,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"deploy", "run"},
	},
	{
		Name:          "tail",
		Usage:         "Stream logs from deployed objects",
		Value:         &opts.TailDev,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug"},
	},
	// We need opts.Force and opts.ForceDev since cobra, overwrites the default value
	// when registering the flag twice.
	{
		Name:          "force",
		Usage:         "Recreate kubernetes resources if necessary for deployment (default false, warning: might cause downtime!)",
		Value:         &opts.Force,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"deploy"},
	},
	{
		Name:          "force",
		Usage:         "Recreate kubernetes resources if necessary for deployment (warning: might cause downtime!)",
		Value:         &opts.ForceDev,
		DefValue:      true,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "run", "debug"},
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
		DefinedOn:     []string{"dev", "debug"},
	},
	{
		Name:          "status-check",
		Usage:         "Wait for deployed resources to stabilize",
		Value:         &opts.StatusCheck,
		DefValue:      false,
		FlagAddMethod: "BoolVar",
		DefinedOn:     []string{"dev", "debug", "deploy", "run"},
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
		Usage:         "Deploy to this kubernetes context",
		Value:         &opts.KubeContext,
		DefValue:      "",
		FlagAddMethod: "StringVar",
		DefinedOn:     []string{"build", "debug", "delete", "deploy", "dev", "run"},
	},
}

var commandFlags []*pflag.Flag

// SetupFlags creates pflag.Flag for all registered flags
func SetupFlags() {
	commandFlags = make([]*pflag.Flag, len(FlagRegistry))
	for i, fl := range FlagRegistry {
		fs := pflag.NewFlagSet(fl.Name, pflag.ContinueOnError)
		inputs := []reflect.Value{
			reflect.ValueOf(fl.Value),
			reflect.ValueOf(fl.Name),
			reflect.ValueOf(fl.DefValue),
			reflect.ValueOf(fl.Usage),
		}
		reflect.ValueOf(fs).MethodByName(fl.FlagAddMethod).Call(inputs)
		f := fs.Lookup(fl.Name)
		if fl.Shorthand != "" {
			f.Shorthand = fl.Shorthand
		}
		f.Annotations = map[string][]string{
			"cmds": fl.DefinedOn,
		}
		commandFlags[i] = f
	}
}

func AddFlags(fs *pflag.FlagSet, cmdName string) {
	for _, f := range commandFlags {
		if hasCmdAnnotation(cmdName, f.Annotations["cmds"]) {
			fs.AddFlag(f)
		}
	}
	fs.MarkHidden("status-check")
}

func hasCmdAnnotation(cmdName string, annotations []string) bool {
	for _, a := range annotations {
		if cmdName == a || a == "all" {
			return true
		}
	}
	return false
}

func init() {
	SetupFlags()
}
