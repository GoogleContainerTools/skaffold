package loader

import (
	"os"

	"github.com/docker/cli/pkg/kvfile"
)

// parseEnvFile reads a file with environment variables enumerated by lines
//
// “Environment variable names used by the utilities in the Shell and
// Utilities volume of [IEEE Std 1003.1-2001] consist solely of uppercase
// letters, digits, and the '_' (underscore) from the characters defined in
// Portable Character Set and do not begin with a digit. *But*, other
// characters may be permitted by an implementation; applications shall
// tolerate the presence of such names.”
//
// As of [moby-16585], it's up to application inside docker to validate or not
// environment variables, that's why we just strip leading whitespace and
// nothing more.
//
// [IEEE Std 1003.1-2001]: http://pubs.opengroup.org/onlinepubs/009695399/basedefs/xbd_chap08.html
// [moby-16585]: https://github.com/moby/moby/issues/16585
func parseEnvFile(filename string) ([]string, error) {
	return kvfile.Parse(filename, os.LookupEnv)
}
