/*
Copyright 2018 The Skaffold Authors

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

	cmdutil "github.com/GoogleContainerTools/skaffold/cmd/skaffold/app/cmd/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/update"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	opts      = &config.SkaffoldOptions{}
	v         string
	filename  string
	overwrite bool

	updateMsg = make(chan string)
)

var rootCmd = &cobra.Command{
	Use:   "skaffold",
	Short: "A tool that facilitates continuous development for Kubernetes applications.",
}

func NewSkaffoldCommand(out, err io.Writer) *cobra.Command {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := SetUpLogs(err, v); err != nil {
			return err
		}
		rootCmd.SilenceUsage = true
		logrus.Infof("Skaffold %+v", version.Get())
		go func() {
			if err := updateCheck(updateMsg); err != nil {
				logrus.Infof("update check failed: %s", err)
			}
		}()
		return nil
	}

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		select {
		case msg := <-updateMsg:
			fmt.Fprintf(out, "%s\n", msg)
		default:
		}
	}

	rootCmd.SilenceErrors = true
	rootCmd.AddCommand(NewCmdCompletion(out))
	rootCmd.AddCommand(NewCmdVersion(out))
	rootCmd.AddCommand(NewCmdRun(out))
	rootCmd.AddCommand(NewCmdDev(out))
	rootCmd.AddCommand(NewCmdBuild(out))
	rootCmd.AddCommand(NewCmdDeploy(out))
	rootCmd.AddCommand(NewCmdDelete(out))
	rootCmd.AddCommand(NewCmdFix(out))

	rootCmd.PersistentFlags().StringVarP(&v, "verbosity", "v", constants.DefaultLogLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	return rootCmd
}

func updateCheck(ch chan string) error {
	if !update.IsUpdateCheckEnabled() {
		logrus.Debugf("Update check not enabled, skipping.")
		return nil
	}
	current, err := version.ParseVersion(version.Get().Version)
	if err != nil {
		return errors.Wrap(err, "parsing current semver, skipping update check")
	}
	latest, err := update.GetLatestVersion(context.Background())
	if err != nil {
		return errors.Wrap(err, "getting latest version")
	}
	if latest.GT(current) {
		ch <- fmt.Sprintf("There is a new version (%s) of skaffold available. Download it at %s\n", latest, constants.LatestDownloadURL)
	}
	return nil
}

func AddDevFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&opts.Cleanup, "cleanup", true, "Delete deployments after dev mode is interrupted")
}

func AddRunDevFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&filename, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	cmd.Flags().BoolVar(&opts.Notification, "toot", false, "Emit a terminal beep after the deploy is complete")
	cmd.Flags().StringArrayVarP(&opts.Profiles, "profile", "p", nil, "Activate profiles by name")
	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "", "Run Helm deployments in the specified namespace")
}

func AddFixFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&filename, "filename", "f", "skaffold.yaml", "Filename or URL to the pipeline file")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite original config with fixed config")
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

func readConfiguration(filename string) (*config.SkaffoldConfig, error) {
	config, err := cmdutil.ParseConfig(filename)
	if err != nil {
		return nil, errors.Wrap(err, "parsing skaffold config")
	}
	err = config.ApplyProfiles(opts.Profiles)
	if err != nil {
		return nil, errors.Wrap(err, "applying profiles")
	}
	return config, nil
}
