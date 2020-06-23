package cmd

import (
	"io/ioutil"
	"log"
	"os"
)

type Command interface {
	Init()
	Args(nargs int, args []string) error
	Privileges() error
	Exec() error
}

func Run(c Command, asSubcommand bool) {
	var (
		printVersion bool
		logLevel     string
	)

	log.SetOutput(ioutil.Discard)
	FlagVersion(&printVersion)
	FlagLogLevel(&logLevel)
	c.Init()
	if asSubcommand {
		if err := flagSet.Parse(os.Args[2:]); err != nil {
			//flagSet exits on error, we shouldn't get here
			Exit(err)
		}
	} else {
		if err := flagSet.Parse(os.Args[1:]); err != nil {
			//flagSet exits on error, we shouldn't get here
			Exit(err)
		}
	}

	if printVersion {
		ExitWithVersion()
	}
	if err := SetLogLevel(logLevel); err != nil {
		Exit(err)
	}
	if err := c.Args(flagSet.NArg(), flagSet.Args()); err != nil {
		Exit(err)
	}
	if err := c.Privileges(); err != nil {
		Exit(err)
	}
	Exit(c.Exec())
}
