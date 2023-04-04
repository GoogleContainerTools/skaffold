package notmain

import (
	"flag"
	"os"

	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/observer"
	"gopkg.in/yaml.v3"
)

func main() {
	configPath := flag.String(
		"config", "config.yml", "Path to boulder-observer configuration file")
	flag.Parse()

	configYAML, err := os.ReadFile(*configPath)
	cmd.FailOnError(err, "failed to read config file")

	// Parse the YAML config file.
	var config observer.ObsConf
	err = yaml.Unmarshal(configYAML, &config)
	if err != nil {
		cmd.FailOnError(err, "failed to parse YAML config")
	}

	// Make an `Observer` object.
	observer, err := config.MakeObserver()
	if err != nil {
		cmd.FailOnError(err, "config failed validation")
	}

	// Start the `Observer` daemon.
	observer.Start()
}

func init() {
	cmd.RegisterCommand("boulder-observer", main)
}
