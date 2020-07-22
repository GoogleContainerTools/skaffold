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
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/server"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/survey"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/update"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

var (
	opts              config.SkaffoldOptions
	v                 string
	defaultColor      int
	forceColors       bool
	overwrite         bool
	interactive       bool
	shutdownAPIServer func() error
)

func NewSkaffoldCommand(out, err io.Writer) *cobra.Command {
	updateMsg := make(chan string)
	surveyPrompt := make(chan bool)

	rootCmd := &cobra.Command{
		Use: "skaffold",
		Long: `A tool that facilitates continuous development for Kubernetes applications.

  Find more information at: https://skaffold.dev/docs/getting-started/`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.Root().SilenceUsage = true

			opts.Command = cmd.Use

			color.SetupColors(out, defaultColor, forceColors)
			cmd.Root().SetOutput(out)

			// Setup logs
			if err := setUpLogs(err, v); err != nil {
				return err
			}

			// Start API Server
			shutdown, err := server.Initialize(opts)
			if err != nil {
				return fmt.Errorf("initializing api server: %w", err)
			}
			shutdownAPIServer = shutdown

			// Print version
			version := version.Get()
			logrus.Infof("Skaffold %+v", version)

			switch {
			case !interactive:
				logrus.Debugf("Update check and survey prompt disabled in non-interactive mode")
			case quietFlag:
				logrus.Debugf("Update check and survey prompt disabled in quiet mode")
			case analyze:
				logrus.Debugf("Update check and survey prompt when running `init --analyze`")
			default:
				go func() {
					if err := updateCheck(updateMsg, opts.GlobalConfig); err != nil {
						logrus.Infof("update check failed: %s", err)
					}
					surveyPrompt <- config.ShouldDisplayPrompt(opts.GlobalConfig)
				}()
			}
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			select {
			case msg := <-updateMsg:
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", msg)
			default:
			}
			// check if survey prompt needs to be displayed
			select {
			case shouldDisplay := <-surveyPrompt:
				if shouldDisplay {
					if err := survey.New(opts.GlobalConfig).DisplaySurveyPrompt(cmd.OutOrStdout()); err != nil {
						fmt.Fprintf(cmd.OutOrStderr(), "%v\n", err)
					}
				}
			default:
			}
		},
	}

	groups := templates.CommandGroups{
		{
			Message: "End-to-end pipelines:",
			Commands: []*cobra.Command{
				NewCmdRun(),
				NewCmdDev(),
				NewCmdDebug(),
			},
		},
		{
			Message: "Pipeline building blocks for CI/CD:",
			Commands: []*cobra.Command{
				NewCmdBuild(),
				NewCmdDeploy(),
				NewCmdDelete(),
				NewCmdRender(),
			},
		},
		{
			Message: "Getting started with a new project:",
			Commands: []*cobra.Command{
				NewCmdInit(),
				NewCmdFix(),
			},
		},
	}
	groups.Add(rootCmd)

	// other commands
	rootCmd.AddCommand(NewCmdVersion())
	rootCmd.AddCommand(NewCmdCompletion())
	rootCmd.AddCommand(NewCmdConfig())
	rootCmd.AddCommand(NewCmdFindConfigs())
	rootCmd.AddCommand(NewCmdDiagnose())
	rootCmd.AddCommand(NewCmdOptions())
	rootCmd.AddCommand(NewCmdCredits())
	rootCmd.AddCommand(NewCmdSchema())

	rootCmd.AddCommand(NewCmdGeneratePipeline())
	rootCmd.AddCommand(NewCmdSurvey())

	templates.ActsAsRootCommand(rootCmd, nil, groups...)
	rootCmd.PersistentFlags().StringVarP(&v, "verbosity", "v", constants.DefaultLogLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().IntVar(&defaultColor, "color", int(color.DefaultColorCode), "Specify the default output color in ANSI escape codes")
	rootCmd.PersistentFlags().BoolVar(&forceColors, "force-colors", false, "Always print color codes (hidden)")
	rootCmd.PersistentFlags().BoolVar(&interactive, "interactive", true, "Allow user prompts for more information")
	rootCmd.PersistentFlags().BoolVar(&update.EnableCheck, "update-check", true, "Check for a more recent version of Skaffold")
	rootCmd.PersistentFlags().MarkHidden("force-colors")

	setFlagsFromEnvVariables(rootCmd)

	return rootCmd
}

func NewCmdOptions() *cobra.Command {
	cmd := &cobra.Command{
		Use: "options",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
	}
	templates.UseOptionsTemplates(cmd)

	return cmd
}

func updateCheck(ch chan string, configfile string) error {
	if !update.IsUpdateCheckEnabled(configfile) {
		logrus.Debugf("Update check not enabled, skipping.")
		return nil
	}
	latest, current, err := update.GetLatestAndCurrentVersion()
	if err != nil {
		return fmt.Errorf("get latest and current Skaffold version: %w", err)
	}
	if latest.GT(current) {
		ch <- fmt.Sprintf("There is a new version (%s) of Skaffold available. Download it from:\n  %s\n", latest, releaseURL(latest))
	}
	return nil
}

func releaseURL(v semver.Version) string {
	return fmt.Sprintf("https://github.com/GoogleContainerTools/skaffold/releases/tag/v" + v.String())
}

// Each flag can also be set with an env variable whose name starts with `SKAFFOLD_`.
func setFlagsFromEnvVariables(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		envVar := FlagToEnvVarName(f)
		if val, present := os.LookupEnv(envVar); present {
			rootCmd.PersistentFlags().Set(f.Name, val)
		}
	})
	for _, cmd := range rootCmd.Commands() {
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

func setUpLogs(stdErr io.Writer, level string) error {
	logrus.SetOutput(stdErr)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}
	logrus.SetLevel(lvl)
	return nil
}

func alwaysSucceedWhenCancelled(ctx context.Context, err error) error {
	// if the context was cancelled act as if all is well
	if err != nil && ctx.Err() == context.Canceled {
		return nil
	}
	return err
}
