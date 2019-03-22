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
	"github.com/hashicorp/go-plugin"
	"io"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner"
	"github.com/pkg/errors"
	"github.com/rivo/tview"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCmdDev describes the CLI command to run a pipeline in development mode.
func NewCmdDev(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Runs a pipeline file in development mode",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Command = "dev"
			return dev(out, opts.ExperimentalGUI)
		},
	}
	AddRunDevFlags(cmd)
	cmd.Flags().BoolVar(&opts.TailDev, "tail", true, "Stream logs from deployed objects")
	cmd.Flags().StringVar(&opts.Trigger, "trigger", "polling", "How are changes detected? (polling, manual or notify)")
	cmd.Flags().BoolVar(&opts.Cleanup, "cleanup", true, "Delete deployments after dev mode is interrupted")
	cmd.Flags().StringArrayVarP(&opts.TargetImages, "watch-image", "w", nil, "Choose which artifacts to watch. Artifacts with image names that contain the expression will be watched only. Default is to watch sources for all artifacts")
	cmd.Flags().IntVarP(&opts.WatchPollInterval, "watch-poll-interval", "i", 1000, "Interval (in ms) between two checks for file changes")
	cmd.Flags().BoolVar(&opts.PortForward, "port-forward", true, "Port-forward exposed container ports within pods")
	cmd.Flags().StringArrayVarP(&opts.CustomLabels, "label", "l", nil, "Add custom labels to deployed objects. Set multiple times for multiple labels")
	cmd.Flags().BoolVar(&opts.ExperimentalGUI, "experimental-gui", false, "Experimental Graphical User Interface")

	return cmd
}

func dev(out io.Writer, ui bool) error {
	opts.EnableRPC = true
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if !ui {
		catchCtrlC(cancel)
	}

	cleanup := func() {}
	if opts.Cleanup {
		defer func() {
			cleanup()
		}()
	}

	var (
		app    *tview.Application
		output *config.Output
	)
	if ui {
		app, output = createApp()
		defer app.Stop()

		go func() {
			app.Run()
			cancel()
		}()
	} else {
		output = &config.Output{
			Main: out,
			Logs: out,
		}
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := func() error {
				r, config, err := newRunner(opts)
				if err != nil {
					return errors.Wrap(err, "creating runner")
				}

				err = r.Dev(ctx, output, config.Build.Artifacts)

				defer func() {
					if err := r.RPCServerShutdown(); err != nil {
						logrus.Error(err)
					}
				}()
				defer plugin.CleanupClients()
				if r.HasDeployed() {
					cleanup = func() {
						if err := r.Cleanup(context.Background(), out); err != nil {
							logrus.Warnln("cleanup:", err)
						}
					}
				}
				if err != nil {
					if errors.Cause(err) != runner.ErrorConfigurationChanged {
						return err
					}
				}
				return nil
			}(); err != nil {
				return err
			}
		}
	}
}

func createApp() (*tview.Application, *config.Output) {
	app := tview.NewApplication()

	mainView := tview.NewTextView()
	mainView.
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("Build")

	logsView := tview.NewTextView()
	logsView.
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle("Logs")

	grid := tview.NewGrid()
	grid.
		SetRows(0, 0).
		SetColumns(0).
		SetBorders(false).
		AddItem(mainView, 0, 0, 1, 1, 0, 0, false).
		AddItem(logsView, 1, 0, 1, 1, 0, 0, false)

	app.
		SetRoot(grid, true).
		SetFocus(grid)

	output := &config.Output{
		Main: color.ColoredWriter{Writer: ansiWriter(mainView)},
		Logs: color.ColoredWriter{Writer: ansiWriter(logsView)},
	}

	return app, output
}

func ansiWriter(writer io.Writer) io.Writer {
	return &ansi{
		Writer: writer,
		replacer: strings.NewReplacer(
			"\033[31m", "[maroon]",
			"\033[32m", "[green]",
			"\033[33m", "[olive]",
			"\033[34m", "[navy]",
			"\033[35m", "[purple]",
			"\033[36m", "[teal]",
			"\033[37m", "[silver]",

			"\033[91m", "[red]",
			"\033[92m", "[lime]",
			"\033[93m", "[yellow]",
			"\033[94m", "[blue]",
			"\033[95m", "[fuchsia]",
			"\033[96m", "[aqua]",
			"\033[97m", "[white]",

			"\033[0m", "",
		),
	}
}

type ansi struct {
	io.Writer
	replacer *strings.Replacer
}

func (a *ansi) Write(text []byte) (int, error) {
	return a.replacer.WriteString(a.Writer, string(text))
}
