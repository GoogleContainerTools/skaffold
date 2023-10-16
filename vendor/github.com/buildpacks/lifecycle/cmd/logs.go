package cmd

import (
	"os"

	"github.com/heroku/color"

	"github.com/buildpacks/lifecycle/log"
)

func init() {
	// Uncomment when https://github.com/buildpacks/pack/issues/493 (lifecycle containers with a tty) is implemented
	// color.Disable(!terminal.IsTerminal(int(os.Stdout.Fd())))
}

var (
	DefaultLogger = log.NewDefaultLogger(Stdout)
	Stdout        = color.NewConsole(os.Stdout)
	Stderr        = color.NewConsole(os.Stderr)
)

func DisableColor(noColor bool) {
	Stdout.DisableColors(noColor)
	Stderr.DisableColors(noColor)
}
