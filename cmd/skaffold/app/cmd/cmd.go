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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	event "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/instrumentation/prompt"
	kubectx "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
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
	timestamps        bool
	shutdownAPIServer func() error

	// for testing
	updateCheck = update.CheckVersion
)

// Annotation for commands that should allow post execution housekeeping messages like updates and surveys
const (
	HouseKeepingMessagesAllowedAnnotation = "skaffold_annotation_housekeeping_allowed"
)

func NewSkaffoldCommand(out, errOut io.Writer) *cobra.Command {
	updateMsg := make(chan string, 1)
	surveyPrompt := make(chan string, 1)
	var metricsPrompt bool
	var s *survey.Runner

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

			opts.Command = cmd.Name()
			// Don't redirect output for Cobra internal `__complete` and `__completeNoDesc` commands.
			// These are used for command completion and send debug messages on stderr.
			if cmd.Name() != cobra.ShellCompRequestCmd && cmd.Name() != cobra.ShellCompNoDescRequestCmd {
				instrumentation.SetCommand(cmd.Name())
				out := output.GetWriter(out, defaultColor, forceColors, timestamps)
				cmd.Root().SetOutput(out)

				// Setup logs
				if err := setUpLogs(errOut, v, timestamps); err != nil {
					return err
				}
			}

			// Setup kubeContext and kubeConfig
			kubectx.ConfigureKubeConfig(opts.KubeConfig, opts.KubeContext)

			// Start API Server
			shutdown, err := server.Initialize(opts)
			if err != nil {
				return fmt.Errorf("initializing api server: %w", err)
			}
			shutdownAPIServer = shutdown

			// Print version
			versionInfo := version.Get()
			version.SetClient(opts.User)
			logrus.Infof("Skaffold %+v", versionInfo)
			if !isHouseKeepingMessagesAllowed(cmd) {
				logrus.Debugf("Disable housekeeping messages for command explicitly")
				return nil
			}
			s = survey.New(opts.GlobalConfig, opts.ConfigurationFile, opts.Command)
			// Always perform all checks.
			go func() {
				updateMsg <- updateCheckForReleasedVersionsIfNotDisabled(versionInfo.Version)
				surveyPrompt <- s.NextSurveyID()
			}()
			metricsPrompt = prompt.ShouldDisplayMetricsPrompt(opts.GlobalConfig)
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if isQuietMode() && !isHouseKeepingMessagesAllowed(cmd) {
				return
			}
			select {
			case msg := <-updateMsg:
				if err := config.UpdateMsgDisplayed(opts.GlobalConfig); err != nil {
					logrus.Debugf("could not update the 'last-prompted' config for 'update-config' section due to %s", err)
				}
				fmt.Fprintf(cmd.OutOrStderr(), "%s\n", msg)
			default:
			}
			// check if survey prompt needs to be displayed
			select {
			case promptSurveyID := <-surveyPrompt:
				if promptSurveyID != "" {
					if err := s.DisplaySurveyPrompt(cmd.OutOrStdout(), promptSurveyID); err != nil {
						fmt.Fprintf(cmd.OutOrStderr(), "%v\n", err)
					}
				}
			default:
			}
			if metricsPrompt {
				if err := prompt.DisplayMetricsPrompt(opts.GlobalConfig, cmd.OutOrStdout()); err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "%v\n", err)
				}
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
				NewCmdTest(),
				NewCmdDeploy(),
				NewCmdDelete(),
				NewCmdRender(),
				NewCmdApply(),
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
	rootCmd.AddCommand(NewCmdFilter())

	rootCmd.AddCommand(NewCmdGeneratePipeline())
	rootCmd.AddCommand(NewCmdSurvey())
	rootCmd.AddCommand(NewCmdInspect())

	templates.ActsAsRootCommand(rootCmd, nil, groups...)
	rootCmd.PersistentFlags().StringVarP(&v, "verbosity", "v", constants.DefaultLogLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().IntVar(&defaultColor, "color", int(output.DefaultColorCode), "Specify the default output color in ANSI escape codes")
	rootCmd.PersistentFlags().BoolVar(&forceColors, "force-colors", false, "Always print color codes (hidden)")
	rootCmd.PersistentFlags().BoolVar(&interactive, "interactive", true, "Allow user prompts for more information")
	rootCmd.PersistentFlags().BoolVar(&update.EnableCheck, "update-check", true, "Check for a more recent version of Skaffold")
	rootCmd.PersistentFlags().BoolVar(&timestamps, "timestamps", false, "Print timestamps in logs.")
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
	return fmt.Sprintf("SKAFFOLD_%s", strings.ReplaceAll(strings.ToUpper(f.Name), "-", "_"))
}

func setUpLogs(stdErr io.Writer, level string, timestamp bool) error {
	logrus.SetOutput(stdErr)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}
	logrus.SetLevel(lvl)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: timestamp,
	})
	logrus.AddHook(event.NewLogHook())
	return nil
}

// alwaysSucceedWhenCancelled returns nil if the context was cancelled.
// If the error is due to cancellation, return it as it gets swallowed
// in skaffold main.
// For all other errors, pass through known errors.
// TODO: Return nil if error is `context.Cancelled` and remove check in main.
func alwaysSucceedWhenCancelled(ctx context.Context, runCtx *runcontext.RunContext, err error) error {
	if err == nil {
		return err
	}
	// if the context was cancelled act as if all is well
	if ctx.Err() == context.Canceled {
		return nil
	} else if err == context.Canceled {
		return err
	}
	return sErrors.ShowAIError(runCtx, err)
}

func isHouseKeepingMessagesAllowed(cmd *cobra.Command) bool {
	if cmd.Annotations == nil {
		return false
	}
	return cmd.Annotations[HouseKeepingMessagesAllowedAnnotation] == fmt.Sprintf("%t", true)
}

func allowHouseKeepingMessages(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}
	cmd.Annotations[HouseKeepingMessagesAllowedAnnotation] = fmt.Sprintf("%t", true)
}

func preReleaseVersion(s string) bool {
	if v, err := version.ParseVersion(s); err == nil && len(v.Pre) > 0 {
		return true
	} else if err != nil {
		return true
	}
	return false
}

func isQuietMode() bool {
	switch {
	case !interactive:
		logrus.Debug("Update check prompt, survey prompt and telemetry prompt disabled in non-interactive mode")
		return true
	case quietFlag:
		logrus.Debug("Update check prompt, survey prompt and telemetry prompt disabled in quiet mode")
		return true
	case analyze:
		logrus.Debug("Update check prompt, survey prompt and telemetry prompt disabled when running `init --analyze`")
		return true
	default:
		return false
	}
}

func updateCheckForReleasedVersionsIfNotDisabled(s string) string {
	if preReleaseVersion(s) {
		logrus.Debug("Skipping update check for pre-release version")
		return ""
	}
	if !update.EnableCheck {
		logrus.Debug("Skipping update check for flag `--update-check` set to false")
		return ""
	}
	msg, err := updateCheck(opts.GlobalConfig)
	if err != nil {
		logrus.Infof("update check failed: %s", err)
	}
	return msg
}
