package main

import (
	"os"

	"github.com/heroku/color"

	"github.com/buildpacks/pack/cmd"
	"github.com/buildpacks/pack/pkg/client"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/pkg/logging"
)

func main() {
	// create logger with defaults
	logger := logging.NewLogWithWriters(color.Stdout(), color.Stderr())

	rootCmd, err := cmd.NewPackCommand(logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	ctx := commands.CreateCancellableContext()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		if _, isSoftError := err.(client.SoftError); isSoftError {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
