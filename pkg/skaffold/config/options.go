package config

import "io"

// SkaffoldOptions are options that are set by command line arguments not included
// in the config file itself
type SkaffoldOptions struct {
	DevMode      bool
	Notification bool
	Output       io.Writer
}
