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

const (
	BuildAnnotation  = "build"
	DevAnnotation    = "dev"
	DeployAnnotation = "deploy"
	TestAnnotation   = "test"
	DeleteAnnotation = "delete"
	DebugAnnotation  = "debug"
	EventsAnnotation = "events"
)

var (
	CommandFlags = commandFlagSet("common")

	AnnotationToFlag = map[string]*flag.FlagSet{
		DevAnnotation:    devFlagSet(DevAnnotation),
		BuildAnnotation:  buildFlagSet(BuildAnnotation),
		EventsAnnotation: eventsAPIFlagSet(EventsAnnotation),
		DeployAnnotation: deployFlagSet(DeployAnnotation),
		TestAnnotation:   testFlagSet(TestAnnotation),
		DebugAnnotation:  debugFlagSet(DebugAnnotation),
	}
)

func commandFlagSet(name string) *flag.FlagSet {
	commonFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	commonFlags.StringVarP(&opts.ConfigurationFile, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	commonFlags.StringArrayVarP(&opts.Profiles, "profile", "p", nil, "Activate profiles by name")
	commonFlags.StringVarP(&opts.Namespace, "namespace", "n", "", "Run deployments in the specified namespace")
	commonFlags.StringVarP(&opts.DefaultRepo, "default-repo", "d", "", "Default repository value (overrides global config)")
	return commonFlags
}

func devFlagSet(name string) *flag.FlagSet {
	devFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	devFlags.BoolVar(&opts.TailDev, "tail", true, "Stream logs from deployed objects")
	devFlags.BoolVar(&opts.NoPrune, "no-prune", false, "Skip removing images and containers built by Skaffold")
	return devFlags
}

func buildFlagSet(name string) *flag.FlagSet {
	buildFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	buildFlags.BoolVar(&opts.CacheArtifacts, "cache-artifacts", false, "Set to true to enable caching of artifacts.")
	buildFlags.StringVarP(&opts.CacheFile, "cache-file", "", "", "Specify the location of the cache file (default $HOME/.skaffold/cache)")
	buildFlags.StringArrayVar(&opts.InsecureRegistries, "insecure-registry", nil, "Target registries for built images which are not secure")
	return buildFlags
}

func eventsAPIFlagSet(name string) *flag.FlagSet {
	eventFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	eventFlags.BoolVar(&opts.EnableRPC, "enable-rpc", false, "Enable gRPC for exposing Skaffold events (true by default for `skaffold dev`)")
	eventFlags.IntVar(&opts.RPCPort, "rpc-port", constants.DefaultRPCPort, "tcp port to expose event API")
	eventFlags.IntVar(&opts.RPCHTTPPort, "rpc-http-port", constants.DefaultRPCHTTPPort, "tcp port to expose event REST API over HTTP")
	return eventFlags
}

func deployFlagSet(name string) *flag.FlagSet {
	deployFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	deployFlags.BoolVar(&opts.Tail, "tail", false, "Stream logs from deployed objects")
	deployFlags.BoolVar(&opts.Force, "force", false, "Recreate kubernetes resources if necessary for deployment (default: false, warning: might cause downtime!)")
	deployFlags.StringArrayVarP(&opts.CustomLabels, "label", "l", nil, "Add custom labels to deployed objects. Set multiple times for multiple labels.")
	deployFlags.BoolVar(&opts.Notification, "toot", false, "Emit a terminal beep after the deploy is complete")
	return deployFlags
}

func testFlagSet(name string) *flag.FlagSet {
	testFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	testFlags.BoolVar(&opts.SkipTests, "skip-tests", false, "Whether to skip the tests after building")
	return testFlags
}

func debugFlagSet(name string) *flag.FlagSet {
	debugFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	debugFlags.BoolVar(&opts.Cleanup, "cleanup", true, "Delete deployments after dev or debug mode is interrupted")
	debugFlags.BoolVar(&opts.PortForward, "port-forward", true, "Port-forward exposed container ports within pods")
	debugFlags.BoolVar(&opts.NoPrune, "no-prune", false, "Skip removing images and containers built by Skaffold")
	return debugFlags
}

func AddFlags(cmd *cobra.Command) {
	cmd.Flags().AddFlagSet(CommandFlags)
	cmd.Flags().AddFlagSet(getAnnotatedFlags(cmd.Use, cmd.Annotations))
}

func getAnnotatedFlags(name string, annotations map[string]string) *flag.FlagSet {
	allFlags := flag.NewFlagSet(name, flag.ContinueOnError)
	for a := range annotations {
		flags, ok := AnnotationToFlag[a]
		if ok {
			allFlags.AddFlagSet(flags)
		}
	}
	return allFlags
}
