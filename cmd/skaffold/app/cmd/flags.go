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
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var (
	AllFlags = []*flag.FlagSet{
		commonFlagSet("common"),
		buildFlagSet("build"),
		eventsAPIFlagSet("events"),
		deployPhaseFlagSet("deploy-phase"),
		deployCommandFlagSet("deploy-cmd"),
		testFlagSet("test"),
		cleanupFlagSet("cleanup"),
		devFlagSet("dev"),
	}
)

func commonFlagSet(name string) *flag.FlagSet {
	commonFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	commonFlags.StringVarP(&opts.ConfigurationFile, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	commonFlags.StringArrayVarP(&opts.Profiles, "profile", "p", nil, "Activate profiles by name")
	commonFlags.StringVarP(&opts.Namespace, "namespace", "n", "", "Run deployments in the specified namespace")
	commonFlags.StringVarP(&opts.DefaultRepo, "default-repo", "d", "", "Default repository value (overrides global config)")
	commonFlags.VisitAll(func(flag *flag.Flag) {
		commonFlags.SetAnnotation(flag.Name, "cmds", []string{"all"})
	})
	return commonFlags
}

func buildFlagSet(name string) *flag.FlagSet {
	buildFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	buildFlags.BoolVar(&opts.CacheArtifacts, "cache-artifacts", false, "Set to true to enable caching of artifacts.")
	buildFlags.StringVarP(&opts.CacheFile, "cache-file", "", "", "Specify the location of the cache file (default $HOME/.skaffold/cache)")
	buildFlags.StringArrayVar(&opts.InsecureRegistries, "insecure-registry", nil, "Target registries for built images which are not secure")
	buildFlags.VisitAll(func(flag *flag.Flag) {
		buildFlags.SetAnnotation(flag.Name, "cmds", []string{"dev", "run", "build", "debug"})
	})
	return buildFlags
}

func eventsAPIFlagSet(name string) *flag.FlagSet {
	eventFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	eventFlags.BoolVar(&opts.EnableRPC, "enable-rpc", false, "Enable gRPC for exposing Skaffold events (true by default for `skaffold dev`)")
	eventFlags.IntVar(&opts.RPCPort, "rpc-port", constants.DefaultRPCPort, "tcp port to expose event API")
	eventFlags.IntVar(&opts.RPCHTTPPort, "rpc-http-port", constants.DefaultRPCHTTPPort, "tcp port to expose event REST API over HTTP")
	eventFlags.VisitAll(func(flag *flag.Flag) {
		eventFlags.SetAnnotation(flag.Name, "cmds", []string{"dev", "run", "build", "deploy", "debug"})
	})
	return eventFlags
}

func deployPhaseFlagSet(name string) *flag.FlagSet {
	deployPhaseFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	deployPhaseFlags.StringArrayVarP(&opts.CustomLabels, "label", "l", nil, "Add custom labels to deployed objects. Set multiple times for multiple labels.")
	deployPhaseFlags.BoolVar(&opts.Notification, "toot", false, "Emit a terminal beep after the deploy is complete")
	deployPhaseFlags.VisitAll(func(flag *flag.Flag) {
		deployPhaseFlags.SetAnnotation(flag.Name, "cmds", []string{"dev", "run", "deploy", "debug"})
	})
	return deployPhaseFlags
}

func deployCommandFlagSet(name string) *flag.FlagSet {
	deployCommandFlags := flag.NewFlagSet("deploy-command", flag.ContinueOnError)
	deployCommandFlags.BoolVar(&opts.Tail, "tail", false, "Stream logs from deployed objects")
	deployCommandFlags.BoolVar(&opts.Force, "force", false, "Recreate kubernetes resources if necessary for deployment (default: false, warning: might cause downtime!)")
	deployCommandFlags.VisitAll(func(flag *flag.Flag) {
		deployCommandFlags.SetAnnotation(flag.Name, "cmds", []string{"deploy"})
	})
	return deployCommandFlags
}

func devFlagSet(name string) *flag.FlagSet {
	devFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	devFlags.BoolVar(&opts.TailDev, "tail", true, "Stream logs from deployed objects")
	devFlags.BoolVar(&opts.ForceDev, "force", true, "Recreate kubernetes resources if necessary for deployment (default: false, warning: might cause downtime!)")
	devFlags.VisitAll(func(flag *flag.Flag) {
		devFlags.SetAnnotation(flag.Name, "cmds", []string{"dev", "run", "debug"})
	})
	return devFlags
}

func testFlagSet(name string) *flag.FlagSet {
	testFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	testFlags.BoolVar(&opts.SkipTests, "skip-tests", false, "Whether to skip the tests after building")
	testFlags.VisitAll(func(flag *flag.Flag) {
		testFlags.SetAnnotation(flag.Name, "cmds", []string{"dev", "run", "debug", "build"})
	})
	return testFlags
}

func cleanupFlagSet(name string) *flag.FlagSet {
	cleanupFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	cleanupFlags.BoolVar(&opts.Cleanup, "cleanup", true, "Delete deployments after dev or debug mode is interrupted")
	cleanupFlags.BoolVar(&opts.NoPrune, "no-prune", false, "Skip removing images and containers built by Skaffold")
	cleanupFlags.VisitAll(func(flag *flag.Flag) {
		cleanupFlags.SetAnnotation(flag.Name, "cmds", []string{"dev", "run", "debug"})
	})
	return cleanupFlags
}

func AddFlags(cmd *cobra.Command) {
	for _, flagSet := range AllFlags {
		flagSet.VisitAll(func(flag *flag.Flag) {
			if hasCmdAnnotation(cmd.Use, flag.Annotations["cmds"]) {
				cmd.Flags().AddFlag(flag)
			}
		})
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
