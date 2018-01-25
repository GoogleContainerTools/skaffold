package cmd

import (
	"io"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/version"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/constants"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var v string

func NewSkaffoldCommand(out, err io.Writer) *cobra.Command {
	c := &cobra.Command{
		Use: "skaffold",
		Short: "A tool that makes the onboarding of existing applications to Kubernetes Engine simple and repeatable.		",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := SetUpLogs(err, v); err != nil {
				return err
			}
			logrus.Infof("Skaffold %s", version.GetVersion())
			return nil
		},
	}

	c.AddCommand(NewCmdVersion(out))

	c.PersistentFlags().StringVarP(&v, "verbosity", "v", constants.DefaultLogLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	return c
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
