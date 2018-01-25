package app

import (
	"os"

	"github.com/GoogleCloudPlatform/skaffold/cmd/skaffold/app/cmd"
)

func Run() error {
	c := cmd.NewSkaffoldCommand(os.Stdout, os.Stderr)
	return c.Execute()
}
