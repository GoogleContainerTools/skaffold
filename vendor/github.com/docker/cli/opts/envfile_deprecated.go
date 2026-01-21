package opts

import (
	"os"

	"github.com/docker/cli/pkg/kvfile"
)

// ParseEnvFile reads a file with environment variables enumerated by lines
//
// Deprecated: use [kvfile.Parse] and pass [os.LookupEnv] to lookup env-vars from the current environment.
func ParseEnvFile(filename string) ([]string, error) {
	return kvfile.Parse(filename, os.LookupEnv)
}
