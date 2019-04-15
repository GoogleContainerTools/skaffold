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
	"io"
	"os"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/update"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	opts         = &config.SkaffoldOptions{}
	v            string
	defaultColor int
	overwrite    bool
)

func NewSkaffoldCommand(out, err io.Writer) *cobra.Command {
	updateMsg := make(chan string)

	rootCmd := &cobra.Command{
		Use:           "skaffold",
		Short:         "A tool that facilitates continuous development for Kubernetes applications.",
		SilenceErrors: true,
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		opts.Command = cmd.Use

		if err := SetUpLogs(err, v); err != nil {
			return err
		}

		rootCmd.SilenceUsage = true
		logrus.Infof("Skaffold %+v", version.Get())
		color.OverwriteDefault(color.Color(defaultColor))

		if quietFlag {
			logrus.Debugf("Update check is disabled because of quiet mode")
		} else {
			go func() {
				if err := updateCheck(updateMsg); err != nil {
					logrus.Infof("update check failed: %s", err)
				}
			}()
		}

		return nil
	}

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		select {
		case msg := <-updateMsg:
			fmt.Fprintf(out, "%s\n", msg)
		default:
		}
	}

	rootCmd.SetOutput(out)
	rootCmd.AddCommand(NewCmdCompletion(out))
	rootCmd.AddCommand(NewCmdVersion(out))
	rootCmd.AddCommand(NewCmdRun(out))
	rootCmd.AddCommand(NewCmdDev(out))
	rootCmd.AddCommand(NewCmdDebug(out))
	rootCmd.AddCommand(NewCmdBuild(out))
	rootCmd.AddCommand(NewCmdDeploy(out))
	rootCmd.AddCommand(NewCmdDelete(out))
	rootCmd.AddCommand(NewCmdFix(out))
	rootCmd.AddCommand(NewCmdConfig(out))
	rootCmd.AddCommand(NewCmdInit(out))
	rootCmd.AddCommand(NewCmdDiagnose(out))

	rootCmd.PersistentFlags().StringVarP(&v, "verbosity", "v", constants.DefaultLogLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().IntVar(&defaultColor, "color", int(color.Default), "Specify the default output color in ANSI escape codes")

	setFlagsFromEnvVariables(rootCmd.Commands())

	return rootCmd
}

func updateCheck(ch chan string) error {
	if !update.IsUpdateCheckEnabled() {
		logrus.Debugf("Update check not enabled, skipping.")
		return nil
	}
	latest, current, err := update.GetLatestAndCurrentVersion()
	if err != nil {
		return errors.Wrap(err, "get latest and current Skaffold version")
	}
	if latest.GT(current) {
		ch <- fmt.Sprintf("There is a new version (%s) of Skaffold available. Download it at %s\n", latest, constants.LatestDownloadURL)
	}
	return nil
}

// Each flag can also be set with an env variable whose name starts with `SKAFFOLD_`.
func setFlagsFromEnvVariables(commands []*cobra.Command) {
	for _, cmd := range commands {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			// special case for backward compatibility.
			if f.Name == "namespace" {
				if val, present := os.LookupEnv("SKAFFOLD_DEPLOY_NAMESPACE"); present {
					logrus.Warnln("Using SKAFFOLD_DEPLOY_NAMESPACE env variable is deprecated. Please use SKAFFOLD_NAMESPACE instead.")
					cmd.Flags().Set(f.Name, val)
				}
			}

			envVar := FlagToEnvVarName(f)
			if val, present := os.LookupEnv(envVar); present {
				cmd.Flags().Set(f.Name, val)
			}
		})
	}
}

func FlagToEnvVarName(f *pflag.Flag) string {
	return fmt.Sprintf("SKAFFOLD_%s", strings.Replace(strings.ToUpper(f.Name), "-", "_", -1))
}

func AddRunDeployFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&opts.Tail, "tail", false, "Stream logs from deployed objects")
	cmd.Flags().StringArrayVarP(&opts.CustomLabels, "label", "l", nil, "Add custom labels to deployed objects. Set multiple times for multiple labels.")
}

func AddRunDevFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&opts.EnableRPC, "enable-rpc", false, "Enable gRPC for exposing Skaffold events (true by default for `skaffold dev`)")
	cmd.Flags().IntVar(&opts.RPCPort, "rpc-port", constants.DefaultRPCPort, "tcp port to expose event API")
	cmd.Flags().IntVar(&opts.RPCHTTPPort, "rpc-http-port", constants.DefaultRPCHTTPPort, "tcp port to expose event REST API over HTTP")
	cmd.Flags().StringVarP(&opts.ConfigurationFile, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	cmd.Flags().BoolVar(&opts.Notification, "toot", false, "Emit a terminal beep after the deploy is complete")
	cmd.Flags().StringArrayVarP(&opts.Profiles, "profile", "p", nil, "Activate profiles by name")
	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "", "Run deployments in the specified namespace")
	cmd.Flags().StringVarP(&opts.DefaultRepo, "default-repo", "d", "", "Default repository value (overrides global config)")
	cmd.Flags().BoolVar(&opts.SkipTests, "skip-tests", false, "Whether to skip the tests after building")
	cmd.Flags().BoolVar(&opts.CacheArtifacts, "cache-artifacts", false, "Set to true to enable caching of artifacts.")
	cmd.Flags().StringVarP(&opts.CacheFile, "cache-file", "", "", "Specify the location of the cache file (default $HOME/.skaffold/cache)")
	cmd.Flags().BoolVar(&opts.NoPrune, "no-prune", false, "Skip removing images and containers built by Skaffold")
	cmd.Flags().StringArrayVar(&opts.InsecureRegistries, "insecure-registry", nil, "Target registries for built images which are not secure")
}

func AddDevDebugFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&opts.TailDev, "tail", true, "Stream logs from deployed objects")
	cmd.Flags().BoolVar(&opts.Cleanup, "cleanup", true, "Delete deployments after dev mode is interrupted")
	cmd.Flags().BoolVar(&opts.PortForward, "port-forward", true, "Port-forward exposed container ports within pods")
	cmd.Flags().StringArrayVarP(&opts.CustomLabels, "label", "l", nil, "Add custom labels to deployed objects. Set multiple times for multiple labels")
}

func SetUpLogs(out io.Writer, level string) error {
	logrus.SetOutput(out)
	lvl, err := logrus.ParseLevel(v)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	logrus.SetLevel(lvl)
	return nil
}
